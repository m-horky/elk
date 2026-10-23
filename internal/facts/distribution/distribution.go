package distribution

import (
	"fmt"

	"github.com/m-horky/elk/internal/fs"
)

// maxSize is the maximum number of bytes read from an os-release file.
const maxSize = 4 * 1024 // 4 KiB

// Distribution contains the distribution facts derived from an os-release
// file. A nil field means that the corresponding key or CPE component is
// absent.
type Distribution struct {
	// ID is the lower-case distribution identifier from the ID key.
	ID *string
	// Name is the human-readable distribution name from the NAME key.
	Name *string
	// Version is the distribution version from the VERSION_ID key.
	Version *string
	// VersionModifier is the CPE version modifier derived from the CPE_NAME
	// key, such as "ga" or "beta".
	VersionModifier *string
}

// Collect reads an os-release file at path and parses its distribution facts.
// It reads at most one mebibyte and returns any read or parse error to the
// caller, which decides whether collection is best effort.
func Collect(filesystem fs.FS, path string) (Distribution, error) {
	data, err := filesystem.Read(path, maxSize)
	if err != nil {
		return Distribution{}, fmt.Errorf("read %s: %w", path, err)
	}

	return parse(data)
}
