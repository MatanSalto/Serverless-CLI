package packager

import (
	corev1 "k8s.io/api/core/v1"
)

// FileMapToConfigData converts a file map to ConfigMap Data (key -> file content - no path here).
// The map key is used as the ConfigMap key; the value is FileEntry.Content.
func FileMapToConfigData(filesMap map[string]FileEntry) map[string]string {
	data := make(map[string]string, len(filesMap))
	for k, entry := range filesMap {
		data[k] = entry.Content
	}
	return data
}

// FileMapToVolumeItems converts a file map to KeyToPath items for a ConfigMap volume.
// The key is the configmap key, and the value is the relative path of the file in the container.
func FileMapToVolumeItems(filesMap map[string]FileEntry) []corev1.KeyToPath {
	items := make([]corev1.KeyToPath, 0, len(filesMap))
	for k, entry := range filesMap {
		items = append(items, corev1.KeyToPath{
			Key:  k,
			Path: entry.RelativePath,
		})
	}
	return items
}

// FileMapTotalSize returns the total byte size of all file contents in the map.
func FileMapTotalSize(filesMap map[string]FileEntry) int64 {
	var n int64
	for _, entry := range filesMap {
		n += int64(len(entry.Content))
	}
	return n
}
