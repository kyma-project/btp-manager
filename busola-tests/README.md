# BTP Operator Busola Extension E2E Tests

This directory contains E2E test scenarios for the BTP Operator Busola extension. These tests are injected into the Busola repository during CI/CD workflow execution to validate the extension's functionality in an isolated k3d cluster environment.

## Overview

The tests in this directory are designed to run against a local Busola instance with a k3d Kubernetes cluster. The workflow automatically:
1. Creates a k3d cluster
2. Clones the Busola repository
3. Injects extension ConfigMap and test scenarios from this directory
4. Builds and runs Busola locally
5. Executes the Cypress E2E tests

## Directory Structure

```
busola-tests/
├── ext-test-btp-operator.spec.js    # Main E2E test specification
├── fixtures/
│   └── mock-btp-secret.yaml         # Mock BTP secret for testing
└── README.md                         # This file
```

## Test Scenarios

### ext-test-btp-operator.spec.js

Tests the BTP Operator extension functionality including:

1. **Extension Upload and Verification**
   - Uploads the BTP Operator extension ConfigMap
   - Verifies the "BTP Operators" menu appears in Busola
   - Checks default state (status, credentials namespace, service instances)
   - Tests internal resource links (secrets, CRDs)

2. **Custom Credentials Namespace Configuration**
   - Creates a test namespace with custom secret
   - Modifies the `sap-btp-manager` secret to use custom namespace
   - Adds `kyma-project.io/skip-reconciliation` label
   - Verifies the extension shows correct status badges
   - Tests Service Instance and Binding creation

## Fixtures

### mock-btp-secret.yaml

A mock SAP BTP Manager secret with test credentials. This secret is installed in the k3d cluster before tests run to satisfy the extension's data source requirements.

**Contains:**
- `clientid`: Mock client ID
- `clientsecret`: Mock client secret
- `sm_url`: Mock Service Manager URL
- `tokenurl`: Mock token URL
- `cluster_id`: Mock cluster ID

All values are base64-encoded as required by Kubernetes secrets.

## Workflow Integration

The test files are used in [.github/workflows/btp-operator-e2e.yaml](../.github/workflows/btp-operator-e2e.yaml):

1. **Checkout btp-manager** → Clone this repository
2. **Setup k3d cluster** → Create isolated test cluster
3. **Checkout Busola** → Clone Busola repository to `./busola`
4. **Inject extension** → Copy `config/busola-extension/sap-btp-operator-extension.yaml` to Busola fixtures
5. **Inject test** → Copy `busola-tests/ext-test-btp-operator.spec.js` to Busola tests
6. **Update Cypress config** → Add test to `cypress.config.js` specPattern array
7. **Install prerequisites** → Apply CRDs, BtpOperator CR, mock secret
8. **Build Busola** → Run Busola setup script
9. **Run tests** → Execute Cypress with `CYPRESS_DOMAIN=http://localhost:3001`

## Local Testing

To run tests locally:

### Quick start (using helper script)

```bash
# 1. Create k3d cluster
k3d cluster create kyma --agents 1 --port 80:80@loadbalancer --port 443:443@loadbalancer

# 2. Install prerequisites
kubectl create namespace kyma-system
kubectl apply -f config/crd/bases/operator.kyma-project.io_btpoperators.yaml
kubectl apply -f examples/btp-operator.yaml
kubectl apply -f busola-tests/fixtures/mock-btp-secret.yaml

# 3. Clone and setup Busola (if not already cloned)
git clone https://github.com/kyma-project/busola.git ../busola
cd ../busola
.github/scripts/setup_local_busola.sh &
# Wait for Busola to start (check http://localhost:3001)

# 4. Inject test files using helper script
cd ../btp-manager
./busola-tests/inject-to-busola.sh ../busola

# 5. Generate kubeconfig and run tests
k3d kubeconfig get kyma > ../busola/tests/integration/fixtures/kubeconfig.yaml
cd ../busola/tests/integration
npm ci
CYPRESS_DOMAIN=http://localhost:3001 cypress run --spec "tests/ext-test-btp-operator.spec.js" --browser chrome

# 6. Cleanup
k3d cluster delete kyma
```

### Manual step-by-step

