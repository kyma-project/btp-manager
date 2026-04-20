# BTP Operator Busola Extension E2E Tests

Cypress E2E tests for the BTP Operator Busola extension. Tests run against a local k3d cluster and a locally built Busola instance — no external cluster or credentials required.

## Prerequisites

- [k3d](https://k3d.io/#installation) v5.6.0+
- kubectl
- Node.js v18+ and npm
- Python 3
- Chrome

## Run Tests

To run all tests from the repository root, use:

```shell
./busola-tests/run-local-test.sh --cleanup
```

The script creates a k3d cluster, installs prerequisites, builds and starts Busola, injects the extension and test files, and runs Cypress. The `--cleanup` flag deletes the cluster after the run.

## Script Options

| Option | Description |
|--------|-------------|
| `--busola-path PATH` | Path to local Busola repository (default: `../busola`) |
| `--skip-cluster` | Reuse existing k3d cluster `kyma` |
| `--skip-busola` | Skip Busola build/start (Busola must already be running on `:3001`) |
| `--headed` | Run Cypress with a visible browser window |
| `--interactive` | Open Cypress GUI for manual test selection |
| `--cleanup` | Delete k3d cluster after tests complete |

## Develop Tests Iteratively

For iterative development, run a full setup once and reuse the cluster and Busola instance on subsequent runs:

```shell
# First run — sets everything up and opens Cypress GUI
./busola-tests/run-local-test.sh --interactive

# Subsequent runs — skip cluster and Busola setup
./busola-tests/run-local-test.sh --skip-cluster --skip-busola --headed
```

To test extension changes without re-running the full suite, apply the updated ConfigMap to the running cluster and rerun:

```shell
kubectl apply -f config/busola-extension/sap-btp-operator-extension.yaml -n kube-public
./busola-tests/run-local-test.sh --skip-cluster --skip-busola --headed
```

## Test Scenarios

All scenarios are in `ext-test-btp-operator.spec.js`.

**1. Upload Extension ConfigMap**

Uploads the extension ConfigMap, verifies the **BTP Operators** menu appears, opens the `btpoperator` detail view, and validates:
- The Metadata card shows the Documentation link, Service Instances count, and Service Bindings count
- The **BTP Operator Secrets** panel renders with **BTP Manager Secret** (Managed badge) and **SAP BTP Service Operator Secret** (Inherited badge)
- Credentials Namespace defaults to `kyma-system`
- The **Edit** ResourceLink navigates to the `sap-btp-manager` secret
- The Service Instances and Service Bindings count links navigate to the respective CRD pages

**2. Configure Custom Credentials Namespace**

Creates a `test` namespace with a namespace-based secret, edits `sap-btp-manager` to add the `kyma-project.io/skip-reconciliation` label and set `credentials_namespace: test`, then validates:
- The **BTP Manager Secret** badge switches to Unmanaged
- Credentials Namespace shows `test`
- The **Namespace-Based Secrets** table shows the test secret with In Use status
- After uploading a ServiceInstance and ServiceBinding, counts update in the header

**3. Custom Secrets**

Creates a ServiceInstance with `spec.btpAccessCredentialsSecret` set, then validates:
- The **Custom Secrets** panel renders with the referenced secret
- Status shows Not in Use (the secret namespace differs from the credentials namespace)
- The Service Instances count is correct

## CI

Tests run automatically on pull requests that modify `config/busola-extension/**`, `busola-tests/**`, `config/crd/**`, `examples/btp-operator.yaml`, or the workflow file itself.

See [`.github/workflows/btp-operator-e2e.yaml`](../.github/workflows/btp-operator-e2e.yaml) for the full workflow definition. On failure, Cypress videos, screenshots, and Busola logs are uploaded as artifacts with a 7-day retention period.
