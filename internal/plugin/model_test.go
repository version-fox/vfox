package plugin

import (
	"testing"

	"github.com/version-fox/vfox/internal/shared"
)

func TestPreInstallPackageItem_Checksum(t *testing.T) {
	t.Run("Returns NoneChecksum when CheckSumItem is nil", func(t *testing.T) {
		// This is the case that was causing the nil pointer dereference
		item := &PreInstallPackageItem{
			Name:         "test-sdk",
			Version:      "1.0.0",
			Path:         "https://example.com/test.tar.gz",
			CheckSumItem: nil, // Explicitly nil
		}

		checksum := item.Checksum()

		if checksum != shared.NoneChecksum {
			t.Errorf("Expected NoneChecksum, got %v", checksum)
		}

		if checksum.Type != "none" {
			t.Errorf("Expected checksum type 'none', got '%s'", checksum.Type)
		}

		if checksum.Value != "" {
			t.Errorf("Expected empty checksum value, got '%s'", checksum.Value)
		}
	})

	t.Run("Returns checksum when CheckSumItem has sha256", func(t *testing.T) {
		item := &PreInstallPackageItem{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    "https://example.com/test.tar.gz",
			CheckSumItem: &CheckSumItem{
				Sha256: "abc123def456",
			},
		}

		checksum := item.Checksum()

		if checksum.Type != "sha256" {
			t.Errorf("Expected checksum type 'sha256', got '%s'", checksum.Type)
		}

		if checksum.Value != "abc123def456" {
			t.Errorf("Expected checksum value 'abc123def456', got '%s'", checksum.Value)
		}
	})

	t.Run("Returns checksum when CheckSumItem has md5", func(t *testing.T) {
		item := &PreInstallPackageItem{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    "https://example.com/test.tar.gz",
			CheckSumItem: &CheckSumItem{
				Md5: "xyz789",
			},
		}

		checksum := item.Checksum()

		if checksum.Type != "md5" {
			t.Errorf("Expected checksum type 'md5', got '%s'", checksum.Type)
		}

		if checksum.Value != "xyz789" {
			t.Errorf("Expected checksum value 'xyz789', got '%s'", checksum.Value)
		}
	})

	t.Run("Returns NoneChecksum when CheckSumItem has no checksums", func(t *testing.T) {
		item := &PreInstallPackageItem{
			Name:    "test-sdk",
			Version: "1.0.0",
			Path:    "https://example.com/test.tar.gz",
			CheckSumItem: &CheckSumItem{
				// All fields empty
			},
		}

		checksum := item.Checksum()

		if checksum != shared.NoneChecksum {
			t.Errorf("Expected NoneChecksum when all checksum fields are empty, got %v", checksum)
		}
	})
}
