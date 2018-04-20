package name

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spaolacci/murmur3"
)

// AnnoAlias is alias is the key in the k8s annotation.
const AnnoAlias = "caicloud.io/alias"

// GenerateHashName returns name of CRD. see https://github.com/caicloud/platform/issues/525 for more
func GenerateHashName(module string, namespace string) string {
	timestamp := strconv.FormatUint(uint64(time.Now().UnixNano()), 10)

	arr := []string{module}
	if namespace != "" {
		arr = append(arr, namespace)
	}
	arr = append(arr, timestamp)

	name := strings.Join(arr, "-")
	return fmt.Sprintf("%s-%d", module, murmur3.Sum32([]byte(name)))
}
