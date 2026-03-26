#!/usr/bin/env bash
set -euo pipefail

AGONES_VERSION="1.56.0"
CERT_MANAGER_VERSION="v1.17.1"
CONTOUR_VERSION="1.32.1"

AGONES_NAMESPACE="agones-system"
CERT_MANAGER_NAMESPACE="cert-manager"
CONTOUR_NAMESPACE="projectcontour"

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

require kubectl helm

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
# Contour ingress controller
# -----------------------------------------------------------------------------

info "Installing Contour ${CONTOUR_VERSION} ..."

kubectl apply -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour.yaml"

kubectl rollout status deployment/contour -n "${CONTOUR_NAMESPACE}" --timeout=300s
kubectl rollout status daemonset/envoy -n "${CONTOUR_NAMESPACE}" --timeout=300s

success "Contour installed."

# -----------------------------------------------------------------------------
# Summary
# -----------------------------------------------------------------------------

echo ""
echo "============================================"
echo " Infrastructure ready"
echo "============================================"
echo " Agones:        v${AGONES_VERSION}  (${AGONES_NAMESPACE})"
echo " cert-manager:  ${CERT_MANAGER_VERSION}  (${CERT_MANAGER_NAMESPACE})"
echo " Contour:       ${CONTOUR_VERSION}  (${CONTOUR_NAMESPACE})"
echo "============================================"
echo ""
echo "Next steps:"
echo "  1. Apply an issuer:   kubectl apply -f deploy/cert-manager/<issuer>.yaml"
echo "  2. Deploy controller: kubectl apply -f deploy/install.yaml"
