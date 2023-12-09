package env

type ShellType string

type Manager interface {
	Load([]*KV) error
	Get(key string) (string, error)
	ReShell() error
}

type KV struct {
	Key   string
	Value string
}

type ShellInfo struct {
	ShellType
	ShellPath  string
	ConfigPath string
}
