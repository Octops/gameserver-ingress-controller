#!/usr/bin/env bash
set -euo pipefail

AGONES_VERSION="1.56.0"
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

# -----------------------------------------------------------------------------
# Preflight
# -----------------------------------------------------------------------------

require kubectl helm

kubectl cluster-info &>/dev/null || die "Cannot reach the cluster. Check your kubeconfig."

# -----------------------------------------------------------------------------
# Teardown
# -----------------------------------------------------------------------------

info "Uninstalling Agones ..."
helm uninstall agones -n "${AGONES_NAMESPACE}" --ignore-not-found 2>/dev/null || true
success "Agones uninstalled."

info "Uninstalling cert-manager ..."
helm uninstall cert-manager -n "${CERT_MANAGER_NAMESPACE}" --ignore-not-found 2>/dev/null || true
success "cert-manager uninstalled."

info "Uninstalling Contour ..."
kubectl delete -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour.yaml" --ignore-not-found 2>/dev/null || true
success "Contour uninstalled."

info "Removing LoadBalancer finalizers (required on Docker Desktop) ..."
kubectl patch service envoy -n "${CONTOUR_NAMESPACE}" \
  -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null || true
success "LoadBalancer finalizers cleared."

info "Deleting namespaces ..."
kubectl delete namespace \
  "${AGONES_NAMESPACE}" \
  "${CERT_MANAGER_NAMESPACE}" \
  "${CONTOUR_NAMESPACE}" \
  --ignore-not-found

success "Namespaces deleted."

echo ""
echo "============================================"
echo " Teardown complete"
echo "============================================"
echo " Run hack/setup-infra.sh to start fresh."
