#!/usr/bin/env bash

# Script to inject BTP Operator test into local Busola repository
# Usage: ./inject-to-busola.sh [path-to-busola-repo]
#
# If no path provided, assumes busola is cloned at ../busola relative to btp-manager

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get script directory (busola-tests/)
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BTP_MANAGER_ROOT="$(dirname "$SCRIPT_DIR")"

# Busola path
BUSOLA_PATH="${1:-$BTP_MANAGER_ROOT/../busola}"

if [ ! -d "$BUSOLA_PATH" ]; then
  echo -e "${RED}Error: Busola directory not found at $BUSOLA_PATH${NC}"
  echo "Usage: $0 [path-to-busola-repo]"
  exit 1
fi

if [ ! -d "$BUSOLA_PATH/tests/integration" ]; then
  echo -e "${RED}Error: Invalid Busola repository - tests/integration directory not found${NC}"
  exit 1
fi

echo -e "${YELLOW}Injecting BTP Operator test files into Busola...${NC}"
echo "Busola path: $BUSOLA_PATH"
echo ""

# 1. Copy extension ConfigMap
echo -e "${GREEN}1. Copying extension ConfigMap...${NC}"
cp "$BTP_MANAGER_ROOT/config/busola-extension/sap-btp-operator-extension.yaml" \
   "$BUSOLA_PATH/tests/integration/fixtures/"
echo "   ✓ Copied to fixtures/sap-btp-operator-extension.yaml"

# 2. Copy test spec
echo -e "${GREEN}2. Copying test spec...${NC}"
cp "$SCRIPT_DIR/ext-test-btp-operator.spec.js" \
   "$BUSOLA_PATH/tests/integration/tests/"
echo "   ✓ Copied to tests/ext-test-btp-operator.spec.js"

# 3. Update cypress.config.js
echo -e "${GREEN}3. Updating cypress.config.js...${NC}"

# Check if test already exists in config
if grep -q "tests/ext-test-btp-operator.spec.js" "$BUSOLA_PATH/tests/integration/cypress.config.js"; then
  echo "   ⚠ Test already exists in cypress.config.js"
else
  # Add test to specPattern array (using Python for reliability)
  python3 -c "
import re
config_file = '$BUSOLA_PATH/tests/integration/cypress.config.js'
with open(config_file, 'r') as f:
    content = f.read()
# Add our test after companion-feedback-dialog test
content = re.sub(
    r\"(tests/companion/test-companion-feedback-dialog\.spec\.js',)\",
    r\"\1\\n      'tests/ext-test-btp-operator.spec.js',\",
    content
)
with open(config_file, 'w') as f:
    f.write(content)
  "
  
  # Verify it was added
  if grep -q "tests/ext-test-btp-operator.spec.js" "$BUSOLA_PATH/tests/integration/cypress.config.js"; then
    echo "   ✓ Added test to specPattern array"
  else
    echo -e "   ${RED}✗ Failed to add test to cypress.config.js${NC}"
    exit 1
  fi
fi

# 4. Verify files
echo -e "${GREEN}4. Verifying injected files...${NC}"
if [ -f "$BUSOLA_PATH/tests/integration/fixtures/sap-btp-operator-extension.yaml" ]; then
  echo "   ✓ Extension ConfigMap present"
else
  echo -e "   ${RED}✗ Extension ConfigMap missing${NC}"
  exit 1
fi

if [ -f "$BUSOLA_PATH/tests/integration/tests/ext-test-btp-operator.spec.js" ]; then
  echo "   ✓ Test spec present"
else
  echo -e "   ${RED}✗ Test spec missing${NC}"
  exit 1
fi

echo ""
echo -e "${GREEN}✓ Successfully injected BTP Operator test files into Busola!${NC}"
echo ""
echo "Next steps:"
echo "  1. Generate kubeconfig:"
echo "     k3d kubeconfig get kyma > $BUSOLA_PATH/tests/integration/fixtures/kubeconfig.yaml"
echo ""
echo "  2. Run the test:"
echo "     cd $BUSOLA_PATH/tests/integration"
echo "     CYPRESS_DOMAIN=http://localhost:3001 npx cypress run --spec 'tests/ext-test-btp-operator.spec.js' --browser chrome"
echo ""
echo "  Or open Cypress UI:"
echo "     CYPRESS_DOMAIN=http://localhost:3001 npx cypress open"
