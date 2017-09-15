package render

import "strings"

const delimiter = "\n---\n"

// MergeManifests merges a list of resources to a manifest.
func MergeResources(resources []string) string {
	return strings.Join(resources, delimiter)
}

// SplitManifest splits a manifest to a list of resources
func SplitManifest(manifest string) []string {
	result := strings.Split(manifest, delimiter)
	length := 0
	for _, res := range result {
		if r := strings.TrimSpace(res); r != "" {
			result[length] = r
			length++
		}
	}
	return result[:length:length]
}
