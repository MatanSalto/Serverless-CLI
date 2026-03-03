package packager

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildFileMap(t *testing.T) {
	t.Run("single file path returns one entry with content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "foo.txt")
		if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		data, err := BuildFileMap(path)
		if err != nil {
			t.Fatalf("BuildFileMap: %v", err)
		}
		if len(data) != 1 {
			t.Fatalf("len(data) = %d, want 1", len(data))
		}
		key := "foo.txt"
		if data[key].Content != "hello" {
			t.Errorf("data[%q].Content = %q, want %q", key, data[key].Content, "hello")
		}
		if data[key].RelativePath != "foo.txt" {
			t.Errorf("data[%q].RelativePath = %q, want %q", key, data[key].RelativePath, "foo.txt")
		}
	})

	t.Run("directory with files maps path to content with __ for slashes", func(t *testing.T) {
		root := t.TempDir()
		sub := filepath.Join(root, "a", "b")
		if err := os.MkdirAll(sub, 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "top.txt"), []byte("top"), 0644); err != nil {
			t.Fatalf("WriteFile top: %v", err)
		}
		if err := os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("nested"), 0644); err != nil {
			t.Fatalf("WriteFile nested: %v", err)
		}
		data, err := BuildFileMap(root)
		if err != nil {
			t.Fatalf("BuildFileMap: %v", err)
		}
		if len(data) != 2 {
			t.Fatalf("len(data) = %d, want 2", len(data))
		}
		if data["top.txt"].Content != "top" {
			t.Errorf("data[\"top.txt\"].Content = %q, want %q", data["top.txt"].Content, "top")
		}
		if data["top.txt"].RelativePath != "top.txt" {
			t.Errorf("data[\"top.txt\"].RelativePath = %q, want %q", data["top.txt"].RelativePath, "top.txt")
		}
		key := "a__b__nested.txt"
		if data[key].Content != "nested" {
			t.Errorf("data[%q].Content = %q, want %q", key, data[key].Content, "nested")
		}
		if data[key].RelativePath != "a/b/nested.txt" {
			t.Errorf("data[%q].RelativePath = %q, want %q", key, data[key].RelativePath, "a/b/nested.txt")
		}
	})

	t.Run("empty directory returns empty map", func(t *testing.T) {
		root := t.TempDir()
		data, err := BuildFileMap(root)
		if err != nil {
			t.Fatalf("BuildFileMap: %v", err)
		}
		if data == nil {
			t.Fatal("expected non-nil map")
		}
		if len(data) != 0 {
			t.Errorf("len(data) = %d, want 0", len(data))
		}
	})

	t.Run("a path that does not exist returns error", func(t *testing.T) {
		_, err := BuildFileMap(filepath.Join(t.TempDir(), "missing"))
		if err == nil {
			t.Error("BuildFileMap: want error for nonexistent path, got nil")
		}
	})

	t.Run("key uses double underscore for path seperators", func(t *testing.T) {
		root := t.TempDir()
		one := filepath.Join(root, "dir", "file.txt")
		if err := os.MkdirAll(filepath.Dir(one), 0755); err != nil {
			t.Fatalf("MkdirAll: %v", err)
		}
		if err := os.WriteFile(one, []byte("x"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		data, err := BuildFileMap(root)
		if err != nil {
			t.Fatalf("BuildFileMap: %v", err)
		}
		wantKey := "dir__file.txt"
		if _, ok := data[wantKey]; !ok {
			t.Errorf("expected key %q in data; got keys: %v", wantKey, mapKeys(data))
		}
		if data[wantKey].Content != "x" {
			t.Errorf("data[%q].Content = %q, want %q", wantKey, data[wantKey].Content, "x")
		}
		if data[wantKey].RelativePath != "dir/file.txt" {
			t.Errorf("data[%q].RelativePath = %q, want %q", wantKey, data[wantKey].RelativePath, "dir/file.txt")
		}
	})
}

func mapKeys(m map[string]FileEntry) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
