package util

import "runtime"

type OSType string

const (
	MacOS   OSType = "darwin"
	Windows OSType = "windows"
	Linux   OSType = "linux"
)

func GetOSType() OSType {
	switch runtime.GOOS {
	case "darwin":
		return MacOS
	case "windows":
		return Windows
	case "linux":
		return Linux
	default:
		return OSType(runtime.GOOS)
	}
}
