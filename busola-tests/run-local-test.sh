#!/usr/bin/env bash

# Complete local E2E test runner for BTP Operator Busola extension
# Usage: ./run-local-test.sh [options]
#
# Options:
#   --busola-path PATH    Path to busola repo (default: ../busola)
#   --skip-cluster        Skip k3d cluster creation
#   --skip-busola         Skip Busola build and start
#   --headed              Run Cypress in headed mode (visible browser)
#   --interactive         Open Cypress GUI
#   --cleanup             Cleanup after tests
#   -h, --help            Show this help

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BTP_MANAGER_ROOT="$(dirname "$SCRIPT_DIR")"

# Default values
BUSOLA_PATH="${BTP_MANAGER_ROOT}/../busola"
SKIP_CLUSTER=false
SKIP_BUSOLA=false
HEADED=false
INTERACTIVE=false
CLEANUP=false
CLUSTER_NAME="kyma"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --busola-path)
      BUSOLA_PATH="$2"
      shift 2
      ;;
    --skip-cluster)
      SKIP_CLUSTER=true
      shift
      ;;
    --skip-busola)
      SKIP_BUSOLA=true
      shift
      ;;
    --headed)
      HEADED=true
      shift
      ;;
    --interactive)
      INTERACTIVE=true
      shift
      ;;
    --cleanup)
      CLEANUP=true
      shift
      ;;
    -h|--help)
      cat << EOF
Usage: $0 [options]

Complete local E2E test runner for BTP Operator Busola extension

Options:
  --busola-path PATH    Path to busola repo (default: ../busola)
  --skip-cluster        Skip k3d cluster creation
  --skip-busola         Skip Busola build and start
  --headed              Run Cypress in headed mode (visible browser)
  --interactive         Open Cypress GUI instead of running tests
  --cleanup             Cleanup k3d cluster after tests
  -h, --help            Show this help

Examples:
  # Full run with cleanup
  $0 --cleanup

  # Interactive debugging (skip cluster/busola if already running)
  $0 --skip-cluster --skip-busola --interactive

  # Quick rerun after changes
  $0 --skip-cluster --skip-busola --headed
EOF
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      exit 1
      ;;
  esac
done

echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  BTP Operator Busola Extension - Local E2E Test Runner    ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Step 1: Create k3d cluster
if [ "$SKIP_CLUSTER" = false ]; then
  echo -e "${GREEN}[1/6] Creating k3d cluster '${CLUSTER_NAME}'...${NC}"
  
  # Check if cluster exists
  if k3d cluster list | grep -q "^${CLUSTER_NAME}"; then
    echo -e "${YELLOW}   ⚠ Cluster '${CLUSTER_NAME}' already exists. Deleting...${NC}"
    k3d cluster delete ${CLUSTER_NAME}
  fi
  
  k3d cluster create ${CLUSTER_NAME} \
    --agents 1 \
    --port 80:80@loadbalancer \
    --port 443:443@loadbalancer \
    --wait
  
  echo -e "${GREEN}   ✓ Cluster created${NC}"
else
  echo -e "${YELLOW}[1/6] Skipping cluster creation${NC}"
fi

# Step 2: Install BTP Operator prerequisites
echo -e "${GREEN}[2/6] Installing BTP Operator prerequisites...${NC}"

# Create kyma-system namespace
kubectl create namespace kyma-system --dry-run=client -o yaml | kubectl apply -f -
echo -e "${GREEN}   ✓ Namespace kyma-system created${NC}"

# Install BtpOperator CRD
kubectl apply -f "${BTP_MANAGER_ROOT}/config/crd/bases/operator.kyma-project.io_btpoperators.yaml"
echo -e "${GREEN}   ✓ BtpOperator CRD installed${NC}"

# Install SAP BTP Service Operator CRDs (ServiceInstance, ServiceBinding)
kubectl apply -f "${BTP_MANAGER_ROOT}/module-chart/chart/templates/crd.yml"
echo -e "${GREEN}   ✓ ServiceInstance and ServiceBinding CRDs installed${NC}"

# Create BtpOperator CR
kubectl apply -f "${BTP_MANAGER_ROOT}/examples/btp-operator.yaml"
echo -e "${GREEN}   ✓ BtpOperator CR created${NC}"

# Create mock BTP secret
kubectl apply -f "${BTP_MANAGER_ROOT}/busola-tests/fixtures/mock-btp-secret.yaml"
echo -e "${GREEN}   ✓ Mock BTP secret created${NC}"

# Wait a moment for resources to settle
sleep 2

echo -e "${GREEN}   ✓ Prerequisites installed${NC}"
kubectl get btpoperators -A
kubectl get secrets -n kyma-system sap-btp-manager

