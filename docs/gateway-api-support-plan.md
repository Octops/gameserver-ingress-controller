# Plan: Kubernetes Gateway API Support

## Overview

This document describes the plan to add support for the Kubernetes Gateway API
(`gateway.networking.k8s.io`) as an alternative to the existing `networking.k8s.io/v1`
Ingress resource. Users would opt in via a new annotation on their GameServer or Fleet.

**Effort estimate: Medium** — the codebase is well-structured with clear separation of
concerns. Each layer (store, reconciler, handler, options) maps cleanly to what needs to
be added. No existing code needs to be deleted; it is purely additive until the opt-in
annotation logic is introduced.

---

## Background: What Changes Between Ingress and Gateway API

| Concern | Ingress | Gateway API equivalent |
|---|---|---|
| Routing rules | `networking.k8s.io/v1/Ingress` | `gateway.networking.k8s.io/v1/HTTPRoute` |
| Entry point config | Managed by the ingress controller class | `gateway.networking.k8s.io/v1/Gateway` (pre-provisioned by ops) |
| TLS termination | `spec.tls` on Ingress | TLS config on the `Gateway` listener; `HTTPRoute` references the backend |
| Class selection | `spec.ingressClassName` | `spec.parentRefs[].name` pointing to a `Gateway` |
| Path routing | `spec.rules[].http.paths` | `spec.rules[].matches[].path` |
| Domain routing | `spec.rules[].host` | `spec.hostnames[]` on the HTTPRoute |

The key difference: users **pre-create** the `Gateway` resource (or have ops do it); this
controller only creates `HTTPRoute` objects that reference that Gateway, just like it
currently only creates `Ingress` objects and not the underlying ingress controller.

---

## User-Facing Interface (New Annotations)

Introduce a new top-level annotation to select the routing backend:

```
octops.io/router-backend: ingress   # default — existing behaviour, no change required
octops.io/router-backend: gateway   # new — creates HTTPRoute instead of Ingress
```

Additional annotations required when `gateway` backend is selected:

| Annotation | Description | Example |
|---|---|---|
| `octops.io/gateway-name` | Name of the pre-existing `Gateway` resource | `prod-gateway` |
| `octops.io/gateway-namespace` | Namespace of the Gateway (optional, defaults to same ns) | `infra` |
| `octops.io/gateway-section-name` | `sectionName` inside the Gateway listener (optional) | `https` |
| `octops.io/gameserver-ingress-mode` | Reused as-is: `domain` or `path` routing | `domain` |
| `octops.io/gameserver-ingress-domain` | Reused as-is for domain mode | `game.example.com` |
| `octops.io/gameserver-ingress-fqdn` | Reused as-is for path mode | `game.example.com` |
| `octops.io/terminate-tls` | Reused — signals TLS intent (termination is at the Gateway) | `true` |

The existing `octops.io/ingress-class-name` annotation becomes irrelevant when using the
Gateway backend (Gateway name/section replace it).

---

## Implementation Plan

### Step 1 — Add the Gateway API Go dependency

Add `sigs.k8s.io/gateway-api` to `go.mod`. This provides the typed Go structs for
`Gateway`, `HTTPRoute`, etc. without pulling in a full controller-runtime dependency
upgrade.

**Files:** `go.mod`, `go.sum`

---

### Step 2 — Extend `pkg/gameserver/gameserver.go` with new constants

Add:
- `OctopsAnnotationRouterBackend = "octops.io/router-backend"`
- `RouterBackendIngress RouterBackend = "ingress"` (default)
- `RouterBackendGateway RouterBackend = "gateway"`
- `OctopsAnnotationGatewayName = "octops.io/gateway-name"`
- `OctopsAnnotationGatewayNamespace = "octops.io/gateway-namespace"`
- `OctopsAnnotationGatewaySectionName = "octops.io/gateway-section-name"`
- Helper `GetRouterBackend(gs) RouterBackend`

No existing constants or functions change.

---

### Step 3 — Create `pkg/stores/gateway_store.go`

Mirrors the existing `ingress_store.go` structure:

```
type gatewayStore struct {
    client   kubernetes.Interface  // replaced by the gateway-api typed client
    informer gatewayinformers.HTTPRouteInformer
}

type HTTPRouteStore interface {
    CreateHTTPRoute(ctx, route, opts) (*gatewayv1.HTTPRoute, error)
    GetHTTPRoute(name, namespace string) (*gatewayv1.HTTPRoute, error)
}
```

The store uses the gateway-api typed client (`sigs.k8s.io/gateway-api/pkg/client/clientset`)
and a shared informer factory from that package.

---

### Step 4 — Extend `pkg/stores/store.go`

- Add `*gatewayStore` to the `Store` struct (alongside the existing `*ingressStore`)
- In `NewStore`, conditionally (or always) set up the gateway informer factory
- In `HasSynced`, add the `httpRouteInformer.HasSynced` condition

Because informer setup and sync are already handled generically, this is a small, localised
change.

---

### Step 5 — Create `pkg/reconcilers/gateway_options.go`

Mirrors `ingress_options.go`. Defines:

```go
type HTTPRouteOption func(gs *agonesv1.GameServer, route *gatewayv1.HTTPRoute) error
```

Implement the following options (analogous to the Ingress equivalents):

