package kube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GetOptions is a group options for getting resources.
type GetOptions struct {
	// OwnerReference confirms the desired owner for resources.
	// If the option is nil, it means that all matched object
	// are allowed, even if they have different owners.
	OwnerReference *metav1.OwnerReference
	// IgnoreNonexistence ignores resources which is not found.
	// It won't return an error when a desired resource does
	// not exist.
	IgnoreNonexistence bool
}

// CreateOptions is a group options for creating resources
type CreateOptions struct {
	// OwnerReference desides the owner for created resources.
	OwnerReference *metav1.OwnerReference
}

// UpdateOptions is a  group options for updating resources
type UpdateOptions struct {
	// OwnerReference enforces the owner when create/update/
	// delete resources in an update operation.
	OwnerReference *metav1.OwnerReference
}

// DeleteOptions is a  group options for deleting resources
type DeleteOptions struct {
	// OwnerReference is used to make sure that all deleted
	// resources are belong to the owner.
	OwnerReference *metav1.OwnerReference
}
