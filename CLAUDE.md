# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```text
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

## 5. Project-Specific Guidelines

BTP Manager is a Kubernetes operator (built with Kubebuilder/controller-runtime) that manages the SAP BTP service operator lifecycle within Kyma clusters. It reconciles a `BtpOperator` CRD in the `kyma-system` namespace and orchestrates Helm chart deployment, secrets validation, certificate management, and resource cleanup.

### Common Commands

#### Build & Generate
```bash
make build          # Build the manager binary to bin/manager
make run            # Run controller locally (requires KUBECONFIG)
make generate       # Regenerate DeepCopy methods (after API changes)
make manifests      # Regenerate CRDs, RBAC, webhooks (after API/marker changes)
```

#### Testing
```bash
make test           # Run all tests using envtest (no cluster needed)
make test-docs      # Validate condition reasons match documentation
```

To run a single test or subset using Ginkgo labels:
```bash
. ./scripts/testing/set-env-vars.sh
GINKGO_LABEL_FILTER="<label>" ./bin/ginkgo controllers
```

To run with verbose output or increased timeouts (useful when debugging):
```bash
USE_EXISTING_CLUSTER=true SUITE_TIMEOUT=180s SINGLE_TEST_TIMEOUT=60s make test
```

#### Code Quality
```bash
make fmt            # go fmt ./...
make vet            # go vet ./...
make go-lint        # Run golangci-lint (config: .golangci.yml)
make fix            # go mod tidy + golangci-lint --fix
```

#### Deployment
```bash
make install        # Install CRDs into the cluster
make deploy         # Deploy controller to the cluster
make docker-build   # Build Docker image (runs tests first)
make module-image   # Build and push module image
```

### Architecture

#### Core Components

**Entry point:** `main.go` initializes the controller-runtime manager with leader election, registers reconcilers, sets up health/metrics probes, and starts config watching.

**Primary reconciler:** `controllers/btpoperator_controller.go` (~2600 lines) — contains the main reconciliation logic. It owns the BtpOperator state machine and drives all provisioning, updating, and deprovisioning work.

**Supporting reconcilers:**
- `controllers/serviceinstance_controller.go` — watches ServiceInstance/ServiceBinding resources to trigger BtpOperator status updates when instances or bindings change.
- `controllers/instance_binding_controller_manager.go` — controls lifecycle of the SISB (ServiceInstance/ServiceBinding) cleanup controller, disabling it during deletion and enabling it after provisioning.
- `controllers/config/handler.go` — watches a ConfigMap for dynamic config changes.

#### State Machine

The `BtpOperator` CR flows through these states:

```
[Empty] → Processing → Ready
                ↓
         Warning / Error  (transient or permanent failures)
                ↓
            Deleting → [finalizer removed, CR gone]
```

Each state has a dedicated handler method (`HandleInitialState`, `HandleProcessingState`, `HandleReadyState`, `HandleWarningState`, `HandleErrorState`, `HandleDeletingState`) dispatched from `Reconcile()`.

#### Provisioning Flow (HandleProcessingState)

1. Fetch and validate the `sap-btp-manager` secret (must contain `clientid`, `clientsecret`, `sm_url`, `tokenurl`, `cluster_id`).
2. Check credentials namespace and cluster ID consistency across secrets/configmaps.
3. Delete outdated resources from `module-resources/delete/`.
4. Apply resources from `module-resources/apply/`.
5. Enable the SISB controller.
6. Transition to Ready.

#### Deletion Flow (HandleDeletingState)

1. Disable SISB controller.
2. Attempt hard delete of operator resources (with graceful timeout for active instances/bindings).
3. Fall back to soft delete if hard delete fails.
4. Remove the finalizer (`operator.kyma-project.io/btp-manager`).

#### Key Directories

| Directory | Purpose |
|---|---|
| `api/v1alpha1/` | BtpOperator CRD type definitions |
| `controllers/` | All reconcilers and config handler |
| `internal/` | Utilities: `certs/`, `conditions/`, `manifest/`, `metrics/`, `ymlutils/`, `gvksutils/` |
| `module-chart/` | Helm chart for the SAP BTP operator that BTP Manager deploys |
| `module-resources/apply/` | K8s manifests applied during provisioning |
| `module-resources/delete/` | K8s manifests deleted during provisioning (outdated resources) |
| `manager-resources/` | BTP Manager-specific resources (e.g. network policies) |
| `config/` | Kustomize manifests for deploying BTP Manager itself |
| `cmd/autodoc/` | Tool that validates condition reason constants match documentation |
| `scripts/testing/` | Test environment setup (`set-env-vars.sh`, envtest, e2e scripts) |

#### Conditions System

Conditions track reconciliation progress via the Kubernetes conditions API. Reason constants are defined in `internal/conditions/conditions.go` and are validated against inline documentation by `make test-docs`. When adding or changing condition reasons, update the documentation comments in that file.

#### Testing Structure

Tests are organized by concern alongside the controllers they test:
- `controllers/btpoperator_controller_provisioning_test.go`
- `controllers/btpoperator_controller_deprovisioning_test.go`
- `controllers/btpoperator_controller_updating_test.go`
- `controllers/btpoperator_controller_certificates_test.go`
- `controllers/btpoperator_controller_network_policies_test.go`
- `controllers/btpoperator_controller_configuration_test.go`
- `controllers/btpoperator_controller_secret_customization_test.go`

Shared test setup (envtest bootstrapping, scheme registration, client creation) lives in `controllers/suite_test.go`. Test helpers live in `controllers/utils_test.go`.

Tests use Ginkgo v2 (BDD-style) with Gomega assertions against a local envtest Kubernetes environment (k8s 1.25.0). Key test environment variables are sourced from `scripts/testing/set-env-vars.sh`.

### Claude Skills

Skills for Claude Code are stored in `.claude/skills/`. Each skill is a directory containing a `SKILL.md` file.

Available skills:
- **`commit`** — Drafts and creates a git commit following BTP Manager commit message conventions.
- **`create-pr`** — Creates a pull request from a fork branch to `kyma-project/btp-manager:main`.
- **`review-pr`** — Reviews a PR against BTP Manager conventions.

**Maintenance:** Any PR that changes project structure, adds a new pattern, or modifies the build/test workflow must update `CLAUDE.md` and the relevant skills in the same PR. This keeps Claude Code's assistance accurate as the project evolves.