| Option function | Ingress equivalent |
|---|---|
| `WithHTTPRouteParentRef(name, ns, section)` | `WithIngressClassName` |
| `WithHTTPRouteRules(mode)` | `WithIngressRule(mode)` |
| `WithHTTPRouteHostnames(mode)` | Part of `WithIngressRule` |
| `WithCustomHTTPRouteAnnotations()` | `WithCustomAnnotations()` |
| `WithCustomHTTPRouteAnnotationsTemplate()` | `WithCustomAnnotationsTemplate()` |

`WithHTTPRouteRules` maps the two existing routing modes:
- **domain mode**: sets `spec.hostnames` to `[{gs.Name}.{domain}]` and a single catch-all
  path match
- **path mode**: sets `spec.hostnames` to the FQDN list and a path match of `/{gs.Name}`

Both modes set the backend ref to the Service created by `ServiceReconciler` (same name as
the GameServer, same namespace) — the Service creation path is completely unchanged.

---

### Step 6 — Create `pkg/reconcilers/gateway_reconciler.go`

Mirrors `ingress_reconciler.go` exactly in structure:

```go
type HTTPRouteStore interface {
    CreateHTTPRoute(...) (*gatewayv1.HTTPRoute, error)
    GetHTTPRoute(name, namespace string) (*gatewayv1.HTTPRoute, error)
}

type GatewayReconciler struct {
    store    HTTPRouteStore
    recorder *record.EventRecorder
}

func (r *GatewayReconciler) Reconcile(ctx, gs) (*gatewayv1.HTTPRoute, bool, error)
func (r *GatewayReconciler) reconcileNotFound(ctx, gs) (*gatewayv1.HTTPRoute, bool, error)
```

`reconcileNotFound` reads the gateway annotations from the GameServer and calls
`newHTTPRoute(gs, opts...)` using the options from Step 5.

---

### Step 7 — Update `pkg/handlers/gameserver_handler.go`

This is the only file that needs logic branching:

1. Add `gatewayReconciler *reconcilers.GatewayReconciler` field to `GameSeverEventHandler`.
2. In `NewGameSeverEventHandler`, construct both reconcilers.
3. In `Reconcile`, after `serviceReconciler.Reconcile`, branch on the router backend
   annotation:

```go
backend := gameserver.GetRouterBackend(gs)
switch backend {
case gameserver.RouterBackendGateway:
    _, reconciled, err = h.gatewayReconciler.Reconcile(ctx, gs)
default:
    _, reconciled, err = h.ingressReconciler.Reconcile(ctx, gs)
}
```

The `gameserverReconciler.Reconcile` call and all surrounding logic are unchanged.

---

### Step 8 — Add `record` kinds for HTTPRoute

In `pkg/record/recorder.go` (or wherever `IngressKind` is defined), add `HTTPRouteKind`
so event recording messages remain consistent.

---

### Step 9 — Tests

Add the following test files following the patterns of existing tests:

- `pkg/reconcilers/gateway_options_test.go` — unit tests for each `HTTPRouteOption`
- `pkg/reconcilers/gateway_reconciler_test.go` — table-driven tests for
  `GatewayReconciler.Reconcile` covering "route exists" and "route not found" cases

No changes needed to existing test files.

---

## What Does NOT Change

- `ServiceReconciler` — HTTPRoute points to the same headless Service; no changes needed
- `GameServerReconciler` — annotation `octops.io/ingress-ready` is reused as-is (its
  meaning is "routing resource is ready", regardless of the resource type)
- `ingress_reconciler.go`, `ingress_options.go` — left intact; Ingress remains the default
- The controller manager setup (`pkg/app/controller.go`, `pkg/manager/manager.go`)
- The Agones store (`pkg/stores/agones_store.go`)

---

## Risks and Open Questions

1. **Gateway API CRD availability**: The cluster must have the Gateway API CRDs installed.
   The controller should log a clear error and not crash if the HTTPRoute CRD is absent
   (e.g., wrap the informer start with a CRD presence check).

2. **TLS handling**: With Ingress, TLS termination config (secret name, cert-manager
   issuer) is on the Ingress object. With Gateway API, TLS is configured on the Gateway
   listener by the cluster operator, not on the HTTPRoute. The `octops.io/terminate-tls`
   and `octops.io/issuer-tls-name` annotations become advisory/informational for the
   gateway mode. Document this clearly; consider emitting a warning event if those
   annotations are present but the backend is `gateway`.

3. **Annotation naming**: Reusing `octops.io/gameserver-ingress-mode/domain/fqdn` for
   both backends keeps the surface area small, but the names are Ingress-centric. An
   alternative is to introduce `octops.io/gameserver-routing-mode` etc. as aliases.
   Recommendation: keep the existing annotation names for now to minimise migration burden.

4. **Gateway API version**: The `gateway.networking.k8s.io/v1` group is GA as of
   Kubernetes 1.28. If older clusters must be supported, the `v1beta1` types need to be
   considered. The go client in `sigs.k8s.io/gateway-api` supports both.

5. **Status conditions**: HTTPRoute surfaces `Accepted` and `ResolvedRefs` conditions via
   its status. A future improvement could watch these conditions and reflect them back on
   the GameServer annotation (currently the annotation is set immediately after creation,
   not after the route is admitted).
