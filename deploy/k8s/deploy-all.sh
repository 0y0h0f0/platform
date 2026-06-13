#!/usr/bin/env bash
set -euo pipefail

# ============================================================
# One-click deploy all services (frontend + backend) to K8s
# ============================================================
#
# Usage:
#   ./deploy/k8s/deploy-all.sh [dev|prod]            # full deploy
#   ./deploy/k8s/deploy-all.sh dev --build            # build images + deploy
#   ./deploy/k8s/deploy-all.sh dev --skip-infra       # skip postgres/redis, deploy app+observability only
#   ./deploy/k8s/deploy-all.sh prod --dry-run         # preview manifests
#
# Prerequisites:
#   - kubectl installed and configured
#   - Docker (if --build is used)
#   - A container registry (if not using minikube/kind)

ENV="${1:-dev}"
BUILD="${2:-}"
DRY_RUN="${2:-}"
SKIP_INFRA="${2:-}"

# Handle combined flags from $2
for arg in "$@"; do
  case "$arg" in
    --build)   BUILD=true ;;
    --dry-run) DRY_RUN=true ;;
    --skip-infra) SKIP_INFRA=true ;;
  esac
done

REPO="task-platform"
OVERLAY="deploy/k8s/overlays/${ENV}"
KUBECTL_OPTS=""
if [ "$DRY_RUN" = "true" ]; then
  KUBECTL_OPTS="--dry-run=client"
  echo ">>> DRY RUN MODE (no changes applied) <<<"
  echo ""
fi

# ---------- Step 1: Build images ----------
if [ "$BUILD" = "true" ]; then
  echo "=== Step 1: Building Docker images ==="

  echo "  -> api-gateway"
  docker build --build-arg SERVICE=api-gateway -t ${REPO}/api-gateway:latest .

  echo "  -> user-service"
  docker build --build-arg SERVICE=user-service -t ${REPO}/user-service:latest .

  echo "  -> task-service"
  docker build --build-arg SERVICE=task-service -t ${REPO}/task-service:latest .

  echo "  -> web (frontend)"
  docker build -t ${REPO}/web:latest web/

  echo "  All images built."
  echo ""
fi

# ---------- Step 2: Deploy ----------
if [ "$DRY_RUN" = "true" ]; then
  echo "=== Previewing manifests ==="
  kubectl apply -k "$OVERLAY" $KUBECTL_OPTS
  echo ""
  echo "=== If this looks correct, run: ==="
  echo "  ./deploy/k8s/deploy-all.sh $ENV"
  exit 0
fi

echo "=== Step 2: Deploying to Kubernetes ($ENV) ==="
kubectl apply -k "$OVERLAY"

echo ""
echo "=== Step 3: Waiting for all Pods to be ready ==="
echo "    (this may take 2-5 minutes on first deploy)"
kubectl wait --for=condition=Ready pods --all -n task-platform --timeout=300s 2>/dev/null || true

echo ""
echo "=== Deployment Complete ==="
echo ""
echo "--- Access URLs ---"

# Detect cluster type for access hints
if command -v minikube &>/dev/null && minikube status &>/dev/null 2>&1; then
  MINIKUBE_IP=$(minikube ip 2>/dev/null || echo "MINIKUBE_IP")
  echo "  Frontend:    http://${MINIKUBE_IP}:$(kubectl get svc web -n task-platform -o jsonpath='{.spec.ports[0].nodePort}' 2>/dev/null || echo '?')"
  echo "  or:          minikube service web -n task-platform"
elif command -v kubectl &>/dev/null && kubectl config current-context 2>/dev/null | grep -q 'kind-'; then
  echo "  Frontend:    kubectl port-forward svc/web 8080:80 -n task-platform"
  echo "               then open http://localhost:8080"
else
  echo "  Frontend:    kubectl port-forward svc/web 8080:80 -n task-platform"
  echo "               then open http://localhost:8080"
fi

echo "  Grafana:     kubectl port-forward svc/grafana 3000:3000 -n task-platform"
echo "  Jaeger:      kubectl port-forward svc/jaeger 16686:16686 -n task-platform"
echo ""
echo "--- Status ---"
kubectl get pods -n task-platform
echo ""
echo "--- Quick Test ---"
echo "  curl http://localhost:8080/api/v1/healthz  (after port-forward)"
echo "  curl http://localhost:8080/                 (frontend SPA)"
