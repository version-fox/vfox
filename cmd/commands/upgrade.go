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
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/version-fox/vfox/internal"
	"github.com/version-fox/vfox/internal/util"
)

const SelfUpgradeName = "upgrade"

var Upgrade = &cli.Command{
	Name:   SelfUpgradeName,
	Usage:  "upgrade vfox to the latest version",
	Action: upgradeCmd,
}

func fetchLatestVersion(c *http.Client) (string, error) {
	resp, err := c.Get("https://github.com/version-fox/vfox/tags")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(`href="/version-fox/vfox/releases/tag/(v[0-9.]+)"`)
	if err != nil {
		return "", err
	}
	matches := re.FindAllStringSubmatch(string(body), -1)

	if len(matches) == 0 {
		return "", fmt.Errorf("Failed to fetch the version.")
	}

	latestVersion := matches[0][1]
	return latestVersion, nil
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

func generateUrls(currVersion string, tagName string) (string, string) {
	fileName := constructBinaryName(tagName)
	binURL := fmt.Sprintf("https://github.com/version-fox/vfox/releases/download/%s/%s", tagName, fileName)
	diffURL := fmt.Sprintf("https://github.com/version-fox/vfox/compare/%s...%s", currVersion, tagName)
	return binURL, diffURL
}

func downloadFile(c *http.Client, filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := c.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func upgradeCmd(ctx context.Context, cmd *cli.Command) error {
	manager := internal.NewSdkManager()
	defer manager.Close()
	httpClient := manager.HttpClient()

	currVersion := fmt.Sprintf("v%s", internal.RuntimeVersion)
	latestVersion, err := fetchLatestVersion(httpClient)
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
	exePath, err := os.Executable()
	if err != nil {
		return cli.Exit("Failed to get executable path: "+err.Error(), 1)
	}
	exeDir, exeName := filepath.Split(exePath)
	binURL, diffURL := generateUrls(currVersion, latestVersion)
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

	fmt.Println("Fetching", binURL)

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
		cmd := exec.Command("cmd.exe", "/C", batchFile)
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

	fmt.Printf("Updated to version: %s\nSee the diff at: %s\n", latestVersion, diffURL)

	if runtime.GOOS == "windows" {
		fmt.Println("Press any key to continue...")
		var b = make([]byte, 1)
		_, _ = os.Stdin.Read(b)
	}
	return nil
}
