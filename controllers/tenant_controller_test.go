package controllers

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"

	platformv1alpha1 "github.com/mohammed/tenant-controller/api/v1alpha1"
)

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	s := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	if err := platformv1alpha1.AddToScheme(s); err != nil {
		t.Fatal(err)
	}
	return s
}

func newTenant(name string, quota corev1.ResourceList) *platformv1alpha1.Tenant {
	return &platformv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("uid-" + name)},
		Spec:       platformv1alpha1.TenantSpec{Owner: "team-a", Quota: quota},
	}
}

func reconcile(t *testing.T, c client.Client, name string) {
	t.Helper()
	r := &TenantReconciler{Client: c, Scheme: c.Scheme()}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: name}}
	if _, err := r.Reconcile(context.Background(), req); err != nil {
		t.Fatalf("reconcile: %v", err)
	}
}

func TestReconcileCreatesNamespaceOwnedByTenant(t *testing.T) {
	tenant := newTenant("acme", nil)
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).Build()

	reconcile(t, c, "acme")

	var ns corev1.Namespace
	if err := c.Get(context.Background(), types.NamespacedName{Name: "tenant-acme"}, &ns); err != nil {
		t.Fatalf("expected namespace tenant-acme: %v", err)
	}
	if got := ns.Labels["platform.example.com/tenant"]; got != "acme" {
		t.Errorf("tenant label = %q, want %q", got, "acme")
	}
	if len(ns.OwnerReferences) != 1 || ns.OwnerReferences[0].Name != "acme" || ns.OwnerReferences[0].Kind != "Tenant" {
		t.Errorf("owner refs = %+v, want one ref to Tenant acme", ns.OwnerReferences)
	}
}

func TestReconcileCreatesResourceQuotaFromSpec(t *testing.T) {
	quota := corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")}
	tenant := newTenant("acme", quota)
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).Build()

	reconcile(t, c, "acme")

	var rq corev1.ResourceQuota
	key := types.NamespacedName{Namespace: "tenant-acme", Name: "tenant-quota"}
	if err := c.Get(context.Background(), key, &rq); err != nil {
		t.Fatalf("expected resource quota: %v", err)
	}
	if !rq.Spec.Hard[corev1.ResourceCPU].Equal(resource.MustParse("2")) {
		t.Errorf("hard cpu = %v, want 2", rq.Spec.Hard[corev1.ResourceCPU])
	}
}

func TestReconcileUpdatesQuotaWhenSpecChanges(t *testing.T) {
	tenant := newTenant("acme", corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")})
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).Build()
	reconcile(t, c, "acme")

	var current platformv1alpha1.Tenant
	if err := c.Get(context.Background(), types.NamespacedName{Name: "acme"}, &current); err != nil {
		t.Fatal(err)
	}
	current.Spec.Quota = corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("4")}
	if err := c.Update(context.Background(), &current); err != nil {
		t.Fatal(err)
	}
	reconcile(t, c, "acme")

	var rq corev1.ResourceQuota
	key := types.NamespacedName{Namespace: "tenant-acme", Name: "tenant-quota"}
	if err := c.Get(context.Background(), key, &rq); err != nil {
		t.Fatal(err)
	}
	if !rq.Spec.Hard[corev1.ResourceCPU].Equal(resource.MustParse("4")) {
		t.Errorf("hard cpu = %v, want 4 after spec change", rq.Spec.Hard[corev1.ResourceCPU])
	}
}

func TestReconcileSkipsQuotaWhenSpecEmpty(t *testing.T) {
	tenant := newTenant("acme", nil)
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).Build()

	reconcile(t, c, "acme")

	var rq corev1.ResourceQuota
	key := types.NamespacedName{Namespace: "tenant-acme", Name: "tenant-quota"}
	if err := c.Get(context.Background(), key, &rq); err == nil {
		t.Errorf("expected no resource quota when spec.quota is empty, found %+v", rq.Spec.Hard)
	}
}

func TestReconcileSetsStatusReady(t *testing.T) {
	tenant := newTenant("acme", nil)
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).Build()

	reconcile(t, c, "acme")

	var got platformv1alpha1.Tenant
	if err := c.Get(context.Background(), types.NamespacedName{Name: "acme"}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != platformv1alpha1.PhaseReady {
		t.Errorf("status.phase = %q, want %q", got.Status.Phase, platformv1alpha1.PhaseReady)
	}
	if got.Status.Namespace != "tenant-acme" {
		t.Errorf("status.namespace = %q, want tenant-acme", got.Status.Namespace)
	}
}

func TestReconcileIgnoresDeletedTenant(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).Build()

	reconcile(t, c, "gone") // must not error on NotFound
}

func TestReconcileSkipsStatusWriteWhenAlreadyReady(t *testing.T) {
	tenant := newTenant("acme", nil)
	tenant.Status = platformv1alpha1.TenantStatus{Phase: platformv1alpha1.PhaseReady, Namespace: "tenant-acme"}
	statusWrites := 0
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, cl client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				statusWrites++
				return cl.SubResource(sub).Update(ctx, obj, opts...)
			},
			SubResourcePatch: func(ctx context.Context, cl client.Client, sub string, obj client.Object, patch client.Patch, opts ...client.SubResourcePatchOption) error {
				statusWrites++
				return cl.SubResource(sub).Patch(ctx, obj, patch, opts...)
			},
		}).Build()

	reconcile(t, c, "acme")

	if statusWrites != 0 {
		t.Errorf("status writes = %d, want 0 when status already matches", statusWrites)
	}
}

func TestReconcileWritesStatusWithPatchNotUpdate(t *testing.T) {
	tenant := newTenant("acme", nil)
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant).WithStatusSubresource(tenant).
		WithInterceptorFuncs(interceptor.Funcs{
			SubResourceUpdate: func(ctx context.Context, cl client.Client, sub string, obj client.Object, opts ...client.SubResourceUpdateOption) error {
				t.Errorf("status Update called; want Patch so a stale cache read cannot conflict")
				return cl.SubResource(sub).Update(ctx, obj, opts...)
			},
		}).Build()

	reconcile(t, c, "acme")
}

func TestReconcileWaitsWhileOldNamespaceIsTerminating(t *testing.T) {
	tenant := newTenant("acme", corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("2")})
	now := metav1.Now()
	terminating := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{
		Name:              "tenant-acme",
		DeletionTimestamp: &now,
		Finalizers:        []string{"kubernetes"},
	}}
	c := fake.NewClientBuilder().WithScheme(newScheme(t)).WithObjects(tenant, terminating).WithStatusSubresource(tenant).Build()

	r := &TenantReconciler{Client: c, Scheme: c.Scheme()}
	res, err := r.Reconcile(context.Background(), ctrl.Request{NamespacedName: types.NamespacedName{Name: "acme"}})
	if err != nil {
		t.Fatalf("want no error while namespace terminates, got %v", err)
	}
	if res.RequeueAfter <= 0 {
		t.Errorf("RequeueAfter = %v, want a positive delay", res.RequeueAfter)
	}

	var rq corev1.ResourceQuota
	if err := c.Get(context.Background(), types.NamespacedName{Namespace: "tenant-acme", Name: "tenant-quota"}, &rq); err == nil {
		t.Errorf("quota was created in a terminating namespace")
	}
	var got platformv1alpha1.Tenant
	if err := c.Get(context.Background(), types.NamespacedName{Name: "acme"}, &got); err != nil {
		t.Fatal(err)
	}
	if got.Status.Phase != platformv1alpha1.PhasePending {
		t.Errorf("status.phase = %q, want %q", got.Status.Phase, platformv1alpha1.PhasePending)
	}
}
