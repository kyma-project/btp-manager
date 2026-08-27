---
name: create-pr
description: Creates a pull request from a fork branch to kyma-project/btp-manager main.
---

# create-pr

Create a pull request from the current fork branch to the upstream `kyma-project/btp-manager` repository.

## Usage

```
/create-pr [optional hint about what changed]
```

**Examples:**
- `/create-pr`
- `/create-pr Add network policy reconciliation`

---

## What to do

### 1. Confirm intent

Ask the user:

> You are about to open a PR from branch `<current-branch>` to `kyma-project/btp-manager:main`. Continue?

Stop if they say no.

### 2. Gather changes

The local `main` branch may be stale. Always diff against the upstream remote to get only the changes this branch uniquely introduces:

```bash
git fetch upstream main
git diff upstream/main...HEAD
```

The three-dot syntax (`...`) diffs from the merge-base, so only commits unique to this branch are shown — not unrelated upstream commits that the local branch happens to include.

If no `upstream` remote exists, fall back to `origin/main`:

```bash
git fetch origin main
git diff origin/main...HEAD
```

Read the diff carefully and summarize the most important changes as bullet points. Base the summary entirely on what the code actually changed.

### 3. Draft the PR body

Read `.github/pull-request-template.md` and use it as the body structure. Fill in the **Description** bullets from the diff.

Rules:
- Never include `Closes #<issue>`, `Fixes #<issue>`, or any issue-closing keywords unless the user explicitly asks.
- Keep bullets factual and concise — derived from the actual diff, not paraphrased vaguely.
- Start each bullet with an infinitive verb (e.g. "Add...", "Rename...", "Update...", "Remove...", "Fix...").

### 4. Ask about related issues

Ask the user:

> Would you like to reference any related issues? (e.g. `See also #123`) If yes, provide the issue number(s).

If yes, place the reference on the line immediately after `**Related issue(s)**` with no blank line between them. Use `See also #<n>` unless the user explicitly asks for a closing keyword.

### 5. Ask about the changelog label

Ask the user which changelog label to apply:

> Which changelog label should this PR have?
> - `kind/feature` — New feature
> - `kind/enhancement` — Enhancement to an existing feature
> - `kind/bug` — Bug fix

Apply exactly one of these labels. No other labels unless the user explicitly requests them.

### 6. Draft the PR title

Derive the title from the diff:

- Imperative mood, title case, no trailing period, ≤72 chars.
- No `feat:`, `fix:`, `chore:` prefixes.
- Examples: `Add Network Policy Reconciliation`, `Fix Deprovisioning Race Condition`

Show the user the full title + body and ask: "Shall I open the PR with this title and description?"

### 7. Create the PR

To determine `<fork-owner>` and the PR creator's GitHub username, run:

```bash
git remote get-url origin
gh api user --jq .login
```

Extract `<fork-owner>` from the origin URL. The PR creator's GitHub username comes from `gh api user`.

```bash
gh pr create \
  --repo kyma-project/btp-manager \
  --base main \
  --head <fork-owner>:<current-branch> \
  --title "<title>" \
  --label "<chosen-label>" \
  --assignee "<github-username>" \
  --body "$(cat <<'EOF'
<body>
EOF
)"
```

Return the PR URL to the user once created.

---

## Rules

- Always target `--base main` on `kyma-project/btp-manager` — never an upstream feature branch.
- Never add issue-closing keywords (`Closes`, `Fixes`, `Resolves`) unless the user explicitly asks.
- Never use `--no-verify` or bypass any git hooks.
- Apply exactly one `kind/*` label per PR.
