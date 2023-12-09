package util

import "runtime"

type ArchType string

const (
	AMD64 ArchType = "amd64"
	ARM64 ArchType = "arm64"
)

func GetArchType() ArchType {
	switch runtime.GOARCH {
	case "amd64":
		return AMD64
	case "arm64":
		return ARM64
	default:
		return ArchType(runtime.GOARCH)
	}
}
