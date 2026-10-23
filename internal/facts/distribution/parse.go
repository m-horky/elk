package distribution

import (
	"bytes"
	"fmt"
	"log/slog"
	"strings"

	"github.com/hashicorp/go-envparse"
)

// cpePrefix is the strict CPE name prefix accepted for version modifiers.
const cpePrefix = "cpe:/o:"

// cpeVersionModifierIndex is the zero-based component index that holds the
// version modifier in an operating system CPE name.
const cpeVersionModifierIndex = 5

// cpeComponentCount is the minimum number of components a CPE name must have
// for the version modifier component to exist.
const cpeComponentCount = cpeVersionModifierIndex + 1

// parse converts os-release contents into distribution facts. It returns an
// error when the input cannot be parsed as a key/value file.
func parse(data []byte) (Distribution, error) {
	raw, err := envparse.Parse(bytes.NewReader(data))
	if err != nil {
		return Distribution{}, fmt.Errorf("parse os-release: %w", err)
	}

	distribution := Distribution{
		ID:      lookup(raw, "ID"),
		Name:    lookup(raw, "NAME"),
		Version: lookup(raw, "VERSION_ID"),
	}

	if cpeName, ok := raw["CPE_NAME"]; ok {
		distribution.VersionModifier = parseVersionModifier(cpeName)
	}

	return distribution, nil
}

// lookup returns a pointer to the value stored under key, or nil when the key
// is absent.
func lookup(raw map[string]string, key string) *string {
	value, ok := raw[key]
	if !ok {
		return nil
	}

	return &value
}

// parseVersionModifier extracts the lower-cased version modifier from a CPE
// name. Bad CPE names are logged at debug level and produce nil.
func parseVersionModifier(cpeName string) *string {
	if cpeName == "" {
		return nil
	}

	if !strings.HasPrefix(cpeName, cpePrefix) {
		slog.Debug("ignoring unsupported CPE name", "cpe_name", cpeName)

		return nil
	}

	components, ok := splitCPE(cpeName)
	if !ok || len(components) < cpeComponentCount {
		slog.Debug("ignoring malformed CPE name", "cpe_name", cpeName)

		return nil
	}

	modifier := components[cpeVersionModifierIndex]
	if modifier == "" {
		return nil
	}

	modifier = strings.ToLower(modifier)
	modifier = strings.ReplaceAll(modifier, `\:`, ":")

	return &modifier
}

// splitCPE splits a CPE name on unescaped colons while preserving escaped
// colons in the returned components. It reports false when the value ends with
// a dangling escape.
func splitCPE(value string) ([]string, bool) {
	components := make([]string, 0, cpeComponentCount)

	var component strings.Builder

	escaped := false

	for i := 0; i < len(value); i++ {
		char := value[i]

		if escaped {
			component.WriteByte('\\')
			component.WriteByte(char)

			escaped = false

			continue
		}

		switch char {
		case '\\':
			escaped = true
		case ':':
			components = append(components, component.String())
			component.Reset()
		default:
			component.WriteByte(char)
		}
	}

	if escaped {
		return nil, false
	}

	components = append(components, component.String())

	return components, true
}
