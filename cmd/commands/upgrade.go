/*
 *    Copyright 2024 Han Li and contributors
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
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
	"github.com/version-fox/vfox/internal/util"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var Upgrade = &cli.Command{
	Name:   "upgrade",
	Usage:  "Upgrade vfox to the latest version.",
	Action: upgradeCmd,
}

type ReleaseInfo struct {
	TagName string `json:"tag_name"`
}

func getLatestReleaseInfo(apiURL string) (ReleaseInfo, error) {
	var releaseInfo ReleaseInfo

	resp, err := http.Get(apiURL)
	if err != nil {
		return releaseInfo, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return releaseInfo, err
	}

	err = json.Unmarshal(body, &releaseInfo)
	return releaseInfo, err
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

	fileName := fmt.Sprintf("vfox_%s_%s_%s.tar.gz", tagName[1:], osType, archType)
	return fileName
}

// constructDiffURL constructs a GitHub diff URL between two version tags
func constructDiffURL(oldTag, newTag string) string {
	return fmt.Sprintf("https://github.com/version-fox/vfox/compare/%s...%s", oldTag, newTag)
}

func getUrls(apiURL string, currVersion string) (string, string) {
	version := currVersion
	releaseInfo, err := getLatestReleaseInfo(apiURL)
	if err != nil {
		fmt.Println("Error fetching release info:", err)
		return "", ""
	}
	fileName := constructBinaryName(releaseInfo.TagName)
	binURL := fmt.Sprintf("https://github.com/version-fox/vfox/releases/download/%s/%s", releaseInfo.TagName, fileName)
	diffURL := constructDiffURL(version, releaseInfo.TagName)
	return binURL, diffURL
}

func downloadFile(filepath string, url string) error {
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

func extractTarGz(gzipFilePath, destDir string) error {
	decompressor := util.NewDecompressor(gzipFilePath)
	if err := decompressor.Decompress(destDir); err != nil {
		return err
	} else {
		return nil
	}
}

func upgradeCmd(ctx *cli.Context) error {
	apiURL := "https://api.github.com/repos/version-fox/vfox/releases/latest"
	releaseInfo, err := getLatestReleaseInfo(apiURL)
	currVersion := fmt.Sprintf("v%s", ctx.App.Version)
	latestVersion := releaseInfo.TagName
	fmt.Println("Current version: ", currVersion)
	fmt.Println("Latest available:", latestVersion)

	if currVersion == latestVersion {
		return cli.Exit("vfox is already at latest version.", 0)
	}
	exePath, err := os.Executable()
	if err != nil {
		return cli.Exit("Failed to get executable path: "+err.Error(), 1)
	}

	binURL, diffURL := getUrls(apiURL, currVersion)
	exeDir := filepath.Dir(exePath)
	tempFile := "latest_vfox.tar.gz"
	tempDir := filepath.Join(exeDir, "update_vfox")

	if err := downloadFile(tempFile, binURL); err != nil {
		return cli.Exit("Failed to download file: "+err.Error(), 1)
	}
	if err := extractTarGz(tempFile, tempDir); err != nil {
		return cli.Exit("Failed to extract file: "+err.Error(), 1)
	}
	newExePath := filepath.Join(tempDir, "vfox")
	if err := os.Rename(newExePath, exePath); err != nil {
		return cli.Exit("Failed to replace executable: "+err.Error(), 1)
	}
	if err := os.Chmod(exePath, 0755); err != nil {
		panic("Failed to make executable: " + err.Error())
	}
	if err = os.RemoveAll(tempDir); err != nil {
		fmt.Println("Error removing directory:", err)
	}
	if err = os.RemoveAll(tempFile); err != nil {
		fmt.Println("Error removing directory:", err)
	}

	updateMsg := fmt.Sprintf("Updated to version: %s \nSee the diff at: %s \n", latestVersion, diffURL)
	return cli.Exit(updateMsg, 0)
}
