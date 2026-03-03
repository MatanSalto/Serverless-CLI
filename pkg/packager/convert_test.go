package packager

import (
	"reflect"
	"testing"
)

func TestFileMapToConfigData(t *testing.T) {
	t.Run("empty map returns empty data", func(t *testing.T) {
		got := FileMapToConfigData(nil)
		if got == nil {
			t.Fatal("expected non-nil map")
		}
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("single entry maps key to content", func(t *testing.T) {
		filesMap := map[string]FileEntry{
			"foo.txt": {Content: "hello", RelativePath: "foo.txt"},
		}
		got := FileMapToConfigData(filesMap)
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1", len(got))
		}
		if got["foo.txt"] != "hello" {
			t.Errorf("got[%q] = %q, want %q", "foo.txt", got["foo.txt"], "hello")
		}
	})

	t.Run("multiple entries with path keys", func(t *testing.T) {
		filesMap := map[string]FileEntry{
			"top.txt":       {Content: "top", RelativePath: "top.txt"},
			"a__b__nested": {Content: "nested", RelativePath: "a/b/nested"},
		}
		got := FileMapToConfigData(filesMap)
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
		if got["top.txt"] != "top" || got["a__b__nested"] != "nested" {
			t.Errorf("got = %v", got)
		}
	})
}

func TestFileMapToVolumeItems(t *testing.T) {
	t.Run("empty map returns empty slice", func(t *testing.T) {
		got := FileMapToVolumeItems(nil)
		if len(got) != 0 {
			t.Errorf("len(got) = %d, want 0", len(got))
		}
	})

	t.Run("single entry produces one KeyToPath", func(t *testing.T) {
		filesMap := map[string]FileEntry{
			"foo.txt": {Content: "x", RelativePath: "foo.txt"},
		}
		got := FileMapToVolumeItems(filesMap)
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1", len(got))
		}
		if got[0].Key != "foo.txt" || got[0].Path != "foo.txt" {
			t.Errorf("got[0] = {Key: %q, Path: %q}, want Key=foo.txt Path=foo.txt", got[0].Key, got[0].Path)
		}
	})

	t.Run("nested path preserves RelativePath", func(t *testing.T) {
		filesMap := map[string]FileEntry{
			"src__greet.py": {Content: "def greet()", RelativePath: "src/greet.py"},
		}
		got := FileMapToVolumeItems(filesMap)
		if len(got) != 1 {
			t.Fatalf("len(got) = %d, want 1", len(got))
		}
		if got[0].Key != "src__greet.py" || got[0].Path != "src/greet.py" {
			t.Errorf("got[0] = {Key: %q, Path: %q}, want Key=src__greet.py Path=src/greet.py", got[0].Key, got[0].Path)
		}
	})

	t.Run("multiple entries produce all items", func(t *testing.T) {
		filesMap := map[string]FileEntry{
			"main.py":       {Content: "print(1)", RelativePath: "main.py"},
			"a__b__nested": {Content: "nested", RelativePath: "a/b/nested"},
		}
		got := FileMapToVolumeItems(filesMap)
		if len(got) != 2 {
			t.Fatalf("len(got) = %d, want 2", len(got))
		}
		keys := make(map[string]string)
		for _, item := range got {
			keys[item.Key] = item.Path
		}
		want := map[string]string{"main.py": "main.py", "a__b__nested": "a/b/nested"}
		if !reflect.DeepEqual(keys, want) {
			t.Errorf("keys/paths = %v, want %v", keys, want)
		}
	})
}
