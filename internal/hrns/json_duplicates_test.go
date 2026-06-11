package hrns

import "testing"

func TestDuplicateJSONKeysDetectsNestedDuplicate(t *testing.T) {
	dups, err := duplicateJSONKeys(`{"a":1,"b":{"c":1,"c":2},"a":3}`)
	if err != nil {
		t.Fatalf("duplicateJSONKeys returned error: %v", err)
	}
	want := map[string]bool{"b.c": true, "a": true}
	if len(dups) != len(want) {
		t.Fatalf("expected %d duplicate(s), got %v", len(want), dups)
	}
	for _, dup := range dups {
		if !want[dup] {
			t.Fatalf("unexpected duplicate path %q in %v", dup, dups)
		}
	}
}

func TestDuplicateJSONKeysRejectsInvalidJSON(t *testing.T) {
	if _, err := duplicateJSONKeys(`{"a":`); err == nil {
		t.Fatalf("expected invalid JSON error")
	}
}
