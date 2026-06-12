package metadata

import "testing"

func TestIsSystemSchema(t *testing.T) {
	tests := map[string]bool{
		"pg_catalog":         true,
		"information_schema": true,
		"pg_toast":           true,
		"pg_toast_temp_1":    true,
		"pg_temp_3":          true,
		"public":             false,
		"app":                false,
	}

	for schema, want := range tests {
		if got := IsSystemSchema(schema); got != want {
			t.Fatalf("IsSystemSchema(%q) = %v, want %v", schema, got, want)
		}
	}
}

func TestExcluded(t *testing.T) {
	patterns := []string{"audit", "tmp_*"}
	if !Excluded("audit", patterns) {
		t.Fatal("exact match should be excluded")
	}
	if !Excluded("tmp_orders", patterns) {
		t.Fatal("glob match should be excluded")
	}
	if Excluded("orders", patterns) {
		t.Fatal("non-matching name should not be excluded")
	}
}
