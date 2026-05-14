---
name: review-pr
description: Reviews a PR against BTP Manager conventions (reconciler patterns, conditions API, state machine, test structure, etc.).
---

# review-pr

Review a pull request with BTP Manager-specific context in mind.

## Usage

```
/review-pr <PR number or URL>
```

**Examples:**
- `/review-pr 1234`
- `/review-pr https://github.com/kyma-project/btp-manager/pull/1234`

---

## What to do

Fetch the PR diff and review it through the lens of BTP Manager conventions. Structure your review as:

### Summary

One paragraph: what the PR does and whether the approach makes sense.

### Issues (if any)

List only real problems — not style nits unless they violate a hard rule. For each issue:
- **File:line** — description of the problem
- Severity: `blocking` | `suggestion`

### BTP Manager checklist

Evaluate every item below. **Only print items that have a note or failure** — skip items that pass cleanly. If everything passes, write a single line: `BTP Manager checklist: all clear`.

Use this format for printed items only:
- ⚠️ **[category]** — description of the note
- ❌ **[category]** — description of the failure

**Reconciler / state machine (if controllers/ is touched):**
- New state handlers follow the existing pattern: dedicated `Handle*State` method dispatched from `Reconcile()`
- Status updates go through `UpdateBtpOperatorStatus()` — not direct client patches
- Finalizer (`operator.kyma-project.io/btp-manager`) is only added/removed in the correct lifecycle phases
- Requeue intervals use the configured values (`processingStateRequeueInterval`, `readyStateRequeueInterval`) — no hardcoded `time.Duration` literals

**Conditions (if internal/conditions/ or status conditions are touched):**
- New condition reasons are defined as constants in `internal/conditions/conditions.go`
- Documentation comment for each new reason is present — `make test-docs` would pass
- Reasons map to the correct state (Ready/Processing/Error/Warning/Deleting)

**Resource management (if module-resources/ or manifest handling is touched):**
- Resources to apply go in `module-resources/apply/`, resources to delete go in `module-resources/delete/`
- No direct `client.Apply` / `client.Delete` calls that bypass the manifest reconciliation path

**Tests:**
- Tests use Ginkgo v2 + Gomega (not `testify`)
- New controller tests are placed in the appropriately named file (`btpoperator_controller_<concern>_test.go`)
- Shared setup uses `controllers/suite_test.go` — no duplicate envtest bootstrapping
- `make test` would pass (envtest env vars sourced from `scripts/testing/set-env-vars.sh`)

**Code generation (if api/v1alpha1/ or controller markers are touched):**
- `make generate` and `make manifests` were run — generated files (`zz_generated.deepcopy.go`, CRD YAMLs) are up to date

**Documentation:**
- `CLAUDE.md` updated if project structure, conventions, or build/test workflow changed

**General:**
- No speculative abstractions or features beyond the PR scope
- Imports cleaned up (no unused imports left from changes)
- No backwards-compatibility shims for removed code

### Verdict

`Approve` / `Request changes` / `Comment` — with one sentence explaining why.

---

## Tone

- Be direct and specific. Point to file and line numbers.
- Don't praise code that is merely correct — save positive comments for non-obvious good decisions.
- Don't suggest improvements outside the PR scope.
