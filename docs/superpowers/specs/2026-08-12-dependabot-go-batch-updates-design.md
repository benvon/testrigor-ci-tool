# Dependabot Go batching + toolchain refresh

Date: 2026-08-12  
Status: Approved  
Branch: `chore/dependabot-go-batch-updates`

## Problem

Open Dependabot security update PRs for `tools/go.mod` (`golang.org/x/net`, `golang.org/x/crypto`, `google.golang.org/grpc`) arrive one dependency at a time. Related transitive updates need to land together or remaining alerts stay open and security posture stays red. `tools/` is also missing from Dependabot config, so alert-driven PRs are unmanaged. Separately, PR titles that are not Conventional Commits fail `release-policy`.

## Goals

1. Batch Go module updates per directory for both version updates and security updates.
2. Bump the Go toolchain to the latest patch (`1.26.5` at design time).
3. Refresh all dependencies in `/` and `/tools`.
4. Resolve any code/lint/test breakage from those bumps.
5. Close existing one-off Dependabot PRs (#37, #38, #39) before opening the replacement PR.

## Non-goals

- Changing GitHub Actions Dependabot grouping.
- Changing reusable workflows or release-policy rules.
- Unrelated refactors.

## Approach

Use Dependabot dual catch-all groups per Go module directory (approach 1 from brainstorming):

- One group for `version-updates`
- One group for `security-updates`

Omitting `applies-to` is **not** sufficient: Dependabot defaults that to version updates only.

## Dependabot design

Update `.github/dependabot.yml`:

- Keep existing `gomod` entry for `/`.
- Add `gomod` entry for `/tools`.
- In each Go entry, add:

```yaml
groups:
  go-modules:
    applies-to: version-updates
    patterns:
      - "*"
  go-security:
    applies-to: security-updates
    patterns:
      - "*"
```

- Preserve weekly schedule, `open-pull-requests-limit: 10`, labels, and `chore` + scope commit-message prefix (required for `release-policy`).
- Leave `github-actions` ecosystem unchanged.

Expected behavior after merge: at most one batched version-update PR and one batched security-update PR per directory per cycle, instead of per-package security PRs.

## Toolchain and dependency refresh

1. Close PRs #37, #38, #39.
2. Set Go to `1.26.5` in:
   - `go.mod`
   - `tools/go.mod`
   - `.tool-versions`
   - `.golangci.yml` (`go:` field) if needed
3. Update all modules:
   - Root: `go get -u ./...` then `go mod tidy`
   - Tools: update via `-modfile=tools/go.mod`, then `go mod tidy -modfile=tools/go.mod`
4. Run `make ci` and `make security` (govulncheck). Fix compile/lint/API fallout.

## PR delivery

- Single PR titled like `chore(deps): batch go module updates and bump to 1.26.5`
- Includes Dependabot config, toolchain pins, full module refresh, and any required code fixes
- Verification before claiming done:
  - `make ci` passes locally
  - PR `quality` and `release-policy` checks are green
  - Open Dependabot alerts for `tools/go.mod` (`x/net`, `x/crypto`, `grpc`) are cleared

## Risks and notes

- Dependabot may still open separate version vs security grouped PRs in the same week; each remains batched.
- Aggressive `go get -u` can pull major bumps; resolve breakages in this same PR or pin only if a major is intentionally deferred (prefer fixing forward).
- Local `main` must be based on current `origin/main` (includes `tools/` module and reusable workflow setup).