```bash
# 1. Create k3d cluster
k3d cluster create kyma --agents 1 --port 80:80@loadbalancer --port 443:443@loadbalancer

# 2. Install prerequisites
kubectl create namespace kyma-system
kubectl apply -k config/crd
kubectl apply -f examples/btp-operator.yaml
kubectl apply -f busola-tests/fixtures/mock-btp-secret.yaml

# 3. Clone and setup Busola
git clone https://github.com/kyma-project/busola.git
cd busola
.github/scripts/setup_local_busola.sh &

# Wait for Busola to start (check http://localhost:3001)

# 4. Inject test files
k3d kubeconfig get kyma > tests/integration/fixtures/kubeconfig.yaml
cp ../config/busola-extension/sap-btp-operator-extension.yaml tests/integration/fixtures/
cp ../busola-tests/ext-test-btp-operator.spec.js tests/integration/tests/

# Add test to cypress.config.js specPattern (using Python for reliability)
python3 -c "
import re
with open('tests/integration/cypress.config.js', 'r') as f:
    content = f.read()
content = re.sub(
    r\"(tests/companion/test-companion-feedback-dialog\.spec\.js',)\",
    r\"\1\\n      'tests/ext-test-btp-operator.spec.js',\",
    content
)
with open('tests/integration/cypress.config.js', 'w') as f:
    f.write(content)
"

# 5. Run tests
cd tests/integration
npm ci
CYPRESS_DOMAIN=http://localhost:3001 cypress run --spec "tests/ext-test-btp-operator.spec.js" --browser chrome

# 6. Cleanup
k3d cluster delete kyma
```

## Test Modification Guidelines

When modifying tests:

1. **Use Busola test helpers**: Tests use cy.* commands from Busola's support files (login-commands.js, navigate-to.js, etc.)
2. **Import config**: Always use `import config from '../config'` for CYPRESS_DOMAIN
3. **Login pattern**: Use `cy.loginAndSelectCluster({fileName: 'kubeconfig.yaml'})` 
4. **No hardcoded URLs**: All URLs should use `config.domain`
5. **Wait times**: Use reasonable waits (avoid excessive cy.wait() calls)
6. **Cleanup**: Tests should not leave orphaned resources

## Prerequisites

The following resources must exist in the cluster before tests run:

- **Namespace**: `kyma-system`
- **CRD**: `btpoperators.operator.kyma-project.io`
- **CR**: `BtpOperator` named `btpoperator` in `kyma-system`
- **Secret**: `sap-btp-manager` in `kyma-system` (from mock-btp-secret.yaml)
- **CRDs**: `serviceinstances.services.cloud.sap.com`, `servicebindings.services.cloud.sap.com`

## CI/CD Artifacts

On test failure, the workflow uploads:
- Cypress videos: `busola/tests/integration/cypress/videos`
- Cypress screenshots: `busola/tests/integration/cypress/screenshots`

Retention: 7 days

## Related Files

- Extension ConfigMap: [config/busola-extension/sap-btp-operator-extension.yaml](../config/busola-extension/sap-btp-operator-extension.yaml)
- Workflow definition: [.github/workflows/btp-operator-e2e.yaml](../.github/workflows/btp-operator-e2e.yaml)
- BtpOperator CRD: [config/crd/bases/](../config/crd/bases/)
- Example BtpOperator CR: [examples/btp-operator.yaml](../examples/btp-operator.yaml)

## Troubleshooting

**Test fails with "Extension not loaded"**
- Ensure extension ConfigMap was uploaded successfully
- Check if `kyma-system` namespace exists
- Verify BtpOperator CR is present and in Ready state

**Login fails**
- Verify kubeconfig.yaml is in `busola/tests/integration/fixtures/`
- Check if Busola is running on http://localhost:3001
- Ensure k3d cluster is accessible

**Secret tests fail**
- Check if mock-btp-secret.yaml was applied
- Verify secret data is properly base64-encoded
- Confirm secret exists in `kyma-system` namespace

**Service Instance/Binding tests fail**
- These resources may fail webhook validation in k3d (expected)
- Tests verify UI behavior, not actual SAP BTP integration
- Check if SAP BTP Service Operator CRDs are installed
