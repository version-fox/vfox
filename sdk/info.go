package sdk

type Version string

type Source interface {
	Install(version Version) error
	Uninstall(version Version) error
	Search(version Version) error
	Use(version Version) error
	List() []Version
	Current() Version
	Name() string
}
