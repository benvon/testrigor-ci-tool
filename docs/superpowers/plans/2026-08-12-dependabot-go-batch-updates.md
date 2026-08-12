# Dependabot Go Batching + Toolchain Refresh Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Batch Dependabot Go updates per module directory (version + security), bump the toolchain to Go 1.26.5, refresh all root and tools dependencies, and land one green PR that supersedes the open one-off Dependabot PRs.

**Architecture:** Dependabot gets dual catch-all groups (`go-modules` for version-updates, `go-security` for security-updates) on both `/` and `/tools`. Toolchain pins move to 1.26.5, then both `go.mod` graphs are fully upgraded and verified with `make ci` + `make security`.

**Tech Stack:** Go 1.26.5, Dependabot `gomod`, `tools/go.mod` tool directive, Makefile (`make ci`, `make security`), GitHub CLI (`gh`).

## Global Constraints

- Go toolchain target: `1.26.5` (verbatim in `go.mod`, `tools/go.mod`, `.tool-versions`)
- `.golangci.yml` may keep `go: '1.26'` (minor pin is fine)
- Dependabot commit-message prefix must remain `chore` with `include: scope`
- Do not change `github-actions` Dependabot config
- Do not change reusable workflows / release-policy workflows
- Prefer fixing forward on dependency breakages; only pin if a major bump is intentionally deferred
- Work on branch `chore/dependabot-go-batch-updates` based on current `origin/main`
- Close Dependabot PRs #37, #38, #39 before opening the replacement PR
- Final PR title: `chore(deps): batch go module updates and bump to 1.26.5`

## File map

| File | Responsibility |
|------|----------------|
| `.github/dependabot.yml` | Batching rules for `/` and `/tools` gomod ecosystems |
| `go.mod` / `go.sum` | App module toolchain + dependency graph |
| `tools/go.mod` / `tools/go.sum` | Dev-tool module toolchain + dependency graph |
| `.tool-versions` | mise Go version pin |
| Application `.go` files | Only if dependency upgrades force API fixes |

---

### Task 1: Close open Dependabot PRs

**Files:**
- None (GitHub PR state only)

**Interfaces:**
- Consumes: open PRs #37, #38, #39
- Produces: those PRs closed so the replacement branch has a clear path

- [ ] **Step 1: Confirm the three PRs are still open**

```bash
gh pr view 37 --json number,title,state,url
gh pr view 38 --json number,title,state,url
gh pr view 39 --json number,title,state,url
```

Expected: each shows `"state":"OPEN"`.

- [ ] **Step 2: Close them with an explanatory comment**

```bash
gh pr close 37 --comment "Superseded by chore/dependabot-go-batch-updates: batching Go Dependabot updates and refreshing tools/root deps in one PR."
gh pr close 38 --comment "Superseded by chore/dependabot-go-batch-updates: batching Go Dependabot updates and refreshing tools/root deps in one PR."
gh pr close 39 --comment "Superseded by chore/dependabot-go-batch-updates: batching Go Dependabot updates and refreshing tools/root deps in one PR."
```

Expected: each command reports the PR was closed.

- [ ] **Step 3: Verify closed**

```bash
gh pr list --author "app/dependabot" --state open --json number,title
```

