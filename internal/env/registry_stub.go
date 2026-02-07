//go:build !windows

package env

// ApplyEnvsToRegistry is a no-op outside Windows.
func ApplyEnvsToRegistry(envs *Envs) error {
	return nil
}

// RemoveEnvsFromRegistry is a no-op outside Windows.
func RemoveEnvsFromRegistry(envs *Envs) error {
	return nil
}
