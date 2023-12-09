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
	archType := n.operation.archType
	if n.operation.archType == "amd64" {
		archType = "x64"
	}
	urlStr := fmt.Sprintf(nodeDownloadUrl, version, version, n.operation.osType, archType)
	u, _ := url.Parse(urlStr)
	println(fmt.Sprintf("1.start downloading node@%s file", version))
	filePath, err := n.operation.Download(u)
	if err != nil {
		println(fmt.Errorf("failed to download file, err:%s", err))
		return err
	}
	fileName := strings.TrimSuffix(filepath.Base(filePath), ".tar.gz")
	destPath := filepath.Dir(filePath)
	//destPath := strings.TrimSuffix(filePath, ".tar.gz")
	println(fmt.Sprintf("2.start decompress node@%s file", version))
	err = util.DecompressGzipTar(filePath, destPath)
	if err != nil {
		return err
	}
	newDirPath := filepath.Join(n.operation.localPath, fmt.Sprintf("v%s", version))
	err = os.Rename(filepath.Join(destPath, fileName), newDirPath)
	if err != nil {
		return err
	}
	println(fmt.Sprintf("3.set env PATH variable: %s", filepath.Join(newDirPath, "bin")))
	err = n.envManger.Load([]*env.KV{
		{
			Key:   "PATH",
			Value: filepath.Join(newDirPath, "bin"),
		},
	})
	if err != nil {
		return fmt.Errorf("install node@%s error, err: %s\n", version, err)
	}
	fmt.Printf("install node@%s success!\n", version)
	// del cache file
	_ = os.Remove(filePath)
	return n.envManger.ReShell()
}

func (n *NodeSource) Uninstall(version Version) error {
	return nil
}

func (n *NodeSource) Search(version Version) error {
	//TODO implement me
	panic("implement me")
}

func (n *NodeSource) Use(version Version) error {
	//TODO implement me
	panic("implement me")
}

func (n *NodeSource) List() []Version {
	//TODO implement me
	panic("implement me")
}

func (n *NodeSource) Current() Version {
	//TODO implement me
	panic("implement me")
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
