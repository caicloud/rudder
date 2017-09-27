package status

import (
	"fmt"
	"regexp"
	"strings"

	releaseapi "github.com/caicloud/clientset/pkg/apis/release/v1alpha1"
	"github.com/golang/glog"
)

// detect checks alive releases and update status for releases.
func (sc *StatusController) detect() {
	// Get releases
	sc.lock.RLock()
	var releases []*releaseapi.Release
	for _, v := range sc.releases {
		releases = append(releases, v)
	}
	sc.lock.RUnlock()

	if len(releases) <= 0 {
		return
	}

	for _, r := range releases {
		_, err := sc.backend.ReleaseStorage(r).Patch(func(release *releaseapi.Release) {
			if release.Status.Details == nil {
				release.Status.Details = make(map[string]releaseapi.ReleaseDetailStatus)
			}
			for _, detector := range sc.detectors {
				kind := detector.Kind()
				// Detect
				result, err := detector.Detect(sc.store, release)
				if err != nil {
					glog.Errorf("Can't detect status for %s/%s: %v", r.Namespace, r.Name, err)
					continue
				}

				// Clear status for detector
				for key, _ := range release.Status.Details {
					k, _, err := ParseKey(key)
					if err != nil {
						glog.Errorf("Invalid key in details: %v", err)
					} else {
						if kind != k {
							continue
						}
					}
					delete(release.Status.Details, key)
				}

				// Set status for detector
				for name, status := range result {
					key, err := Key(kind, name)
					if err != nil {
						glog.Errorf("Can't get key for kind %s and name %s: %v", detector.Kind(), name, err)
						continue
					}
					release.Status.Details[key] = status
				}
			}
		})
		if err != nil {
			// Output the error and continue
			glog.Errorf("Can't update status for %s/%s: %v", r.Namespace, r.Name, err)
		}
	}

}

// kindValidator and nameValidator validates kind and name of a key
var kindValidator = regexp.MustCompile(`[a-zA-Z0-9]*`)
var nameValidator = regexp.MustCompile(`[a-zA-Z0-9/\.]+`)

// Key generates a key for kind and name.
func Key(kind, name string) (string, error) {
	if !kindValidator.Match([]byte(kind)) {
		return "", fmt.Errorf("invalid kind of key")
	}
	if !nameValidator.Match([]byte(name)) {
		return "", fmt.Errorf("invalid name of key")
	}
	if kind == "" {
		return name, nil
	}
	return fmt.Sprintf("%s:%s", kind, name), nil
}

// keyValidator checks if a key is valid.
var keyValidator = regexp.MustCompile(`([a-zA-Z0-9]+:)?[a-zA-Z0-9/\.]+`)

// ParseKey parses key to kind and name.
func ParseKey(key string) (kind string, name string, err error) {
	if !keyValidator.Match([]byte(key)) {
		return "", "", fmt.Errorf("invalid key")
	}
	index := strings.IndexAny(key, ":")
	if index < 0 {
		return "", key, nil
	}
	return key[:index], key[index+1:], err
}
