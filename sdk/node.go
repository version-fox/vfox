package sdk

import (
	"fmt"
	"github.com/aooohan/version-fox/env"
	"github.com/aooohan/version-fox/util"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const nodeDownloadUrl = "https://nodejs.org/dist/v%s/node-v%s-%s-%s.tar.gz"

type NodeSource struct {
	operation *Operation
	envManger env.Manager
}

func (n *NodeSource) Name() string {
	return "Node"
}

func (n *NodeSource) Install(version Version) error {
	if n.checkExists(version) {
		fmt.Printf("node@%s has been installed, no need to install it.\n", version)
		return fmt.Errorf("node@%s has been installed, no need to install it.\n", version)
	}
	archType := n.operation.archType
	if n.operation.archType == "amd64" {
		archType = "x64"
	}
	urlStr := fmt.Sprintf(nodeDownloadUrl, version, version, n.operation.osType, archType)
	u, _ := url.Parse(urlStr)
	filePath, err := n.operation.Download(u)
	if err != nil {
		println(fmt.Errorf("failed to download file, err:%s", err))
		return err
	}
	fileName := strings.TrimSuffix(filepath.Base(filePath), ".tar.gz")
	destPath := filepath.Dir(filePath)
	//destPath := strings.TrimSuffix(filePath, ".tar.gz")
	err = util.DecompressGzipTar(filePath, destPath)
	if err != nil {
		return err
	}
	newDirPath := n.versionPath(version)
	err = os.Rename(filepath.Join(destPath, fileName), newDirPath)
	if err != nil {
		return err
	}
	fmt.Printf("install node@%s success!\n", version)
	// del cache file
	_ = os.Remove(filePath)
	return nil
}

func (n *NodeSource) Uninstall(version Version) error {
	if !n.checkExists(version) {
		fmt.Printf("node@%s is not installed, no need to uninstall it.\n", version)
		return fmt.Errorf("node@%s is not installed, no need to uninstall it.\n", version)
	}
	err := os.RemoveAll(n.versionPath(version))
	if err != nil {
		return err
	}
	fmt.Printf("Uninstall node@%s success!\n", version)
	remainVersion := n.List()
	if len(remainVersion) == 0 {
		_ = os.RemoveAll(n.operation.localPath)
	}
	firstVersion := remainVersion[0]
	return n.Use(firstVersion)
}

func (n *NodeSource) Search(version Version) error {
	//TODO implement me
	panic("implement me")
}

func (n *NodeSource) Use(version Version) error {
	if !n.checkExists(version) {
		fmt.Printf("node@%s is not installed, please install it first.\n", version)
		return fmt.Errorf("node@%s is not installed, please install it first.\n", version)
	}
	err := n.envManger.Load([]*env.KV{
		{
			"NODE_VERSION",
			string(version),
		},
		{
			"PATH",
			n.versionPath(version),
		},
	})
	if err != nil {
		return fmt.Errorf("use node@%s error, err: %s\n", version, err)
	}
	fmt.Printf("Now using node@%s \n", version)
	return n.envManger.ReShell()
}

func (n *NodeSource) List() []Version {
	if !util.FileExists(n.operation.localPath) {
		return make([]Version, 0)
	}
	var versions []Version
	err := filepath.Walk(n.operation.localPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() && strings.HasPrefix(info.Name(), "v-") {
			versions = append(versions, Version(strings.TrimPrefix(info.Name(), "v-")))
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return versions
}

func (n *NodeSource) Current() Version {
	value, _ := n.envManger.Get("NODE_VERSION")
	return Version(value)
}

func (n *NodeSource) checkExists(version Version) bool {
	return util.FileExists(n.versionPath(version))
}

func (n *NodeSource) versionPath(version Version) string {
	return filepath.Join(n.operation.localPath, fmt.Sprintf("v-%s", version))
}

func NewNodeSource(manager *Manager) (Source, error) {
	operation := &Operation{
		localPath:    filepath.Join(manager.sdkCachePath, "node"),
		vfConfigPath: manager.configPath,
		osType:       manager.osType,
		archType:     manager.archType,
	}
	envManger, err := env.NewEnvManager(manager.configPath, manager.sdkCachePath, "node")
	if err != nil {
		return nil, err
	}
	return &NodeSource{
		operation: operation,
		envManger: envManger,
	}, nil
}
