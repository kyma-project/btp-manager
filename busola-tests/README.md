# BTP Operator Busola Extension E2E Tests

This directory contains automated E2E test scenarios for the BTP Operator Busola extension. Tests run in an isolated k3d Kubernetes cluster with a local Busola dashboard instance, validating extension functionality without external dependencies.

## Why This Approach?

**Before**: Tests required access to `dev.kyma.cloud.sap` cluster via `DEV_KUBECONFIG` secret, creating external dependencies and maintenance overhead.

**Now**: 
- ✅ Fully isolated testing in ephemeral k3d clusters
- ✅ No external cluster dependencies
- ✅ Reproducible locally and in CI/CD
- ✅ Complete automation via single script
- ✅ Interactive debugging support

## Quick Start

Run tests locally with one command:

```bash
# Full automated run with cleanup
./busola-tests/run-local-test.sh --cleanup

# Interactive debugging with Cypress GUI
./busola-tests/run-local-test.sh --interactive --cleanup
```

**First time?** Script will automatically:
1. Create k3d cluster `kyma` 
2. Install all Kubernetes prerequisites (CRDs, namespaces, secrets)
3. Clone/build Busola (if needed)
4. Inject extension and test files
5. Run Cypress tests
6. Clean up (with `--cleanup` flag)

## Directory Structure

```
busola-tests/
├── run-local-test.sh                 # 🚀 Main automation script (all-in-one)
├── ext-test-btp-operator.spec.js    # Cypress E2E test specification
├── fixtures/
│   └── mock-btp-secret.yaml         # Mock BTP credentials for testing
└── README.md                         # This file
```

## Script Usage

### run-local-test.sh

Complete automation for local E2E testing. Handles cluster creation, prerequisites installation, Busola setup, and test execution.

**Options:**
- `--busola-path PATH` - Custom Busola repo path (default: `../busola`)
- `--skip-cluster` - Skip k3d cluster creation (reuse existing)
- `--skip-busola` - Skip Busola build/start (must be running on :3001)
- `--headed` - Run Cypress with visible browser window
- `--interactive` - Open Cypress GUI for manual test selection
- `--cleanup` - Delete k3d cluster after tests complete
- `-h, --help` - Show detailed help

**Examples:**

```bash
# Standard full run with cleanup
./busola-tests/run-local-test.sh --cleanup

# Debug failing test with visible browser
./busola-tests/run-local-test.sh --headed

# Interactive test development (GUI mode)
./busola-tests/run-local-test.sh --interactive

# Fast iteration (reuse cluster & Busola after first run)
./busola-tests/run-local-test.sh --skip-cluster --skip-busola --headed

# Use custom Busola location
./busola-tests/run-local-test.sh --busola-path ~/projects/busola --cleanup
```

## Test Scenarios

### ext-test-btp-operator.spec.js

Comprehensive Cypress test covering BTP Operator extension functionality in Busola UI.

**Test Coverage:**

1. **Extension Loading & Navigation**
   - Uploads BTP Operator extension ConfigMap to cluster
   - Verifies "BTP Operators" menu item appears in Busola navigation
   - Validates extension renders correctly in dashboard

2. **Default State Validation**
   - Checks BtpOperator CR status display
   - Verifies default credentials namespace (`kyma-system`)
   - Validates Service Instance and Binding counts
   - Tests internal resource links (secrets, CRDs)

3. **Custom Namespace Configuration**
   - Creates custom namespace with test secret
   - Updates `sap-btp-manager` secret to reference custom namespace
   - Adds `kyma-project.io/skip-reconciliation` label
   - Validates extension updates UI accordingly
   - Verifies status badges reflect configuration changes

4. **Resource Creation UI**
   - Tests "Create Service Instance" form
   - Tests "Create Service Binding" form
   - Validates form validation and user flows

