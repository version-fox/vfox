package env

import (
	"reflect"
	"testing"
)

func TestDedupOrderedPaths(t *testing.T) {
	inputs := []string{
		"C:/Foo",
		"C:/Foo/../Foo",
		"C:/Bar",
		"c:/bar",
		"  ",
		"",
	}

	got := dedupOrderedPaths(inputs)
	want := []string{"C:/Foo", "C:/Bar"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Dedup mismatch: got %v want %v", got, want)
	}
}

func TestRemovePaths(t *testing.T) {
	existing := []string{"C:/Foo", "C:/Bar", "C:/Baz"}
	toRemove := []string{"c:/bar", "C:/foo", "  "}

	got := removePaths(existing, toRemove)
	want := []string{"C:/Baz"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Remove mismatch: got %v want %v", got, want)
	}
}

func TestSplitSemicolonSeparated(t *testing.T) {
	value := ";C:/Foo;;C:/Bar ;   C:/Baz;"

	got := splitSemicolonSeparated(value)
	want := []string{"C:/Foo", "C:/Bar", "C:/Baz"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Split mismatch: got %v want %v", got, want)
	}
}
