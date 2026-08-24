// Package v1alpha1 contains the Tenant API types.
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion identifies this API group and version.
	GroupVersion = schema.GroupVersion{Group: "platform.example.com", Version: "v1alpha1"}

	// SchemeBuilder registers the types below with a runtime.Scheme.
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme adds the types in this group-version to the given scheme.
	AddToScheme = SchemeBuilder.AddToScheme
)