**Test Pattern:**
- Uses `cy.loginAndSelectCluster()` for authentication
- Leverages Busola's built-in support commands
- Runs headless by default, supports `--headed` and `--interactive` modes

## Fixtures

### mock-btp-secret.yaml

Mock SAP BTP Manager credentials secret installed in k3d cluster before tests run.

**Purpose**: Satisfies extension's `$btpSecret()` dataSource requirement, enabling UI panels and features.

**Contents** (base64-encoded):
- `clientid` - Mock SAP BTP client ID
- `clientsecret` - Mock client secret
- `sm_url` - Mock Service Manager URL (`https://mock-sm.example.com`)
- `tokenurl` - Mock token endpoint URL (`https://mock-auth.example.com/oauth/token`)
- `cluster_id` - Mock cluster identifier

**Labels**:
- `app.kubernetes.io/managed-by: kcp-kyma-environment-broker` - Matches production secret pattern

**Note**: These are test credentials only. No actual SAP BTP connection is made.

## CI/CD Workflow Integration

Tests run automatically in GitHub Actions on pull requests that modify:
- `config/busola-extension/**` - Extension ConfigMap changes
- `busola-tests/**` - Test scenario changes
- `config/crd/**` - BtpOperator CRD changes
- `examples/btp-operator.yaml` - CR definition changes

**Workflow Steps** (see [.github/workflows/btp-operator-e2e.yaml](../.github/workflows/btp-operator-e2e.yaml)):

1. **Setup Environment**
   - Checkout `btp-manager` repository
   - Checkout `busola` repository (latest main branch)
   - Install k3d v5.8.3, setup Node.js 24 with npm cache

2. **Create Isolated Cluster**
   - `k3d cluster create kyma --agents 1 --port 80:80 --port 443:443`
   - Install BtpOperator CRD from `config/crd/bases/`
   - Install SAP BTP Service Operator CRDs from `module-chart/chart/templates/crd.yml`
   - Create `kyma-system` namespace
   - Apply BtpOperator CR from `examples/btp-operator.yaml`
   - Apply mock secret from `busola-tests/fixtures/mock-btp-secret.yaml`

3. **Inject Test Files**
   - Copy extension ConfigMap to Busola fixtures
   - Copy test spec to Busola tests directory
   - Update `cypress.config.js` to include test spec (Python-based injection)

4. **Run Tests**
   - Build Busola with `.github/scripts/setup_local_busola.sh`
   - Generate kubeconfig for cluster
   - Execute Cypress: `npx cypress run --spec "tests/ext-test-btp-operator.spec.js"`

5. **Collect Artifacts** (on failure)
   - Upload Cypress videos (7-day retention)
   - Upload Cypress screenshots (7-day retention)
   - Upload Busola logs (7-day retention)

**Benefits:**
- No external cluster dependencies
- Consistent test environment
- Fast feedback on PRs (~10-15 minutes)
- Full artifact collection for debugging

## Local Testing Workflow

### Prerequisites

Before running tests locally, ensure you have:

- **k3d** (v5.6.0+) - [Installation guide](https://k3d.io/#installation)
- **kubectl** - Kubernetes CLI
- **Node.js** (v18+) and npm
- **Python 3** - For config file manipulation
- **Chrome browser** - Cypress default

### First-Time Setup

1. **Clone Busola** (if not already cloned):
   ```bash
   cd /Users/I767610/Documents/repositories
   git clone https://github.com/kyma-project/busola.git
   cd btp-manager
   ```

2. **Run tests**:
   ```bash
   ./busola-tests/run-local-test.sh --cleanup
   ```

That's it! The script handles everything else automatically.

### Development Workflow

**Option 1: Full automated run** (recommended for CI validation)
```bash
./busola-tests/run-local-test.sh --cleanup
```
- Creates fresh cluster
- Installs all prerequisites
- Builds Busola
- Runs tests headlessly
- Cleans up cluster

**Option 2: Interactive debugging** (recommended for test development)
```bash
# First run - setup everything
./busola-tests/run-local-test.sh --interactive

# In Cypress GUI:
# - Select test file
# - Run with live reload
# - Inspect each step

# After changes - quick rerun (reuses cluster & Busola)
./busola-tests/run-local-test.sh --skip-cluster --skip-busola --headed
```

**Option 3: Watch mode** (recommended for extension development)
```bash
# Terminal 1: Keep cluster running
./busola-tests/run-local-test.sh --interactive
# Keep Cypress GUI open

# Terminal 2: Make changes to extension
vim config/busola-extension/sap-btp-operator-extension.yaml

# Terminal 3: Reapply and retest
kubectl apply -f config/busola-extension/sap-btp-operator-extension.yaml
# Rerun test in Cypress GUI
```

### What the Script Does

The `run-local-test.sh` script automates these steps:

1. **Cluster Setup**
   - Creates k3d cluster named `kyma` with port forwarding (80, 443)
   - Or reuses existing cluster (with `--skip-cluster`)

2. **Prerequisites Installation**
   - Creates `kyma-system` namespace
   - Installs BtpOperator CRD (`operator.kyma-project.io/btpoperators`)
   - Installs SAP BTP Service Operator CRDs (`services.cloud.sap.com/serviceinstances`, `servicebindings`)
   - Creates BtpOperator CR named `btpoperator`
   - Applies mock BTP secret with test credentials

3. **Busola Setup**
   - Checks if Busola is running on `http://localhost:3001`
   - If not, builds and starts Busola (via `.github/scripts/setup_local_busola.sh`)
   - Or skips build (with `--skip-busola`)

4. **Test Injection**
   - Copies extension ConfigMap to Busola fixtures
   - Copies test spec to Busola tests directory
   - Updates `cypress.config.js` specPattern array (Python-based)

5. **Test Execution**
   - Generates kubeconfig from k3d cluster
   - Runs Cypress in selected mode (headless/headed/interactive)
   - Collects videos and screenshots on failure

6. **Cleanup** (optional)
   - Deletes k3d cluster (with `--cleanup` flag)

### Advanced: Manual Step-by-Step

If you need to run steps individually (e.g., for debugging the script itself):

```bash
# 1. Create cluster
k3d cluster create kyma --agents 1 --port 80:80@loadbalancer --port 443:443@loadbalancer

# 2. Install prerequisites
kubectl create namespace kyma-system
kubectl apply -f config/crd/bases/operator.kyma-project.io_btpoperators.yaml
kubectl apply -f module-chart/chart/templates/crd.yml
kubectl apply -f examples/btp-operator.yaml
kubectl apply -f busola-tests/fixtures/mock-btp-secret.yaml

# 3. Setup Busola (in busola repo)
cd ../busola
.github/scripts/setup_local_busola.sh &
# Wait for http://localhost:3001

# 4. Inject files
cd ../btp-manager
k3d kubeconfig get kyma > ../busola/tests/integration/fixtures/kubeconfig.yaml
cp config/busola-extension/sap-btp-operator-extension.yaml ../busola/tests/integration/fixtures/
cp busola-tests/ext-test-btp-operator.spec.js ../busola/tests/integration/tests/

# 5. Update Cypress config
cd ../busola/tests/integration
python3 -c "
import re
with open('cypress.config.js', 'r') as f:
    content = f.read()
content = re.sub(
    r\"(tests/companion/test-companion-feedback-dialog\.spec\.js',)\",
    r\"\1\\n      'tests/ext-test-btp-operator.spec.js',\",
    content
)
with open('cypress.config.js', 'w') as f:
    f.write(content)
"

# 6. Run tests
CYPRESS_DOMAIN=http://localhost:3001 npx cypress run --spec "tests/ext-test-btp-operator.spec.js"

# 7. Cleanup
k3d cluster delete kyma
```

## Extending Tests

When adding new test scenarios to `ext-test-btp-operator.spec.js`:

### Code Guidelines

**1. Use Busola test utilities**
```javascript
import config from '../config';
// Access config.domain instead of hardcoding URLs

cy.loginAndSelectCluster({fileName: 'kubeconfig.yaml'});
// Don't navigate to URLs manually
```

**2. Resource navigation patterns**
```javascript
// Good: Use Busola's navigation commands
cy.navigateTo('Namespaces', 'kyma-system');
cy.clickGenericListLink('sap-btp-manager');

// Avoid: Direct URL navigation
// cy.visit(`${config.domain}/cluster/kyma/namespaces/kyma-system/...`)
```

**3. Wait strategies**
```javascript
// Good: Wait for specific conditions
cy.contains('BTP Operators').should('be.visible');
cy.get('[data-testid="status-badge"]').should('contain', 'Ready');

// Avoid: Arbitrary sleeps
// cy.wait(5000) // Don't do this
```

**4. Clean up after tests**
```javascript
after(() => {
  // Delete test resources
  cy.deleteInDetails('Namespace', 'test-namespace');
});
```

### Test Structure

Follow this pattern for new test cases:

```javascript
context('Feature Name', () => {
  before(() => {
    // Setup: Create resources
  });

  it('should validate behavior', () => {
    // Test steps
  });

  after(() => {
    // Cleanup: Remove test resources
  });
});
```

## Cluster Prerequisites

The following Kubernetes resources must exist before tests run:

| Resource Type | Name | Namespace | Source File |
|--------------|------|-----------|-------------|
| Namespace | `kyma-system` | - | created by script |
| CRD | `btpoperators.operator.kyma-project.io` | - | [config/crd/bases/](../config/crd/bases/) |
| CRD | `serviceinstances.services.cloud.sap.com` | - | [module-chart/chart/templates/crd.yml](../module-chart/chart/templates/crd.yml) |
| CRD | `servicebindings.services.cloud.sap.com` | - | [module-chart/chart/templates/crd.yml](../module-chart/chart/templates/crd.yml) |
| BtpOperator CR | `btpoperator` | `kyma-system` | [examples/btp-operator.yaml](../examples/btp-operator.yaml) |
| Secret | `sap-btp-manager` | `kyma-system` | [busola-tests/fixtures/mock-btp-secret.yaml](fixtures/mock-btp-secret.yaml) |

All prerequisites are installed automatically by `run-local-test.sh` and the CI/CD workflow.

## Troubleshooting

### Common Issues

**❌ "Busola not found at ../busola"**
```bash
# Solution: Clone Busola repository
cd ..
git clone https://github.com/kyma-project/busola.git
cd btp-manager
./busola-tests/run-local-test.sh
```

**❌ "Cluster 'kyma' already exists"**
```bash
# Solution: Script will auto-delete and recreate
# Or manually delete:
k3d cluster delete kyma
```

**❌ "Busola failed to start"**
```bash
# Check Busola logs
cat ../busola/busola.log

# Common issues:
# - Port 3001 already in use: Kill process using port
# - npm install failed: Clear busola/node_modules and retry
# - Missing dependencies: Install Node.js 18+
```

**❌ Test fails with "Extension not loaded"**
```bash
# Verify extension ConfigMap exists
kubectl get cm -n kube-public | grep btp-operator

# Verify BtpOperator CR exists
kubectl get btpoperators -A

# Check Busola can see the extension (in browser):
# http://localhost:3001 → Extensions → Should show BTP Operator
```

**❌ Test fails with "Login failed"**
```bash
# Verify kubeconfig was generated
ls -la ../busola/tests/integration/fixtures/kubeconfig.yaml

# Verify k3d cluster is accessible
kubectl cluster-info

# Check cluster context
k3d kubeconfig get kyma | grep server
# Should show: https://0.0.0.0:XXXXX
```

**❌ "Secret not found" errors**
```bash
# Verify mock secret exists
kubectl get secret sap-btp-manager -n kyma-system

# Reapply if missing
kubectl apply -f busola-tests/fixtures/mock-btp-secret.yaml
```

**❌ "ServiceInstance CRD not found"**
```bash
# Verify SAP BTP Service Operator CRDs installed
kubectl get crd | grep services.cloud.sap.com

# Should show:
# servicebindings.services.cloud.sap.com
# serviceinstances.services.cloud.sap.com

# Reinstall if missing
kubectl apply -f module-chart/chart/templates/crd.yml
```

**❌ Cypress fails with timeout**
```bash
# Increase timeout and run in headed mode to see what's happening
./busola-tests/run-local-test.sh --skip-cluster --skip-busola --headed

# Check if elements are loading slowly
# Look for network errors in browser console
```

### Debug Mode

For detailed debugging:

```bash
# Run with headed browser to see test execution
./busola-tests/run-local-test.sh --headed

# Or use interactive mode to step through manually
./busola-tests/run-local-test.sh --interactive

# Check test artifacts after failure
ls ../busola/tests/integration/cypress/videos/
ls ../busola/tests/integration/cypress/screenshots/
```

### Getting Help

**Check test artifacts:**
- Videos: `busola/tests/integration/cypress/videos/`
- Screenshots: `busola/tests/integration/cypress/screenshots/`
- Busola logs: `busola/busola.log`

**Verify environment:**
```bash
# Check versions
k3d version    # Should be 5.6.0+
kubectl version --client
node --version  # Should be 18+
python3 --version

# Check running processes
ps aux | grep -E "busola|k3d|cypress"

# Check ports
lsof -i :3001  # Busola
lsof -i :6443  # k3d API server
```

## CI/CD Artifacts

When tests fail in GitHub Actions, the following artifacts are uploaded for 7 days:

- **Cypress Videos** - Full test execution recordings
- **Cypress Screenshots** - Screenshots of failed assertions
- **Busola Logs** - Console output from Busola build/startup

Access artifacts from the workflow run page → "Artifacts" section.

## Related Files

| File | Purpose |
|------|---------|
| [config/busola-extension/sap-btp-operator-extension.yaml](../config/busola-extension/sap-btp-operator-extension.yaml) | Extension ConfigMap definition (UI panels, datasources, forms) |
| [.github/workflows/btp-operator-e2e.yaml](../.github/workflows/btp-operator-e2e.yaml) | GitHub Actions workflow for automated testing |
| [config/crd/bases/](../config/crd/bases/) | BtpOperator CRD definition |
| [module-chart/chart/templates/crd.yml](../module-chart/chart/templates/crd.yml) | SAP BTP Service Operator CRDs (ServiceInstance, ServiceBinding) |
| [examples/btp-operator.yaml](../examples/btp-operator.yaml) | Example BtpOperator CR used in tests |

## Architecture Decision

**Why k3d + Busola?**

Previously, tests depended on `dev.kyma.cloud.sap` external cluster requiring:
- `DEV_KUBECONFIG` secret management
- Network access to dev environment
- Coupling with external cluster state
- Maintenance overhead when dev environment changes

**New approach benefits:**
- ✅ **Isolated**: Each test run gets fresh cluster
- ✅ **Reproducible**: Same environment locally and in CI
- ✅ **Fast**: No external dependencies or network latency
- ✅ **Debuggable**: Run tests locally with `--interactive` mode
- ✅ **Maintainable**: Single script handles all setup
- ✅ **Cost-effective**: No permanent test cluster needed

**Trade-offs:**
- ⚠️ Tests run against mock BTP credentials (no real SAP BTP validation)
- ⚠️ ServiceInstance/Binding creation may fail validation (expected - UI behavior is tested)
- ⏱️ Initial Busola build takes ~5-10 minutes (cached in subsequent runs)

For real SAP BTP integration testing, separate integration test suite is recommended.
