package render

import "strings"

const delimiter = "\n---\n"

// MergeManifests merges a list of resources to a manifest.
func MergeResources(resources []string) string {
	return strings.Join(resources, delimiter)
}

// SplitManifest splits a manifest to a list of resources
func SplitManifest(manifest string) []string {
	return strings.Split(manifest, delimiter)
}
