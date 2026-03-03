package packager

import (
	"os"
	"path/filepath"
	"strings"
)

// FileEntry holds a file's content and its full relative path.
type FileEntry struct {
	Content      string // file content
	RelativePath string // full path relative to root, using forward slashes
}

// pathToMapKey converts a file path to a map key by replacing
// path separators with double underscores.
// This is used to create a valid key that could be used as a configmap key later
func pathToMapKey(path string) string {
	return strings.ReplaceAll(filepath.ToSlash(path), "/", "__")
}

// BuildFileMap walks the filesystem at rootPath and builds a map of file
// entries. For each file, the key is the file's path relative to rootPath
// with '/' replaced by "__"; the value holds the file's content and full
// relative path.
func BuildFileMap(rootPath string) (map[string]FileEntry, error) {
	root, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}

	// If we are dealing with a single file, we can just read the content of the file and return it
	if !info.IsDir() {
		content, err := os.ReadFile(root)
		if err != nil {
			return nil, err
		}
		name := info.Name()
		return map[string]FileEntry{
			pathToMapKey(name): {Content: string(content), RelativePath: name},
		}, nil
	}

	filesMap := make(map[string]FileEntry)
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip directories and non-regular files (we are only interesting at getting the content of the files)
		if d.IsDir() || !d.Type().IsRegular() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relativePath := filepath.ToSlash(rel)
		filesMap[pathToMapKey(rel)] = FileEntry{Content: string(content), RelativePath: relativePath}
		return nil
	})
	return filesMap, err
}
