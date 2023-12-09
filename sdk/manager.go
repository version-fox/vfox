package sdk

import (
	"fmt"
	"github.com/aooohan/version-fox/util"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Arg struct {
	Name    string
	Version string
}

type Manager struct {
	configPath    string
	sdkCachePath  string
	envConfigPath string
	sdkMap        map[string]Source
	osType        util.OSType
	archType      util.ArchType
}

func (s *Manager) Install(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	if err := source.Install(Version(config.Version)); err != nil {
		return err
	}
	exec.Command(os.Getenv("SHELL"))
	return nil
}

func (s *Manager) Uninstall(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Uninstall(Version(config.Version))
}

func (s *Manager) Search(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Search(Version(config.Version))
}

func (s *Manager) Use(config Arg) error {
	source := s.sdkMap[config.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", config.Name)
	}
	return s.sdkMap[config.Name].Use(Version(config.Version))
}

func (s *Manager) List(arg Arg) error {
	source := s.sdkMap[arg.Name]
	if source == nil {
		return fmt.Errorf("%s not supported", arg.Name)
	}
	list := source.List()
	for _, version := range list {
		println(fmt.Sprintf("-> \t  %s", version))
	}
	return nil
}

func (s *Manager) Current(sdkName string) error {
	source := s.sdkMap[sdkName]
	if source == nil {
		return fmt.Errorf("%s not supported", sdkName)
	}
	current := source.Current()
	println(fmt.Sprintf("-> \t  %s", current))
	return nil
}

func NewSdkManager() *Manager {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic("get user home dir error")
	}
	manager := &Manager{
		configPath:    filepath.Join(userHomeDir, ".version-fox"),
		sdkCachePath:  filepath.Join(userHomeDir, ".version-fox", ".cache"),
		envConfigPath: filepath.Join(userHomeDir, ".version-fox", "env.sh"),
		sdkMap:        make(map[string]Source),
		osType:        util.GetOSType(),
		archType:      util.GetArchType(),
	}
	_ = os.MkdirAll(manager.sdkCachePath, 0755)
	if !util.FileExists(manager.envConfigPath) {
		_, _ = os.Create(manager.envConfigPath)
	}

	if node, err := NewNodeSource(manager); err == nil {
		manager.sdkMap[strings.ToLower(node.Name())] = node
	}

	return manager
}
