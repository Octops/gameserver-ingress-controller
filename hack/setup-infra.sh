#!/usr/bin/env bash
set -euo pipefail

AGONES_VERSION="1.56.0"
CERT_MANAGER_VERSION="v1.20.0"
CONTOUR_VERSION="1.32.1"
GATEWAY_API_VERSION="v1.5.1"

AGONES_NAMESPACE="agones-system"
CERT_MANAGER_NAMESPACE="cert-manager"
CONTOUR_NAMESPACE="projectcontour"
GATEWAY_NAMESPACE="octops-gateway"

# -----------------------------------------------------------------------------
# Helpers
# -----------------------------------------------------------------------------

info()    { echo "[INFO]  $*"; }
success() { echo "[OK]    $*"; }
die()     { echo "[ERROR] $*" >&2; exit 1; }

require() {
  for cmd in "$@"; do
    command -v "$cmd" &>/dev/null || die "'$cmd' is required but not installed."
  done
}

wait_for_rollout() {
  local namespace="$1"
  local resource="$2"
  info "Waiting for $resource in namespace $namespace ..."
  kubectl rollout status "$resource" -n "$namespace" --timeout=300s
}

# -----------------------------------------------------------------------------
# Preflight
# -----------------------------------------------------------------------------

require kubectl helm jq

kubectl cluster-info &>/dev/null || die "Cannot reach the cluster. Check your kubeconfig."

# -----------------------------------------------------------------------------
# Agones
# -----------------------------------------------------------------------------

info "Installing Agones v${AGONES_VERSION} ..."

helm repo add agones https://agones.dev/chart/stable --force-update
helm repo update agones

helm upgrade --install agones agones/agones \
  --version "${AGONES_VERSION}" \
  --namespace "${AGONES_NAMESPACE}" \
  --create-namespace \
  --set gameservers.namespaces="{default}" \
  --set agones.ping.install=false \
  --wait \
  --timeout 5m

success "Agones installed."

# -----------------------------------------------------------------------------
# cert-manager
# -----------------------------------------------------------------------------

info "Installing cert-manager ${CERT_MANAGER_VERSION} ..."

helm repo add jetstack https://charts.jetstack.io --force-update
helm repo update jetstack

helm upgrade --install cert-manager jetstack/cert-manager \
  --version "${CERT_MANAGER_VERSION}" \
  --namespace "${CERT_MANAGER_NAMESPACE}" \
  --create-namespace \
  --set crds.enabled=true \
  --wait \
  --timeout 5m

success "cert-manager installed."

# -----------------------------------------------------------------------------
# Contour ingress controller (standard install — handles Ingress resources)
# -----------------------------------------------------------------------------

info "Installing Contour ${CONTOUR_VERSION} (ingress mode) ..."

kubectl apply -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour.yaml"

wait_for_rollout "${CONTOUR_NAMESPACE}" deployment/contour
kubectl rollout status daemonset/envoy -n "${CONTOUR_NAMESPACE}" --timeout=300s

success "Contour (ingress mode) installed."

# -----------------------------------------------------------------------------
# Gateway API CRDs (experimental channel — required by Contour v1.32.1)
# -----------------------------------------------------------------------------

info "Installing Gateway API CRDs ${GATEWAY_API_VERSION} ..."

# Use the experimental channel: includes TCPRoute and UDPRoute CRDs that
# Contour v1.32.1 requires. The standard channel omits them.
# --server-side is required: some CRDs (e.g. httproutes) exceed the 262144-byte
# annotation limit imposed by client-side apply.
kubectl apply --server-side --force-conflicts -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/experimental-install.yaml"

# Contour v1.32.1 tries to watch TLSRoute/v1alpha2 and BackendTLSPolicy/v1alpha3,
# but Gateway API v1.5.1 ships with these versions as served=false. Patch them.
info "Patching Gateway API CRDs for Contour ${CONTOUR_VERSION} compatibility ..."
for crd in tlsroutes.gateway.networking.k8s.io backendtlspolicies.gateway.networking.k8s.io; do
  kubectl get crd "$crd" -o json \
    | jq '.spec.versions = (.spec.versions | map(if .name == "v1alpha2" or .name == "v1alpha3" then .served = true else . end))' \
    | kubectl apply -f -
