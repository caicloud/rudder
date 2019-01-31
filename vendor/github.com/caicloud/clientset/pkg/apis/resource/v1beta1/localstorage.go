package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeLocalStorage describes local storage related information.
type NodeLocalStorage struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeLocalStorageSpec   `json:"spec,omitempty"`
	Status NodeLocalStorageStatus `json:"status,omitempty"`
}

// NodeLocalStorageSpec describes the information that needs to create a VG.
type NodeLocalStorageSpec struct {
	// related StorageClass name
	StorageClass string
	// related Node name
	Node string
	// chosen disks
	Disks []string
}

// NodeLocalStorageStatus describes the capacity information of the VG.
type NodeLocalStorageStatus struct {
	// the name of the VG created based on the disks above
	VG string
	// total capacity of VG
	Total int64
	// remaining(unallocated) capacity of VG
	Free int64
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// NodeLocalStorageList is a collection of NodeLocalStorage objects.
type NodeLocalStorageList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []NodeLocalStorage `json:"items"`
}
