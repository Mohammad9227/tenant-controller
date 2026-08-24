// Package controllers holds the Tenant reconciler.
package controllers

import (
	"context"
	"errors"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	platformv1alpha1 "github.com/mohammed/tenant-controller/api/v1alpha1"
)

const (
	// TenantLabel marks the namespace with the tenant that owns it.
	TenantLabel = "platform.example.com/tenant"
	// OwnerLabel carries spec.owner onto the namespace for easy selection.
	OwnerLabel = "platform.example.com/owner"
	// QuotaName is the name of the ResourceQuota created in each tenant namespace.
	QuotaName = "tenant-quota"
)

// TenantReconciler drives a Tenant toward its desired state: one namespace
// named tenant-<name>, plus a ResourceQuota when spec.quota is set.
type TenantReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// NamespaceName returns the namespace a tenant owns.
func NamespaceName(tenant *platformv1alpha1.Tenant) string {
	return "tenant-" + tenant.Name
}

// namespaceTerminatingRetry is how long to wait before checking whether a
// namespace left over from a deleted Tenant of the same name has gone away.
const namespaceTerminatingRetry = 5 * time.Second

// errNamespaceTerminating is returned while a previous namespace with the
// tenant's name is still being deleted by Kubernetes.
var errNamespaceTerminating = errors.New("namespace is terminating")

// Reconcile is called for every change to a Tenant or one of its owned objects.
func (r *TenantReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var tenant platformv1alpha1.Tenant
	if err := r.Get(ctx, req.NamespacedName, &tenant); err != nil {
		// Deleted tenants need no work: the namespace and quota carry owner
		// references, so garbage collection removes them.
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if err := r.reconcileNamespace(ctx, &tenant); err != nil {
		if errors.Is(err, errNamespaceTerminating) {
			// A Tenant was deleted and recreated before its old namespace
			// finished terminating. Wait for it rather than erroring in a loop.
			logger.Info("waiting for previous namespace to terminate", "namespace", NamespaceName(&tenant))
			if err := r.updateStatus(ctx, &tenant, platformv1alpha1.TenantStatus{Phase: platformv1alpha1.PhasePending}); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: namespaceTerminatingRetry}, nil
		}
		return ctrl.Result{}, fmt.Errorf("namespace: %w", err)
	}
	if err := r.reconcileQuota(ctx, &tenant); err != nil {
		return ctrl.Result{}, fmt.Errorf("quota: %w", err)
	}

	if err := r.updateStatus(ctx, &tenant, platformv1alpha1.TenantStatus{
		Phase:     platformv1alpha1.PhaseReady,
		Namespace: NamespaceName(&tenant),
	}); err != nil {
		return ctrl.Result{}, err
	}

	logger.Info("reconciled tenant", "namespace", tenant.Status.Namespace)
	return ctrl.Result{}, nil
}

// updateStatus patches the status subresource when it differs from desired.
// A merge patch carries no resourceVersion, so a reconcile that read the Tenant
// from a slightly stale cache cannot fail with a conflict. Skipping unchanged
// status also avoids a needless write (and a needless watch event) per pass.
func (r *TenantReconciler) updateStatus(ctx context.Context, tenant *platformv1alpha1.Tenant, desired platformv1alpha1.TenantStatus) error {
	if tenant.Status == desired {
		return nil
	}
	base := tenant.DeepCopy()
	tenant.Status = desired
	if err := r.Status().Patch(ctx, tenant, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("status: %w", err)
	}
	return nil
}

func (r *TenantReconciler) reconcileNamespace(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NamespaceName(tenant)}}
	if err := r.Get(ctx, client.ObjectKeyFromObject(ns), ns); err == nil && ns.DeletionTimestamp != nil {
		return errNamespaceTerminating
	} else if client.IgnoreNotFound(err) != nil {
		return err
	}
	ns = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: NamespaceName(tenant)}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ns, func() error {
		if ns.Labels == nil {
			ns.Labels = map[string]string{}
		}
		ns.Labels[TenantLabel] = tenant.Name
		if tenant.Spec.Owner != "" {
			ns.Labels[OwnerLabel] = tenant.Spec.Owner
		}
		return controllerutil.SetControllerReference(tenant, ns, r.Scheme)
	})
	return err
}

func (r *TenantReconciler) reconcileQuota(ctx context.Context, tenant *platformv1alpha1.Tenant) error {
	rq := &corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{
		Name:      QuotaName,
		Namespace: NamespaceName(tenant),
	}}

	if len(tenant.Spec.Quota) == 0 {
		// Quota removed from spec: delete any quota we created earlier.
		err := r.Delete(ctx, rq)
		return client.IgnoreNotFound(err)
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, rq, func() error {
		rq.Spec.Hard = tenant.Spec.Quota.DeepCopy()
		return controllerutil.SetControllerReference(tenant, rq, r.Scheme)
	})
	return err
}

// SetupWithManager registers the reconciler and the objects it watches.
func (r *TenantReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&platformv1alpha1.Tenant{}).
		Owns(&corev1.Namespace{}).
		Owns(&corev1.ResourceQuota{}).
		Complete(r)
}