# Step 3: Setup Busola
if [ "$SKIP_BUSOLA" = false ]; then
  echo -e "${GREEN}[3/6] Setting up Busola...${NC}"
  
  if [ ! -d "$BUSOLA_PATH" ]; then
    echo -e "${RED}   ✗ Busola not found at $BUSOLA_PATH${NC}"
    echo -e "${YELLOW}   Clone it with: git clone https://github.com/kyma-project/busola.git $BUSOLA_PATH${NC}"
    exit 1
  fi
  
  cd "$BUSOLA_PATH"
  
  # Check if Busola is already running
  if curl -s http://localhost:3001 > /dev/null 2>&1; then
    echo -e "${YELLOW}   ⚠ Busola already running on http://localhost:3001${NC}"
  else
    echo -e "${YELLOW}   Building and starting Busola (this may take a few minutes)...${NC}"
    .github/scripts/setup_local_busola.sh > busola.log 2>&1
    
    # Verify Busola is running
    if curl -s http://localhost:3001 > /dev/null 2>&1; then
      echo -e "${GREEN}   ✓ Busola is running on http://localhost:3001${NC}"
    else
      echo -e "${RED}   ✗ Busola failed to start. Check busola.log${NC}"
      tail -20 busola.log
      exit 1
    fi
  fi
  
  cd "$BTP_MANAGER_ROOT"
else
  echo -e "${YELLOW}[3/6] Skipping Busola setup${NC}"
  
  # Verify Busola is running
  if ! curl -s http://localhost:3001 > /dev/null 2>&1; then
    echo -e "${RED}   ✗ Busola is not running on http://localhost:3001${NC}"
    echo -e "${YELLOW}   Remove --skip-busola flag or start Busola manually${NC}"
    exit 1
  fi
  echo -e "${GREEN}   ✓ Busola is running${NC}"
fi

# Step 4: Inject test files into Busola
echo -e "${GREEN}[4/6] Injecting test files into Busola...${NC}"

"${SCRIPT_DIR}/inject-to-busola.sh" "$BUSOLA_PATH"

if [ $? -ne 0 ]; then
  echo -e "${RED}   ✗ Failed to inject test files${NC}"
  exit 1
fi

# Step 5: Generate kubeconfig
echo -e "${GREEN}[5/6] Generating kubeconfig...${NC}"

k3d kubeconfig get ${CLUSTER_NAME} > "${BUSOLA_PATH}/tests/integration/fixtures/kubeconfig.yaml"
echo -e "${GREEN}   ✓ Kubeconfig generated${NC}"

# Step 6: Run Cypress tests
echo -e "${GREEN}[6/6] Running Cypress tests...${NC}"
cd "${BUSOLA_PATH}/tests/integration"

if [ "$INTERACTIVE" = true ]; then
  echo -e "${YELLOW}   Opening Cypress GUI...${NC}"
  CYPRESS_DOMAIN=http://localhost:3001 npx cypress open
elif [ "$HEADED" = true ]; then
  echo -e "${YELLOW}   Running tests in headed mode...${NC}"
  CYPRESS_DOMAIN=http://localhost:3001 npx cypress run \
    --spec "tests/ext-test-btp-operator.spec.js" \
    --browser chrome \
    --headed
else
  echo -e "${YELLOW}   Running tests in headless mode...${NC}"
  CYPRESS_DOMAIN=http://localhost:3001 npx cypress run \
    --spec "tests/ext-test-btp-operator.spec.js" \
    --browser chrome
fi

TEST_EXIT_CODE=$?

# Cleanup
if [ "$CLEANUP" = true ]; then
  echo ""
  echo -e "${GREEN}Cleaning up...${NC}"
  k3d cluster delete ${CLUSTER_NAME}
  echo -e "${GREEN}   ✓ Cluster deleted${NC}"
fi

# Summary
echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════╗${NC}"
if [ $TEST_EXIT_CODE -eq 0 ]; then
  echo -e "${BLUE}║${GREEN}  ✓ Tests PASSED                                             ${BLUE}║${NC}"
else
  echo -e "${BLUE}║${RED}  ✗ Tests FAILED                                             ${BLUE}║${NC}"
fi
echo -e "${BLUE}╚════════════════════════════════════════════════════════════╝${NC}"
echo ""

if [ $TEST_EXIT_CODE -ne 0 ]; then
  echo -e "${YELLOW}Check test artifacts:${NC}"
  echo -e "  Videos:      ${BUSOLA_PATH}/tests/integration/cypress/videos/"
  echo -e "  Screenshots: ${BUSOLA_PATH}/tests/integration/cypress/screenshots/"
fi

exit $TEST_EXIT_CODE
