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

// rewriteInstallRootInRegistry is a no-op outside Windows.
func rewriteInstallRootInRegistry(oldRoot, newRoot string) error {
	return nil
}
