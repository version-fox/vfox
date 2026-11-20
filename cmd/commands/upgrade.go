/*
 *    Copyright 2026 Han Li and contributors
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/config"
	"github.com/version-fox/vfox/internal/shared/util"
)

const (
	SelfUpgradeName         = "upgrade"
	maxReleaseResponseBytes = 1 << 20
	defaultGitHubAPIBase    = "https://api.github.com"
)

var releaseTagPattern = regexp.MustCompile(`^v[0-9.]+$`)

var Upgrade = &cli.Command{
	Name:   SelfUpgradeName,
	Usage:  "upgrade vfox to the latest version",
	Action: upgradeCmd,
}

func fetchLatestVersion(c *http.Client, github *config.GitHub) (string, error) {
	apiBase := strings.TrimSuffix(github.APIBase(), "/")
	releaseURL := apiBase + "/repos/version-fox/vfox/releases/latest"
	req, err := http.NewRequest(http.MethodGet, releaseURL, nil)
	if err != nil {
		return "", util.RedactError(err)
	}
	if token := os.Getenv("GITHUB_TOKEN"); token != "" && apiBase == defaultGitHubAPIBase {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := c.Do(req)
	if err != nil {
		return "", util.RedactError(err)
	}
	defer resp.Body.Close()

	if err = util.UnexpectedStatusError(resp, releaseURL); err != nil {
		return "", err
	}

	var release struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReleaseResponseBytes)).Decode(&release); err != nil {
		return "", err
	}
	if release.Tag == "" {
		return "", fmt.Errorf("latest release has no tag_name")
	}
	if !releaseTagPattern.MatchString(release.Tag) {
		return "", fmt.Errorf("invalid release tag %q", release.Tag)
	}
	return release.Tag, nil
}

func constructBinaryName(tagName string) string {
	osType := strings.ToLower(runtime.GOOS)
	if osType == "darwin" {
		osType = "macos"
	}

	archType := runtime.GOARCH
	if archType == "arm64" {
		archType = "aarch64"
	}
	if archType == "amd64" {
		archType = "x86_64"
	}

	extName := "tar.gz"
	if osType == "windows" {
		extName = "zip"
	}

	fileName := fmt.Sprintf("vfox_%s_%s_%s.%s", tagName[1:], osType, archType, extName)
	return fileName
}

func generateUrls(currVersion string, tagName string, github *config.GitHub) (string, string) {
	fileName := constructBinaryName(tagName)
	binURL := github.WebBase() + fmt.Sprintf("/version-fox/vfox/releases/download/%s/%s", tagName, fileName)
	diffURL := github.WebBase() + fmt.Sprintf("/version-fox/vfox/compare/%s...%s", currVersion, tagName)
	return binURL, diffURL
}

func downloadFile(c *http.Client, filepath string, url string) error {
	resp, err := c.Get(url)
	if err != nil {
		return util.RedactError(err)
	}
	defer resp.Body.Close()

	if err = util.UnexpectedStatusError(resp, url); err != nil {
		return err
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func upgradeCmd(ctx context.Context, cmd *cli.Command) error {
	exePath, err := os.Executable()
	if err != nil {
		return cli.Exit("Failed to get executable path: "+err.Error(), 1)
	}
	// An MSIX-installed vfox lives in the read-only WindowsApps directory;
	// self-upgrading cannot replace the binary there.
	if runtime.GOOS == "windows" && strings.Contains(strings.ToLower(exePath), `\windowsapps\`) {
		return cli.Exit("vfox was installed as an MSIX package and cannot self-upgrade. Install a newer .msixbundle to upgrade.", 1)
	}
	manager, err := internal.NewSdkManager()
	if err != nil {
		return err
	}
	defer manager.Close()
	httpClient, err := manager.RuntimeEnvContext.HttpClient()
	if err != nil {
		return cli.Exit("Failed to create the HTTP client: "+err.Error(), 1)
	}

	currVersion := fmt.Sprintf("v%s", internal.RuntimeVersion)
	github := manager.RuntimeEnvContext.UserConfig.GitHub
	latestVersion, err := fetchLatestVersion(httpClient, github)
	if err != nil {
		return cli.Exit("Failed to fetch the latest version: "+err.Error(), 1)
	}
	fmt.Println("Current version: ", currVersion)
	fmt.Println("Latest available:", latestVersion)
	if currVersion == latestVersion {
		return cli.Exit("vfox is already up to date.", 0)
	}
	if err = RequestPermission(); err != nil {
		return err
	}
	exeDir, exeName := filepath.Split(exePath)
	binURL, diffURL := generateUrls(currVersion, latestVersion, github)
	tempFile := "vfox_latest.tar.gz"
	if runtime.GOOS == "windows" {
		tempFile = "vfox_latest.zip"
	}
	tempDir := filepath.Join(exeDir, "vfox_upgrade")
	tempFile = filepath.Join(tempDir, tempFile)

	// create tempDir if not exist
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		if err := os.Mkdir(tempDir, 0755); err != nil {
			return cli.Exit("Failed to create directory: "+err.Error(), 1)
		}
	} else if err != nil {
		return cli.Exit("Error checking directory: "+err.Error(), 1)
	} else {
		fmt.Println("Directory already exists")
	}

	defer func() {
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Println("Error removing directory: ", err)
		}
	}()

	fmt.Println("Fetching", util.RedactSourceURL(binURL))

	if err := downloadFile(httpClient, tempFile, binURL); err != nil {
		return cli.Exit("Failed to download file: "+err.Error(), 1)
	}
	decompressor := util.NewDecompressor(tempFile)
	if err := decompressor.Decompress(tempDir); err != nil {
		return cli.Exit("Failed to extract file: "+err.Error(), 1)
	}
	tempExePath := filepath.Join(tempDir, exeName)
	if _, err := os.Stat(tempExePath); err != nil {
		return cli.Exit("Failed to find valid executable: "+err.Error(), 1)
	}

	if runtime.GOOS == "windows" {
		backupExePath := filepath.Join(exeDir, "."+exeName)
		batchFile := filepath.Join(exeDir, ".upgrade.bat")
		if err := os.Rename(exePath, backupExePath); err != nil {
			return cli.Exit("Failed to backup: "+err.Error(), 1)
		}
		if err := os.Rename(tempExePath, exePath); err != nil {
			os.Rename(backupExePath, exePath)
			return cli.Exit("Failed to replace executable: "+err.Error(), 1)
		}
		batchContent := fmt.Sprintf(":Repeat\n"+
			"del \"%s\"\n"+
			"if exist \"%s\" goto Repeat\n"+
			"del \"%s\"", backupExePath, backupExePath, batchFile)
		if err := os.WriteFile(batchFile, []byte(batchContent), 0666); err != nil {
			return cli.Exit("Failed to clear: "+err.Error(), 1)
		}
		cmd := util.PowerShellCommand(fmt.Sprintf("& '%s'", strings.ReplaceAll(batchFile, "'", "''")))
		if err := cmd.Start(); err != nil {
			return cli.Exit("Failed to launch shell: "+err.Error(), 1)
		}
	} else {
		if err := os.Rename(tempExePath, exePath); err != nil {
			return cli.Exit("Failed to replace executable: "+err.Error(), 1)
		}
		if err := os.Chmod(exePath, 0755); err != nil {
			return cli.Exit("Failed to make executable: "+err.Error(), 1)
		}
	}

	fmt.Printf("Updated to version: %s\nSee the diff at: %s\n", latestVersion, util.RedactSourceURL(diffURL))

	if runtime.GOOS == "windows" {
		fmt.Println("Press any key to continue...")
		var b = make([]byte, 1)
		_, _ = os.Stdin.Read(b)
	}
	return nil
}
