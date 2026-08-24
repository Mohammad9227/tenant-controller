package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TenantSpec is the desired state of a Tenant.
type TenantSpec struct {
	// Owner is a free-form label for the team that owns the tenant.
	Owner string `json:"owner,omitempty"`
	// Quota is the hard resource quota applied to the tenant namespace.
	// If empty, no ResourceQuota is created.
	Quota corev1.ResourceList `json:"quota,omitempty"`
}

// Tenant phases.
const (
	PhasePending = "Pending"
	PhaseReady   = "Ready"
)

// TenantStatus is the observed state of a Tenant.
type TenantStatus struct {
	// Phase is Pending until the namespace and quota exist, then Ready.
	Phase string `json:"phase,omitempty"`
	// Namespace is the name of the namespace created for this tenant.
	Namespace string `json:"namespace,omitempty"`
}

// Tenant is a cluster-scoped resource that owns one namespace and its quota.
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TenantSpec   `json:"spec,omitempty"`
	Status TenantStatus `json:"status,omitempty"`
}

// TenantList is a list of Tenant.
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Tenant{}, &TenantList{})
}
