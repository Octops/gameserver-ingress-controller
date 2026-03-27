# AGENTS.md — Gameserver Ingress Controller

Guidance for AI agents working in this repository.

---

## What This Project Does

This is a **Kubernetes controller** that automatically provisions network ingress resources for game servers managed by [Agones](https://agones.dev/). When Agones creates a `GameServer`, this controller watches for it and creates the corresponding `Service`, `Ingress` (or `HTTPRoute` for Gateway API), and annotates the `GameServer` as ingress-ready.

**Without this controller**, game server operators would manually manage hundreds of ingress rules as game servers spin up/down dynamically. This automates the full lifecycle.

**Two routing backends are supported:**
- `ingress` (default) — standard `networking.k8s.io/v1` Ingress, one per game server, TLS via cert-manager per server
- `gateway` — Kubernetes Gateway API (`gateway.networking.k8s.io/v1`) HTTPRoute, attaches to a shared pre-provisioned Gateway; no per-server TLS (cert on the Gateway listener instead)

**Two routing modes** apply to both backends:
- `domain` — each server gets a subdomain: `<gs-name>.<domain>`
- `path` — all servers share an FQDN with path-based routing: `<fqdn>/<gs-name>`

---

## Repository Layout

```
cmd/                    CLI entry point (Cobra)
pkg/
  app/                  Bootstrap: wires manager, stores, handler, controller
  controller/           controller-runtime Reconciler that watches GameServer objects
  handlers/             EventHandler: OnAdd/OnUpdate/OnDelete → calls reconcilers
  reconcilers/          Business logic per resource type
    service_reconciler.go
    ingress_reconciler.go
    gateway_reconciler.go   ← new Gateway API feature
    gameserver_reconciler.go
    *_options.go            ← functional options for each reconciler
  stores/               Kubernetes API abstraction (informer-backed caches)
    store.go            ← composite store wiring
    gateway_store.go    ← new Gateway API store
  gameserver/           Constants, annotation helpers, GameServer utility funcs
  record/               Kubernetes event recorder wrapper
  k8sutil/              Cluster config and client helpers
  manager/              controller-runtime manager setup
internal/
  runtime/              Logger (logrus) and signal handling
  version/              Build metadata
deploy/
  install.yaml          Controller RBAC + Deployment manifest
  gateway/              Gateway API infrastructure (GatewayClass for Contour provisioner)
  cert-manager/         ClusterIssuer examples — ingress mode only
examples/               Sample Fleet manifests for all routing modes and backends
docs/                   Design documents (gateway-api-support-plan.md)
```

---

## Core Data Flow

```
GameServer created/updated in Kubernetes
        │
        ▼
GameServerController (controller-runtime Reconciler)
        │
        ▼
GameSeverEventHandler.OnAdd / OnUpdate
        │  skip if: missing octops.io/gameserver-ingress-mode annotation
        │  skip if: GameServer in Shutdown state
        │  only proceed for: Scheduled, RequestReady, Ready states
        │
        ▼
Reconcile() — sequential pipeline:
  1. ServiceReconciler.Reconcile()        → headless ClusterIP Service
  2. IngressReconciler OR GatewayReconciler.Reconcile()
       IngressReconciler  → networking.k8s.io/v1 Ingress
       GatewayReconciler  → gateway.networking.k8s.io/v1 HTTPRoute
  3. GameServerReconciler.Reconcile()     → patch octops.io/ingress-ready=true
        │
        ▼
Kubernetes Events recorded for audit (via record.EventRecorder)
```

All created resources carry an **OwnerReference** back to the `GameServer`, so deleting the `GameServer` cascades to cleanup automatically.

---

## Key Annotations Reference

All annotations live under the `octops.io/` prefix and are set on `GameServer` / `Fleet` spec templates.

| Annotation | Required | Values | Notes |
|---|---|---|---|
| `octops.io/gameserver-ingress-mode` | yes | `domain` \| `path` | Missing = skip reconciliation entirely |
| `octops.io/router-backend` | no | `ingress` (default) \| `gateway` | Selects backend |
| `octops.io/gameserver-ingress-domain` | domain mode | e.g. `game.example.com` | Used as base domain |
| `octops.io/gameserver-ingress-fqdn` | path mode | e.g. `servers.example.com` | Used as hostname |
| `octops.io/terminate-tls` | no | `true` \| `false` | Ingress only; warns if set in gateway mode |
| `octops.io/tls-secret-name` | no | secret name | Custom TLS secret |
| `octops.io/issuer-tls-name` | no | ClusterIssuer name | Adds `cert-manager.io/cluster-issuer`; warns if set in gateway mode |
| `octops.io/ingress-class-name` | no | e.g. `contour` | Sets `spec.ingressClassName` |
| `octops.io/ingress-ready` | internal | `true` | Written by controller, do not set manually |
| `octops.io/gateway-name` | gateway mode | Gateway resource name | Required when `router-backend=gateway` |
| `octops.io/gateway-namespace` | no | namespace | Defaults to GameServer namespace |
| `octops.io/gateway-section-name` | no | listener name | Optional Gateway listener |

**Custom annotation forwarding:**
- `octops-<key>: <value>` → forwarded to Ingress/HTTPRoute as `<key>: <value>` (prefix stripped)
- `octops.service-<key>: <value>` → forwarded to Service as `<key>: <value>` (prefix stripped)
- Values support Go templates with `.Name` and `.Port` fields

---

## Code Patterns — Follow These Exactly

### 1. Functional Options Pattern (used everywhere in reconcilers)

Options are functions with signature `func(*agonesv1.GameServer, *ResourceType) error`. They are composed in the `reconcileNotFound` method:

```go
opts := []IngressOption{
    WithCustomAnnotations(),
    WithIngressRule(mode),
    WithTLS(mode),
}
ingress, err := newIngress(gs, opts...)
```

When adding new resource features, **add a new `WithXxx()` option function** in the corresponding `*_options.go` file — do not bloat the reconciler itself.

### 2. Reconciler Pattern

Every reconciler follows this structure:

```go
func (r *XxxReconciler) Reconcile(ctx, gs) (*Resource, bool, error) {
    resource, err := r.store.GetXxx(gs.Name, gs.Namespace)
    if err != nil {
        if k8serrors.IsNotFound(err) {
            return r.reconcileNotFound(ctx, gs)
        }
        return nil, false, errors.Wrapf(err, "error retrieving Xxx ...")
    }
    return resource, false, nil   // already exists, routeReconciled=false
}

func (r *XxxReconciler) reconcileNotFound(ctx, gs) (*Resource, bool, error) {
    r.recorder.RecordCreating(gs, record.XxxKind)
    // build resource via options
    // create via store
    r.recorder.RecordSuccess(gs, record.XxxKind)
    return result, true, nil      // just created, routeReconciled=true
}
```

The `bool` return indicates whether reconciliation just created the resource (used by the handler to decide whether to proceed).

### 3. Store Pattern

Stores wrap Kubernetes clients + informers. The `GetXxx` methods read from the **informer cache** (fast, no API call). The `CreateXxx` methods write directly to the **API server**.

The composite `Store` in `pkg/stores/store.go` embeds all sub-stores and is the single injection point into handlers.

When adding a new resource type, add:
- a new `*_store.go` with a typed struct implementing a store interface
- wire it in `stores.NewStore()`

### 4. Error Handling

Always use `github.com/pkg/errors` for wrapping (not `fmt.Errorf`):

```go
return nil, errors.Wrapf(err, "failed to create HTTPRoute for gameserver %s", gs.Name)
```

Use `k8serrors.IsNotFound(err)` and `k8serrors.IsAlreadyExists(err)` for Kubernetes API errors.

### 5. Logging

Use the package-level logger from `internal/runtime`:

```go
logger := runtime.Logger().WithField("component", "my_component")
logger.WithFields(logrus.Fields{"key": val}).Info("message")
```

Do not instantiate your own loggers. Pass context via `.WithField()` chaining.

### 6. Event Recording

Record Kubernetes events at each significant reconciliation step:

```go
r.recorder.RecordCreating(gs, record.HTTPRouteKind)
r.recorder.RecordSuccess(gs, record.HTTPRouteKind)
r.recorder.RecordFailed(gs, record.HTTPRouteKind, err)
r.recorder.RecordWarning(gs, record.HTTPRouteKind, "human-readable explanation")
```

`RecordWarning` is used (not errors) when an annotation is set but has no effect in the current mode (e.g., TLS annotations in gateway mode).

---

## Gateway API Backend

`router-backend: gateway` is a fully implemented alternative to the Ingress backend. The implementation lives in:

- `pkg/reconcilers/gateway_reconciler.go` — creates `HTTPRoute` resources
- `pkg/reconcilers/gateway_options.go` — `WithHTTPRouteParentRef()`, `WithHTTPRouteRules()`, etc.
- `pkg/stores/gateway_store.go` — `GetHTTPRoute` / `CreateHTTPRoute`
- `examples/gateway/` — sample Fleet manifests and infrastructure YAML
- `deploy/gateway/gatewayclass.yaml` — GatewayClass for Contour Gateway Provisioner

**Key design decisions:**
- HTTPRoutes do not manage TLS. TLS lives on the Gateway listener (provisioned by ops via `examples/gateway/gateway.yaml` + `certificate.yaml`). This avoids cert-manager rate limits at scale.
- The `Gateway` resource is pre-provisioned by the cluster operator — the controller only creates `HTTPRoute` resources, one per `GameServer`.
- If `octops.io/terminate-tls` or `octops.io/issuer-tls-name` are set on a game server using the gateway backend, the controller emits a warning event rather than failing — these annotations have no effect in gateway mode.
- OwnerReferences are set on each `HTTPRoute` pointing to its `GameServer`, so Kubernetes GC handles cleanup automatically when a game server is deleted.

**Cluster setup for gateway mode** requires two separate installs that can coexist with the standard Contour ingress install:
```bash
# Use experimental-install (not standard-install): includes TCPRoute/UDPRoute CRDs
# required by Contour v1.32.1. Must use --server-side --force-conflicts because
# the httproutes CRD exceeds the 262144-byte client-side apply annotation limit.
kubectl apply --server-side --force-conflicts \
  -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.5.1/experimental-install.yaml

# Contour v1.32.1 watches TLSRoute/v1alpha2 and BackendTLSPolicy/v1alpha3, but
# Gateway API v1.5.1 ships those versions as served=false. Patch them:
for crd in tlsroutes.gateway.networking.k8s.io backendtlspolicies.gateway.networking.k8s.io; do
  kubectl get crd "$crd" -o json \
    | jq '.spec.versions = (.spec.versions | map(if .name == "v1alpha2" or .name == "v1alpha3" then .served = true else . end))' \
    | kubectl apply -f -
done

kubectl apply -f https://raw.githubusercontent.com/projectcontour/contour/v1.32.1/examples/render/contour-gateway-provisioner.yaml
kubectl apply -f deploy/gateway/gatewayclass.yaml
```

Use `hack/setup-infra.sh` to automate all of the above (including Agones and cert-manager).

---

## Local Development

To run the controller locally against a cluster (the published image will not have unreleased changes):

```bash
make docker                        # build image from current source
make run                           # docker run mounting .infrastructure/config.yml as kubeconfig
```

`hack/setup-infra.sh` installs all cluster dependencies (Agones, cert-manager, both Contour variants, Gateway API CRDs, GatewayClass, and a test Gateway) from scratch. `hack/teardown-infra.sh` cleans everything up, including force-finalizing any stuck namespaces. Both scripts require `kubectl`, `helm`, and `jq`.

---

## Gateway API Experimental Status

> **The Gateway API backend is experimental.** It has been validated end-to-end but has not seen production usage. Bugs and feedback should be reported as GitHub issues at https://github.com/Octops/gameserver-ingress-controller/issues.

Known sharp edges for AI agents working in this area:
- Contour v1.32.1 + Gateway API v1.5.1 require CRD patching (see setup above) — this is a Contour bug, not ours.
- `hack/setup-infra.sh` applies the provisioner with `--force-conflicts` because it bundles older Gateway API CRDs (v1.2.1) that conflict with v1.5.1; the resulting CRD errors are expected and harmless.
- The example game server (`octops/gameserver-http:latest`) listens on `containerPort: 8088`. Fleet manifests must match this — do not use the default Agones `7777`.
- On Docker Desktop, NodePorts are not reachable from the host. Use `kubectl port-forward -n octops-gateway svc/envoy-gateway 8080:80` for local testing.

---

## Testing Guidelines

- Tests are in `pkg/reconcilers/*_test.go`
- Use table-driven tests for option combinations
- Create mock `GameServer` objects inline with annotations maps — see existing tests for the pattern
- Use `github.com/stretchr/testify/require` (not `assert`) so tests fail fast on the first error
- No mocking frameworks — tests build real objects and call functions directly

---

## Dependencies Worth Knowing

| Dependency | Version | Purpose |
|---|---|---|
| `agones.dev/agones` | v1.56.0 | GameServer CRD types and client |
| `sigs.k8s.io/gateway-api` | v1.5.1 | HTTPRoute, Gateway CRD types |
| `sigs.k8s.io/controller-runtime` | v0.23.3 | Reconciler framework, manager |
| `k8s.io/client-go` | v0.35.1 | Kubernetes client and informers |
| `github.com/pkg/errors` | v0.9.1 | Error wrapping (use this, not fmt.Errorf) |
| `github.com/sirupsen/logrus` | v1.9.3 | Structured logging |

Go version: **1.26.1**

---

## Deployment Notes

- Namespace: `octops-system`
- RBAC: needs `get/list/watch/create/delete` on Services, Ingresses, HTTPRoutes; `get/list/watch/update` on GameServers
- Health probe: `:30235/healthz`
- Metrics: `:9090` (Prometheus)
- Sync period default: `15s`
- Max concurrent reconciles default: `10`
- Docker image built via `make docker`; `make install` deploys via `kubectl apply`

**Ingress backend infrastructure** (standard Contour):
```bash
kubectl apply -f https://projectcontour.io/quickstart/contour.yaml
```

**Gateway API backend infrastructure** (Contour Gateway Provisioner — independent, can run alongside the above):
```bash
# See the Gateway API Backend section above for full install steps including
# the required --server-side flag and CRD compatibility patches.
kubectl apply -f deploy/gateway/gatewayclass.yaml   # after provisioner is installed
```
