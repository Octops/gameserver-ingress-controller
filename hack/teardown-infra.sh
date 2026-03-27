#!/usr/bin/env bash
set -euo pipefail

AGONES_VERSION="1.56.0"
CONTOUR_VERSION="1.32.1"
GATEWAY_API_VERSION="v1.5.1"

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

info "Deleting Gateway and HTTPRoutes ..."
kubectl delete gateway gateway -n default --ignore-not-found 2>/dev/null || true
kubectl delete httproutes --all -n default --ignore-not-found 2>/dev/null || true
success "Gateway resources deleted."

info "Deleting Fleets and GameServers ..."
kubectl delete fleets --all -n default --ignore-not-found 2>/dev/null || true
kubectl delete gameservers --all -n default --ignore-not-found 2>/dev/null || true
success "Fleets and GameServers deleted."

info "Deleting GatewayClass ..."
kubectl delete gatewayclass contour --ignore-not-found 2>/dev/null || true
success "GatewayClass deleted."

info "Uninstalling Contour Gateway Provisioner ..."
kubectl delete -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour-gateway-provisioner.yaml" --ignore-not-found 2>/dev/null || true
success "Contour Gateway Provisioner uninstalled."

info "Uninstalling Contour (ingress) ..."
kubectl delete -f "https://raw.githubusercontent.com/projectcontour/contour/v${CONTOUR_VERSION}/examples/render/contour.yaml" --ignore-not-found 2>/dev/null || true
success "Contour uninstalled."

info "Removing LoadBalancer finalizers (required on Docker Desktop) ..."
kubectl patch service envoy -n "${CONTOUR_NAMESPACE}" \
  -p '{"metadata":{"finalizers":[]}}' --type=merge 2>/dev/null || true
success "LoadBalancer finalizers cleared."

info "Uninstalling Agones ..."
helm uninstall agones -n "${AGONES_NAMESPACE}" --ignore-not-found 2>/dev/null || true
success "Agones uninstalled."

info "Uninstalling cert-manager ..."
helm uninstall cert-manager -n "${CERT_MANAGER_NAMESPACE}" --ignore-not-found 2>/dev/null || true
success "cert-manager uninstalled."

info "Removing Gateway API CRDs ..."
kubectl delete -f "https://github.com/kubernetes-sigs/gateway-api/releases/download/${GATEWAY_API_VERSION}/experimental-install.yaml" --ignore-not-found 2>/dev/null || true
success "Gateway API CRDs removed."

info "Deleting namespaces ..."
kubectl delete namespace \
  "${AGONES_NAMESPACE}" \
  "${CERT_MANAGER_NAMESPACE}" \
  "${CONTOUR_NAMESPACE}" \
  --ignore-not-found --wait=false

# Namespaces can get stuck in Terminating when services have load-balancer
# finalizers or when API groups from deleted CRDs are still referenced.
# Force-finalize any that are stuck.
info "Force-finalizing any stuck namespaces ..."
for ns in "${AGONES_NAMESPACE}" "${CERT_MANAGER_NAMESPACE}" "${CONTOUR_NAMESPACE}"; do
  phase=$(kubectl get namespace "$ns" -o jsonpath='{.status.phase}' 2>/dev/null || true)
  if [ "$phase" = "Terminating" ]; then
    info "Force-finalizing namespace $ns ..."
    kubectl get namespace "$ns" -o json \
      | jq '.spec.finalizers = []' \
      | kubectl replace --raw "/api/v1/namespaces/${ns}/finalize" -f - &>/dev/null || true
  fi
done

# Wait up to 30s for namespaces to disappear
for i in $(seq 1 6); do
  remaining=0
  for ns in "${AGONES_NAMESPACE}" "${CERT_MANAGER_NAMESPACE}" "${CONTOUR_NAMESPACE}"; do
    kubectl get namespace "$ns" &>/dev/null && remaining=$((remaining+1)) || true
  done
  [ "$remaining" -eq 0 ] && break
  sleep 5
done

success "Namespaces deleted."

echo ""
echo "============================================"
echo " Teardown complete"
echo "============================================"
echo " Run hack/setup-infra.sh to start fresh."
