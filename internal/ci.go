package internal

import (
	"os"
	"strings"
)

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
