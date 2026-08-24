# tenant-controller

A small Kubernetes operator written in Go with
[controller-runtime](https://github.com/kubernetes-sigs/controller-runtime).
A cluster scoped `Tenant` custom resource owns one namespace and an optional
`ResourceQuota`; the controller creates them, keeps them in sync with the spec, and
reports readiness in `status`.

```yaml
apiVersion: platform.example.com/v1alpha1
kind: Tenant
metadata:
  name: acme
spec:
  owner: team-a          # optional, copied to a namespace label
  quota:                 # optional, becomes ResourceQuota.spec.hard
    cpu: "4"
    memory: 8Gi
    pods: "20"
```

```
$ kubectl get tenant
NAME   OWNER    NAMESPACE     PHASE   AGE
acme   team-a   tenant-acme   Ready   3s
```

## What the reconcile loop does

Each pass is idempotent and runs whenever a Tenant changes, or a namespace or
ResourceQuota the Tenant owns changes:

1. Get the Tenant. If it is gone, return. Owner references let Kubernetes garbage
   collect the namespace and quota, so no finalizer is needed.
2. `CreateOrUpdate` namespace `tenant-<name>` with labels
   `platform.example.com/tenant` and `platform.example.com/owner`, owned by the Tenant.
3. If `spec.quota` is set, `CreateOrUpdate` ResourceQuota `tenant-quota` in that
   namespace with `hard` equal to `spec.quota`. If it is unset, delete the quota.
4. Patch the status subresource: `phase: Ready`, `namespace: tenant-<name>`. A merge
   patch is used instead of an update so a pass that read a slightly stale cache cannot
   fail on a resourceVersion conflict, and the write is skipped when nothing changed.

If a Tenant is deleted and recreated before its old namespace finishes terminating, the
controller reports `phase: Pending` and requeues every 5 seconds until the namespace is
gone, then proceeds normally.

Because the controller watches owned objects (`Owns(&Namespace{})`,
`Owns(&ResourceQuota{})`), deleting or editing the namespace or quota by hand triggers
a pass that restores the desired state.

## Layout

```
api/v1alpha1/         Tenant types, scheme registration, DeepCopy methods
controllers/          TenantReconciler and fake client unit tests
cmd/main.go           manager wiring and signal handling
config/crd/           CRD with OpenAPI schema, status subresource, printer columns
config/rbac.yaml      ClusterRole for running in cluster
config/samples/       example Tenant
```

Written against controller-runtime directly rather than the kubebuilder scaffold so
every file is hand written and small enough to read in one sitting.

## Run it

Requires Go 1.25+ and a cluster in your kubeconfig (kind, OrbStack, minikube, etc).

```
go test ./...

kind create cluster --name tenants          # or any local cluster
kubectl apply -f config/crd/tenant.yaml
go run ./cmd                                # in one terminal

kubectl apply -f config/samples/tenant.yaml # in another
kubectl get tenant acme
kubectl get ns tenant-acme --show-labels
kubectl get resourcequota -n tenant-acme

kubectl delete resourcequota tenant-quota -n tenant-acme   # comes back
kubectl patch tenant acme --type=merge -p '{"spec":{"quota":{"cpu":"8"}}}'
kubectl delete tenant acme                                 # namespace is garbage collected
```

Clean up with `kubectl delete crd tenants.platform.example.com`.

## Tests

`controllers/tenant_controller_test.go` uses the controller-runtime fake client, so the
suite runs in under a second with no downloaded binaries. It covers namespace creation
with owner reference, quota creation, quota update on spec change, no quota when
`spec.quota` is empty, status reporting, no status write when already Ready, status
written via Patch rather than Update, waiting on a terminating namespace, and a
NotFound Tenant.

## Not included on purpose

Finalizers, admission webhooks, metrics, leader election, a container image, and a Helm
chart. Each is a reasonable next step; none is needed to understand the reconcile loop.
