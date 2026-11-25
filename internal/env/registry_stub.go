//go:build !windows

package env

// applyEnvsToRegistry is a no-op outside Windows.
func applyEnvsToRegistry(envs *Envs) error {
	return nil
}

// removeEnvsFromRegistry is a no-op outside Windows.
func removeEnvsFromRegistry(envs *Envs) error {
	return nil
}