Expected: empty list `[]` (or no #37/#38/#39).

- [ ] **Step 4: Commit note**

No repo files changed; no commit for this task.

---

### Task 2: Update Dependabot config for batched Go updates

**Files:**
- Modify: `.github/dependabot.yml`

**Interfaces:**
- Consumes: existing root `gomod` + `github-actions` entries
- Produces: `/` and `/tools` gomod entries each with `go-modules` + `go-security` groups

- [ ] **Step 1: Replace `.github/dependabot.yml` with the full target contents**

Write this exact file:

```yaml
version: 2
updates:
  # Enable version updates for Go (app module)
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    labels:
      - "dependencies"
      - "go"
    commit-message:
      prefix: "chore"
      include: "scope"
    groups:
      go-modules:
        applies-to: version-updates
        patterns:
          - "*"
      go-security:
        applies-to: security-updates
        patterns:
          - "*"

  # Enable version updates for Go (tools module)
  - package-ecosystem: "gomod"
    directory: "/tools"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    labels:
      - "dependencies"
      - "go"
    commit-message:
      prefix: "chore"
      include: "scope"
    groups:
      go-modules:
        applies-to: version-updates
        patterns:
          - "*"
      go-security:
        applies-to: security-updates
        patterns:
          - "*"

  # Enable version updates for GitHub Actions
  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    open-pull-requests-limit: 10
    labels:
      - "dependencies"
      - "github-actions"
    commit-message:
      prefix: "chore"
      include: "scope"
```

- [ ] **Step 2: Sanity-check YAML structure**

```bash
python3 -c 'import yaml; yaml.safe_load(open(".github/dependabot.yml")); print("ok")'
```

Expected: `ok`

If `yaml` is missing:

```bash
python3 -c 'import json,sys; print("skip-pyyaml")'
rg -n "directory: \"/tools\"|go-security|applies-to: security-updates" .github/dependabot.yml
```

Expected: matches for `/tools`, `go-security`, and `security-updates`.

- [ ] **Step 3: Commit**

```bash
git add .github/dependabot.yml
git commit -m "$(cat <<'EOF'
chore(deps): batch Dependabot Go module updates

Group version and security updates per go.mod directory so related
bumps land together.
EOF
)"
```

---

### Task 3: Bump Go toolchain pins to 1.26.5

**Files:**
- Modify: `go.mod` (line `go 1.26.3` → `go 1.26.5`)
- Modify: `tools/go.mod` (line `go 1.26.3` → `go 1.26.5`)
- Modify: `.tool-versions` (`golang 1.26.3` → `golang 1.26.5`)
- Leave: `.golangci.yml` `go: '1.26'` unchanged

**Interfaces:**
- Consumes: Go 1.26.5 available via mise
- Produces: consistent toolchain pins for app, tools, and local mise

- [ ] **Step 1: Install and activate Go 1.26.5 with mise**

```bash
mise install golang@1.26.5
mise use -g golang@1.26.5 || true
# Prefer project pin via .tool-versions after Step 2
eval "$(mise activate zsh)"
mise install
go version
```

Expected: `go version go1.26.5 ...`

- [ ] **Step 2: Update pin files**

In `go.mod`, change:

```go
go 1.26.5
```

In `tools/go.mod`, change:

```go
go 1.26.5
```

In `.tool-versions`, set:

```text
golang 1.26.5
```

- [ ] **Step 3: Verify pins**

```bash
rg -n '^go |^golang ' go.mod tools/go.mod .tool-versions .golangci.yml
go version
```

Expected:
- `go.mod` / `tools/go.mod`: `go 1.26.5`
- `.tool-versions`: `golang 1.26.5`
- `.golangci.yml`: still `go: '1.26'`
- `go version` reports `go1.26.5`

- [ ] **Step 4: Commit**

```bash
git add go.mod tools/go.mod .tool-versions
git commit -m "$(cat <<'EOF'
chore(deps): bump Go toolchain to 1.26.5

EOF
)"
```

---

### Task 4: Refresh root module dependencies

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: Go 1.26.5 toolchain from Task 3
- Produces: latest root direct/indirect dependency graph

- [ ] **Step 1: Upgrade root dependencies**

```bash
go get -u ./...
go mod tidy
```

Expected: command succeeds; `go.mod` / `go.sum` may change.

- [ ] **Step 2: Show what changed**

```bash
git diff -- go.mod go.sum
```

Expected: version bumps present; `go` line remains `1.26.5`.

- [ ] **Step 3: Compile/tests smoke for root module**

```bash
go test ./...
```

Expected: PASS. If FAIL due to API changes, fix in Task 6 (do not stop the upgrade; keep the bumped mods).

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "$(cat <<'EOF'
chore(deps): update root Go module dependencies

EOF
)"
```

If there is no diff, skip the commit and note “root already current”.

---

### Task 5: Refresh tools module dependencies

**Files:**
- Modify: `tools/go.mod`
- Modify: `tools/go.sum`

**Interfaces:**
- Consumes: Go 1.26.5 toolchain; tools listed in `tool (...)`
- Produces: updated tools graph including `x/net`, `x/crypto`, `grpc` beyond alert thresholds

- [ ] **Step 1: Upgrade tools and their transitive deps**

```bash
go get -u -modfile=tools/go.mod tool
go get -u -modfile=tools/go.mod \
  golang.org/x/crypto \
  golang.org/x/net \
  google.golang.org/grpc
