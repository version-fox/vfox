package internal

import (
	"fmt"
	"os"
	"strings"
)

// CICommandEvent describes a type of interaction that may be handled specially in CI.
type CICommandEvent string

const (
	// CIEventMissingVersion indicates an install-like command was invoked without a version.
	CIEventMissingVersion CICommandEvent = "missing_version"
)

// CICommandAction describes how a command should behave in CI for a particular event.
type CICommandAction struct {
	MessageTemplate string
	ExitCode        int
}

// FormatMessage renders the action message with provided values.
func (a CICommandAction) FormatMessage(values ...interface{}) string {
	if len(values) == 0 {
		return a.MessageTemplate
	}
	return fmt.Sprintf(a.MessageTemplate, values...)
}

// ResolveCICommand resolves the configured CI action for a command/event pair and
// formats the resulting message/exit code. It automatically no-ops outside CI.
func ResolveCICommand(command string, event CICommandEvent, values ...interface{}) (bool, string, int) {
	if !IsCI() {
		return false, "", 0
	}
	action, ok := LookupCICommandAction(command, event)
	if !ok || action == nil {
		return false, "", 0
	}
	return true, action.FormatMessage(values...), action.ExitCode
}

// CICommandBehavior enumerates all CI-specific handlers for a command.
type CICommandBehavior struct {
	MissingVersion *CICommandAction
}

// CICommandBehaviors centralizes CI handling configuration per command.
var CICommandBehaviors = map[string]CICommandBehavior{ // nolint:gochecknoglobals
	"install": {
		MissingVersion: &CICommandAction{
			MessageTemplate: "CI mode: install requires specifying a version for %s",
			ExitCode:        1,
		},
	},
}

// LookupCICommandAction fetches the CI action for a given command/event combination.
func LookupCICommandAction(command string, event CICommandEvent) (*CICommandAction, bool) {
	behavior, ok := CICommandBehaviors[command]
	if !ok {
		return nil, false
	}
	switch event {
	case CIEventMissingVersion:
		if behavior.MissingVersion != nil {
			return behavior.MissingVersion, true
		}
	}
	return nil, false
}

var (
	ciTruthyEnvVars = []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
	}

	ciPresenceEnvVars = []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"BUILDKITE",
		"TF_BUILD",
		"TEAMCITY_VERSION",
		"TRAVIS",
		"CIRCLECI",
		"APPVEYOR",
		"BITBUCKET_BUILD_NUMBER",
		"JENKINS_URL",
		"DRONE",
		"HUDSON_URL",
		"GO_SERVER_URL",
		"CODEBUILD_BUILD_ID",
	}
)

// IsCI checks if the current environment is CI.
func IsCI() bool {
	for _, key := range ciTruthyEnvVars {
		if isTruthyEnv(os.Getenv(key)) {
			return true
		}
	}

	for _, key := range ciPresenceEnvVars {
		if os.Getenv(key) != "" {
			return true
		}
	}

	return false
}

func isTruthyEnv(value string) bool {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch normalized {
	case "", "0", "false", "no", "off":
		return false
	}
	return true
}

// CIConfirm returns the default confirmation value for CI environments.
func CIConfirm() bool {
	return true
}

// CISelect returns the default selection for CI environments (first option).
func CISelect(options []string) string {
	if len(options) > 0 {
		return options[0]
	}
	return ""
}