done

success "Gateway API CRDs installed and patched."

# -----------------------------------------------------------------------------
# Contour Gateway Provisioner (handles Gateway API HTTPRoute resources)
# Runs alongside the standard Contour install — they do not conflict.
# -----------------------------------------------------------------------------

info "Installing Contour Gateway Provisioner ${CONTOUR_VERSION} ..."

# Use --force-conflicts: the provisioner bundles older Gateway API CRDs (v1.2.1)
# which conflict with the v1.5.1 CRDs already installed above. The CRD apply
# errors are expected and harmless — the Deployment is what matters here.
kubectl apply --force-conflicts -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour-gateway-provisioner.yaml" 2>&1 \
  | grep -v "^Error from server" || true

wait_for_rollout "${CONTOUR_NAMESPACE}" deployment/contour-gateway-provisioner

success "Contour Gateway Provisioner installed."

# -----------------------------------------------------------------------------
# GatewayClass (tells the provisioner which Gateways it owns)
# -----------------------------------------------------------------------------

info "Applying GatewayClass ..."
kubectl apply -f deploy/gateway/gatewayclass.yaml
success "GatewayClass applied."

# -----------------------------------------------------------------------------
# Gateway resource (HTTP listener — no TLS required for local testing)
# Placed in its own namespace so the provisioner's Contour+Envoy pods don't
# land in the default application namespace.
# allowedRoutes.namespaces.from: All lets HTTPRoutes from any namespace attach.
# -----------------------------------------------------------------------------

info "Creating Gateway namespace and Gateway ..."
kubectl create namespace "${GATEWAY_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f - <<EOF
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: gateway
  namespace: ${GATEWAY_NAMESPACE}
spec:
  gatewayClassName: contour
  listeners:
    - name: http
      port: 80
      protocol: HTTP
      allowedRoutes:
        namespaces:
          from: All
EOF

info "Waiting for Gateway to become Programmed ..."
for i in \$(seq 1 30); do
  status=\$(kubectl get gateway gateway -n "${GATEWAY_NAMESPACE}" -o jsonpath='{.status.conditions[?(@.type=="Programmed")].status}' 2>/dev/null || true)
  if [ "\$status" = "True" ]; then
    success "Gateway is Programmed."
    break
  fi
  echo "  ... waiting (\${i}/30)"
  sleep 5
done

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

echo ""
echo "============================================"
echo " Infrastructure ready"
echo "============================================"
echo " Agones:              v${AGONES_VERSION}  (${AGONES_NAMESPACE})"
echo " cert-manager:        ${CERT_MANAGER_VERSION}  (${CERT_MANAGER_NAMESPACE})"
echo " Contour (ingress):   ${CONTOUR_VERSION}  (${CONTOUR_NAMESPACE})"
echo " Contour (gateway):   ${CONTOUR_VERSION}  (${CONTOUR_NAMESPACE})"
echo " Gateway API CRDs:    ${GATEWAY_API_VERSION}"
echo " Gateway:             gateway (${GATEWAY_NAMESPACE} namespace, HTTP :80)"
echo "   Envoy svc:         envoy-gateway.${GATEWAY_NAMESPACE} — port-forward to test locally"
echo "============================================"
echo ""
echo "Next steps:"
echo "  Ingress backend:"
echo "    1. Apply an issuer:   kubectl apply -f deploy/cert-manager/<issuer>.yaml"
echo "    2. Deploy controller: kubectl apply -f deploy/install.yaml"
echo ""
echo "  Gateway backend:"
echo "    1. Deploy controller: kubectl apply -f deploy/install.yaml"
echo "    2. Deploy a fleet:    kubectl apply -f examples/gateway/fleet-domain.yaml"
echo "       or:                kubectl apply -f examples/gateway/fleet-path.yaml"
