package v1alpha1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies the receiver into out.
func (in *Tenant) DeepCopyInto(out *Tenant) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec.Owner = in.Spec.Owner
	out.Spec.Quota = in.Spec.Quota.DeepCopy()
	out.Status = in.Status
}

// DeepCopy returns a deep copy of the receiver.
func (in *Tenant) DeepCopy() *Tenant {
	if in == nil {
		return nil
	}
	out := new(Tenant)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *Tenant) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}

// DeepCopyInto copies the receiver into out.
func (in *TenantList) DeepCopyInto(out *TenantList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]Tenant, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

// DeepCopy returns a deep copy of the receiver.
func (in *TenantList) DeepCopy() *TenantList {
	if in == nil {
		return nil
	}
	out := new(TenantList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject implements runtime.Object.
func (in *TenantList) DeepCopyObject() runtime.Object {
	return in.DeepCopy()
}
