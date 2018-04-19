package node

import (
	"fmt"
	"net"

	"k8s.io/api/core/v1"
)

// GetNodeHostIP returns the provided node's IP, based on the priority:
// 1. try to get IP from node.labels according to the labels slice order
// 2. try to get IP from node.annotations according to the annotations slice order
// 3. NodeExternalIP
// 4. NodeInternalIP
func GetNodeHostIP(node *v1.Node, labels []string, annotations []string) (net.IP, error) {
	ip := getIPFromSource(labels, node.Labels)
	if ip != nil {
		return ip, nil
	}

	ip = getIPFromSource(annotations, node.Annotations)
	if ip != nil {
		return ip, nil
	}

	addresses := node.Status.Addresses
	addressMap := make(map[v1.NodeAddressType][]v1.NodeAddress)
	for i := range addresses {
		addressMap[addresses[i].Type] = append(addressMap[addresses[i].Type], addresses[i])
	}
	if addresses, ok := addressMap[v1.NodeExternalIP]; ok {
		for _, address := range addresses {
			if ip = net.ParseIP(address.Address); ip != nil {
				return ip, nil
			}
		}
	}
	if addresses, ok := addressMap[v1.NodeInternalIP]; ok {
		for _, address := range addresses {
			if ip = net.ParseIP(address.Address); ip != nil {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("host IP unknown; known addresses: %v", addresses)
}

func getIPFromSource(keys []string, source map[string]string) net.IP {
	if len(keys) == 0 || len(source) == 0 {
		return nil
	}

	for _, key := range keys {
		if key == "" {
			continue
		}
		v, ok := source[key]
		if !ok {
			// not found, try next
			continue
		}
		ip := net.ParseIP(v)
		if ip == nil {
			// error format, try next
			continue
		}
		return ip
	}

	return nil

}
