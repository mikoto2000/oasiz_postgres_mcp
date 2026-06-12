package metadata

import (
	"path"
	"strings"
)

func IsSystemSchema(schema string) bool {
	return schema == "pg_catalog" ||
		schema == "information_schema" ||
		strings.HasPrefix(schema, "pg_toast") ||
		strings.HasPrefix(schema, "pg_temp_")
}

func Excluded(name string, patterns []string) bool {
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		if pattern == name {
			return true
		}
		if ok, err := path.Match(pattern, name); err == nil && ok {
			return true
		}
	}
	return false
}

func tableType(relkind string) string {
	switch relkind {
	case "v":
		return "VIEW"
	case "m":
		return "MATERIALIZED VIEW"
	default:
		return "BASE TABLE"
	}
}
