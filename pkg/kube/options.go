package kube

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// GetOptions is a group options for getting resources.
type GetOptions struct {
	// OwnerReference confirms the desired owner for resources.
	// If the option is nil, it means that all matched object
	// are allowed, even if they have different owners.
	OwnerReferences []metav1.OwnerReference
	// IgnoreNonexistence ignores resources which is not found.
	// It won't return an error when a desired resource does
	// not exist.
	IgnoreNonexistence bool
}

// CreateOptions is a group options for creating resources
type CreateOptions struct {
	// OwnerReference desides owners to create resources.
	OwnerReferences []metav1.OwnerReference
}

// UpdateOptions is a  group options for updating resources
type UpdateOptions struct {
	// OwnerReferences enforces owners when create/update/
	// delete resources in an update operation.
	OwnerReferences []metav1.OwnerReference
}

// DeleteOptions is a  group options for deleting resources
type DeleteOptions struct {
	// OwnerReferences is used to make sure that all deleted
	// resources are belong to these owners.
	OwnerReferences []metav1.OwnerReference
}