go mod tidy -modfile=tools/go.mod
```

Expected: succeeds. `tools/go.mod` shows:
- `golang.org/x/crypto` >= `v0.52.0`
- `golang.org/x/net` >= `v0.55.0`
- `google.golang.org/grpc` >= `v1.82.1`
- `go 1.26.5`

- [ ] **Step 2: Verify security-related floors**

```bash
rg -n 'golang.org/x/crypto|golang.org/x/net|google.golang.org/grpc|^go ' tools/go.mod
```

Expected: versions meet or exceed the floors above.

- [ ] **Step 3: Smoke the pinned tools**

```bash
go tool -modfile=tools/go.mod golangci-lint version
go tool -modfile=tools/go.mod gosec -version
go tool -modfile=tools/go.mod govulncheck -version
go tool -modfile=tools/go.mod goimports -h >/dev/null
```

Expected: each tool runs without module resolution errors.

- [ ] **Step 4: Commit**

```bash
git add tools/go.mod tools/go.sum
git commit -m "$(cat <<'EOF'
chore(deps): update tools Go module dependencies

EOF
)"
```

---

### Task 6: Fix fallout and verify CI/security locally

**Files:**
- Modify as needed: application `.go` files under `cmd/`, `internal/`, `main.go`
- Possibly: `go.mod` / `go.sum` / `tools/go.mod` / `tools/go.sum` if tidy/lint require further adjustment

**Interfaces:**
- Consumes: upgraded modules from Tasks 4–5
- Produces: green `make ci` and `make security`

- [ ] **Step 1: Run full CI target**

```bash
make ci
```

Expected: passes end-to-end (deps, fmt-check, lint/gosec, tests with >=70% coverage, build).

- [ ] **Step 2: Run govulncheck**

```bash
make security
```

Expected: no vulnerabilities reported for `./...`.

- [ ] **Step 3: If anything failed, fix forward**

For compile/API breakages:
1. Read the failing package error
2. Update call sites to the new API
3. Re-run the failing command until green
4. Prefer fixing code over downgrading deps

For lint failures from newer golangci-lint:
1. Fix the reported issues in code, or adjust `.golangci.yml` only if the rule is newly noisy and clearly false-positive for this repo’s established patterns
2. Re-run `make ci`

- [ ] **Step 4: Re-verify**

```bash
make ci
make security
```

Expected: both pass.

- [ ] **Step 5: Commit any code fixes**

```bash
git add -A
git status
git commit -m "$(cat <<'EOF'
fix: resolve breakages from Go dependency upgrades

EOF
)"
```

If there is no diff after verification, skip the commit.

---

### Task 7: Open the PR

**Files:**
- None beyond already-committed changes

**Interfaces:**
- Consumes: green local verification from Task 6; closed Dependabot PRs from Task 1
- Produces: GitHub PR URL with conventional title

- [ ] **Step 1: Push branch**

```bash
git push -u origin HEAD
```

- [ ] **Step 2: Create PR**

```bash
gh pr create --title "chore(deps): batch go module updates and bump to 1.26.5" --body "$(cat <<'EOF'
## Summary
- Batch Dependabot Go version and security updates for `/` and `/tools` into catch-all groups
- Bump Go toolchain pins to 1.26.5
- Refresh all root and tools module dependencies (clears open tools alerts for x/net, x/crypto, grpc)
- Close superseded Dependabot PRs #37, #38, #39

## Test plan
- [ ] `make ci` passes locally
- [ ] `make security` (govulncheck) passes locally
- [ ] PR `quality` check is green
- [ ] PR `release-policy` check is green
- [ ] Dependabot alerts for tools/go.mod x/net, x/crypto, grpc are cleared after merge

EOF
)"
```

- [ ] **Step 3: Watch checks**

```bash
gh pr checks --watch
```

Expected: `quality` and `release-policy` pass. If either fails, fix on this branch and push; do not claim done until green.

- [ ] **Step 4: Confirm alert clearance path**

```bash
gh api repos/benvon/testrigor-ci-tool/dependabot/alerts --jq '.[] | select(.state=="open") | "\(.number)\t\(.dependency.package.name)\t\(.dependency.manifest_path)"'
```

Expected after merge (or once GitHub rescans the branch/PR): no open alerts for `golang.org/x/net`, `golang.org/x/crypto`, or `google.golang.org/grpc` in `tools/go.mod`. If alerts remain open only until merge, note that in the PR comment.

---

## Spec coverage checklist

| Spec requirement | Task |
|------------------|------|
| Dual groups for version + security updates | Task 2 |
| Add `/tools` Dependabot directory | Task 2 |
| Keep chore commit-message prefix | Task 2 |
| Leave github-actions unchanged | Task 2 |
| Close #37–#39 first | Task 1 |
| Go 1.26.5 in go.mod, tools/go.mod, .tool-versions | Task 3 |
| Refresh all `/` modules | Task 4 |
| Refresh all `/tools` modules | Task 5 |
| Fix code issues from upgrades | Task 6 |
| Verify make ci + security | Task 6 |
| Single conventional-commit PR | Task 7 |
