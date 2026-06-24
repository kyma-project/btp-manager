---
name: commit
description: Drafts and creates a git commit following BTP Manager commit message conventions.
---

# commit

Create a git commit for staged or unstaged changes in the BTP Manager repository.

## Usage

```
/commit [optional hint about what changed]
```

**Examples:**
- `/commit`
- `/commit Add network policy reconciliation`
- `/commit Fix deprovisioning race condition`

---

## What to do

1. **Check current state** — run `git status` and `git diff` (staged + unstaged) to understand what changed. If nothing has changed, say so and stop.

2. **Stage changes** — if files are unstaged and the user didn't already stage them, ask which files to include before staging. Never run `git add -A` or `git add .` without asking — that can accidentally commit test data, `.env` files, or generated files.

3. **Draft the commit message** using the conventions below.

4. **Show the message** to the user for approval before committing. Ask: "Shall I commit with this message?"

5. **Commit** once confirmed.

6. **Ask about pushing** — after a successful commit, ask: "Would you like to push the changes to `<current-branch>`?" If yes, run `git push -u origin HEAD`. If no, ask which branch to push to and run `git push -u origin HEAD:<target-branch>`.

---

## Commit message conventions

```
<Short summary>

[optional body — only if the why isn't obvious from the diff]
```

**Short summary:** imperative mood, title case, no trailing period, ≤72 chars.

**Examples:**
```
Add network policy reconciliation
Fix deprovisioning race condition
Handle missing cluster ID gracefully
Add unit tests for BtpOperator provisioning flow
Update CLAUDE.md with state machine documentation
Bump golang to 1.26.3-alpine3.22
```

---

## Rules

- Never commit files that look like secrets (`.env`, `credentials*.json`, `*_key.pem`, etc.) — warn the user if such files are staged.
- Never use `--no-verify`.
- Never amend published commits. Create a new commit instead.
- If a pre-commit hook fails, fix the issue and create a **new** commit — do not `--amend`.
