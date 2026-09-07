# trajirctl Prep Work Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract the generic path-safety primitives out of `go/trajir/mcp/paths.go` into a standalone `go/trajir/workdir` package, then cut a `go/vX.Y.Z` module tag, so the new `trajirctl` repo can depend on Trajectory IR's Go SDK at a real semver version instead of a commit SHA.

**Architecture:** `go/trajir/mcp/paths.go` currently mixes two things: `trajir-mcp`'s CWE-73 root-confinement policy (env var driven, fails closed) and generic, policy-free filesystem primitives (safe symlink resolution, subpath containment, symlink-leaf rejection) that policy is built on. Only the generic half moves. The confinement policy stays in `go/trajir/mcp` unchanged and calls the extracted package instead of defining these functions locally, so `trajir-mcp`'s external behavior does not change at all.

**Tech Stack:** Go 1.25, stdlib only (`os`, `path/filepath`, `fmt`, `strings`), Go's built-in `testing` package. No new dependencies.

**Spec:** [docs/superpowers/specs/2026-08-27-trajirctl-design.md](../specs/2026-08-27-trajirctl-design.md) §2 (Prep work in this repo)

## Global Constraints

- No behavior change to `trajir-mcp`: every existing test in `go/trajir/mcp` must still pass unmodified in intent (two tests need mechanical updates to keep compiling, called out in Task 2 — their assertions are unchanged).
- Extracted package exports exactly four functions, no more: `CanonicalizeDir`, `ResolveViaExistingAncestor`, `IsSubpath`, `RequireNonSymlinkLeaf`. The confinement-policy functions (`approvedRoot`, `requireBoundedWorkDir`, `requireBoundedPath`, `resolveUnderRoot`, `workdirSQLitePaths`, `EnvWorkspaceRoot`) stay in `go/trajir/mcp`.
- Do not move an already-pushed git tag (per [docs/RELEASE.md](../../RELEASE.md)) — if a tag is wrong, cut a new patch version instead.
- Every commit needs a DCO sign-off (`git commit -s`) — this repo's `DCO` CI check rejects unsigned commits.
- `main` is protected: no direct pushes. All changes land via a PR from a feature branch.
- PR bodies must follow `.github/PULL_REQUEST_TEMPLATE.md`, disclose AI assistance per `AI_POLICY.md`, and should read like plain human-written prose (no em dashes, no AI-sounding boilerplate) per established convention for this repo.
- Both new PRs in this plan get milestone `trajirctl operator CLI` (already created, GitHub milestone #8).

---

### Task 1: Create `go/trajir/workdir` package

**Files:**
- Create: `go/trajir/workdir/workdir.go`
- Create: `go/trajir/workdir/workdir_test.go`

**Interfaces:**
- Produces (package `github.com/Coder-s-OG-s/Trajectory-IR/go/trajir/workdir`):
  - `func CanonicalizeDir(path string) (string, error)`
  - `func ResolveViaExistingAncestor(path string) (string, error)`
  - `func IsSubpath(root, target string) bool`
  - `func RequireNonSymlinkLeaf(root, leafPath string) error`

- [ ] **Step 1: Write the failing tests**

Create `go/trajir/workdir/workdir_test.go`:

```go
package workdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCanonicalizeDir(t *testing.T) {
	root := t.TempDir()
	got, err := CanonicalizeDir(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}

	if _, err := CanonicalizeDir(filepath.Join(root, "missing")); err == nil {
		t.Fatal("expected error for nonexistent path")
	}

	file := filepath.Join(root, "f.txt")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CanonicalizeDir(file); err == nil {
		t.Fatal("expected error for a file, not a directory")
	} else if !strings.Contains(err.Error(), "not a directory") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestResolveViaExistingAncestorFullyExisting(t *testing.T) {
	root := t.TempDir()
	got, err := ResolveViaExistingAncestor(root)
	if err != nil {
		t.Fatal(err)
	}
	want, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestResolveViaExistingAncestorMissingLeaf(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "newdir", "out.tir")
	got, err := ResolveViaExistingAncestor(candidate)
	if err != nil {
		t.Fatal(err)
	}
	rootResolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(rootResolved, "newdir", "out.tir")
	if got != want {
		t.Fatalf("got=%q want=%q", got, want)
	}
}

func TestResolveViaExistingAncestorThroughSymlinkedParent(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()
	link := filepath.Join(root, "link")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	got, err := ResolveViaExistingAncestor(filepath.Join(link, "newdir", "out.tir"))
	if err != nil {
		t.Fatal(err)
	}
	targetResolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(targetResolved, "newdir", "out.tir")
	if got != want {
		t.Fatalf("got=%q want=%q (must resolve through the symlink target, not stop at the link)", got, want)
	}
}

func TestIsSubpath(t *testing.T) {
	root := t.TempDir()
	if !IsSubpath(root, root) {
		t.Fatal("root itself should be under root")
	}
	if !IsSubpath(root, filepath.Join(root, "child")) {
		t.Fatal("child should be under root")
	}
	if !IsSubpath(root, filepath.Join(root, "..secrets", "a")) {
		t.Fatal("..secrets should count as under root, not parent traversal")
	}
	if IsSubpath(root, filepath.Join(root, "..", "outside")) {
		t.Fatal("actual parent traversal should not count as under root")
	}
}

func TestRequireNonSymlinkLeafMissingIsAllowed(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "nodes.sqlite")
	if err := RequireNonSymlinkLeaf(root, leaf); err != nil {
		t.Fatal(err)
	}
}

func TestRequireNonSymlinkLeafRegularFileIsAllowed(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "nodes.sqlite")
	if err := os.WriteFile(leaf, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := RequireNonSymlinkLeaf(root, leaf); err != nil {
		t.Fatal(err)
	}
}

func TestRequireNonSymlinkLeafRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "nodes.sqlite")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "nodes.sqlite")
	if err := os.Symlink(outsideFile, link); err != nil {
		t.Skipf("symlink not available: %v", err)
	}
	err := RequireNonSymlinkLeaf(root, link)
	if err == nil {
		t.Fatal("expected symlink leaf to be rejected")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestRequireNonSymlinkLeafRejectsEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	// A leaf whose resolved location is not under root, reached without a
	// symlink at the leaf itself (e.g. root was computed wrong upstream).
	err := RequireNonSymlinkLeaf(root, filepath.Join(outside, "nodes.sqlite"))
	if err == nil {
		t.Fatal("expected leaf outside root to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("unexpected err: %v", err)
	}
}
```

- [ ] **Step 2: Run the tests and confirm they fail to compile**

Run: `cd go && go test ./trajir/workdir/...`
Expected: FAIL — `no Go files in .../go/trajir/workdir` or undefined symbols (package doesn't exist yet).

- [ ] **Step 3: Write the implementation**

Create `go/trajir/workdir/workdir.go`:

```go
// Package workdir provides generic, confinement-policy-free filesystem
// helpers for resolving user-supplied paths safely: canonicalizing
// directories, walking up to the nearest existing ancestor before
// resolving symlinks (so a symlinked parent can't be combined with a
// nonexistent leaf path to land somewhere unexpected), checking subpath
// containment, and rejecting symlinked leaf files. This package has no
// opinion on which root a caller should confine paths to — callers that
// need to enforce a specific confinement policy (an approved-root env
// var, for example) build it on top of these primitives.
package workdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CanonicalizeDir resolves path to an absolute, symlink-resolved directory
// path. It returns an error if path does not exist or is not a directory.
func CanonicalizeDir(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", err
	}
	st, err := os.Stat(resolved)
	if err != nil {
		return "", err
	}
	if !st.IsDir() {
		return "", fmt.Errorf("not a directory: %s", path)
	}
	return resolved, nil
}

// ResolveViaExistingAncestor finds the nearest existing ancestor of path,
// resolves symlinks there with EvalSymlinks, then rejoins any missing
// trailing path components. This blocks a symlinked/junction parent from
// being combined with a nonexistent leaf path to resolve somewhere other
// than where the symlink target actually points.
func ResolveViaExistingAncestor(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	var missing []string
	cur := abs
	for {
		_, err := os.Lstat(cur)
		if err == nil {
			break
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		base := filepath.Base(cur)
		missing = append([]string{base}, missing...)
		parent := filepath.Dir(cur)
		if parent == cur {
			return "", fmt.Errorf("no existing ancestor")
		}
		cur = parent
	}

	resolvedAncestor, err := filepath.EvalSymlinks(cur)
	if err != nil {
		return "", err
	}
	if len(missing) == 0 {
		return resolvedAncestor, nil
	}
	return filepath.Join(append([]string{resolvedAncestor}, missing...)...), nil
}

// IsSubpath reports whether target is root itself or lies inside it. A
// name that merely starts with ".." (e.g. "..secrets") is not treated as
// parent traversal; only an actual ".." path segment counts as escaping
// root.
func IsSubpath(root, target string) bool {
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}

// RequireNonSymlinkLeaf rejects a leafPath that is itself a symlink, or
// whose resolved location escapes root. A leafPath that does not exist yet
// is allowed, since callers typically create it as a regular file on
// first use.
func RequireNonSymlinkLeaf(root, leafPath string) error {
	info, err := os.Lstat(leafPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("workdir: inspect %q: %w", filepath.Base(leafPath), err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("workdir: %q must not be a symlink", filepath.Base(leafPath))
	}
	abs, err := ResolveViaExistingAncestor(leafPath)
	if err != nil {
		return err
	}
	if !IsSubpath(root, abs) {
		return fmt.Errorf("workdir: %q escapes root %q", filepath.Base(leafPath), root)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests and confirm they pass**

Run: `cd go && go test ./trajir/workdir/... -v`
Expected: PASS for all tests (symlink-dependent tests may `SKIP` on a filesystem without symlink privileges — that's fine, don't force it).

- [ ] **Step 5: Commit**

```bash
git checkout -b refactor/extract-workdir-package
git add go/trajir/workdir/workdir.go go/trajir/workdir/workdir_test.go
git commit -s -m "feat(go): add trajir/workdir package for generic path-safety primitives"
```

---

### Task 2: Rewire `go/trajir/mcp` to use the extracted package

**Files:**
- Modify: `go/trajir/mcp/paths.go`
- Modify: `go/trajir/mcp/paths_test.go`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: `workdir.CanonicalizeDir`, `workdir.ResolveViaExistingAncestor`, `workdir.IsSubpath`, `workdir.RequireNonSymlinkLeaf` from Task 1.
- Produces: no new symbols. `requireBoundedWorkDir`, `requireBoundedPath`, `workdirSQLitePaths`, `approvedRoot`, `EnvWorkspaceRoot` in package `mcp` keep their existing signatures — `go/trajir/mcp/tools.go` needs zero changes.

- [ ] **Step 1: Confirm the current mcp test suite passes before touching anything**

Run: `cd go && go test ./trajir/mcp/...`
Expected: PASS (baseline, so any later failure is attributable to this task's changes).

- [ ] **Step 2: Replace `go/trajir/mcp/paths.go`**

Replace the full file content with:

```go
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Coder-s-OG-s/Trajectory-IR/go/trajir/workdir"
)

// EnvWorkspaceRoot is the environment variable for the approved workspace root.
// It must be set explicitly: all MCP tool paths must resolve under this root
// (CWE-73 / prompt-injected path confinement), so silently falling back to the
// process's cwd when unset would let confinement widen to whatever directory
// happened to launch the binary. Fail closed instead.
const EnvWorkspaceRoot = "TRAJIR_MCP_ROOT"

// approvedRoot returns the canonical absolute workspace root.
func approvedRoot() (string, error) {
	root := strings.TrimSpace(os.Getenv(EnvWorkspaceRoot))
	if root == "" {
		return "", fmt.Errorf("mcp: %s must be set to an approved workspace root", EnvWorkspaceRoot)
	}
	return workdir.CanonicalizeDir(root)
}

// requireBoundedWorkDir validates work_dir under the approved root.
// Empty work_dir means the root itself. The directory must already exist.
func requireBoundedWorkDir(workDir string) (string, error) {
	root, err := approvedRoot()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(workDir) == "" {
		return root, nil
	}
	return resolveUnderRoot(root, workDir, true)
}

// requireBoundedPath validates path is under root (or under preferredRoot when set).
// preferRoot is typically the validated work_dir for exports.
func requireBoundedPath(path, preferRoot string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("mcp: path is required")
	}
	root, err := approvedRoot()
	if err != nil {
		return "", err
	}
	base := root
	if strings.TrimSpace(preferRoot) != "" {
		pref, err := resolveUnderRoot(root, preferRoot, true)
		if err != nil {
			return "", err
		}
		base = pref
	}
	return resolveUnderRoot(base, path, false)
}

// resolveUnderRoot cleans and absolute-izes path, ensuring it stays under root.
// When requireDir is true, path must already exist and be a directory.
func resolveUnderRoot(root, userPath string, requireDir bool) (string, error) {
	rootAbs, err := workdir.CanonicalizeDir(root)
	if err != nil {
		return "", err
	}

	candidate := userPath
	if !filepath.IsAbs(candidate) {
		candidate = filepath.Join(rootAbs, candidate)
	}
	candidate = filepath.Clean(candidate)

	resolved, err := workdir.ResolveViaExistingAncestor(candidate)
	if err != nil {
		return "", fmt.Errorf("mcp: path %q: %w", userPath, err)
	}
	if !workdir.IsSubpath(rootAbs, resolved) {
		return "", fmt.Errorf("mcp: path %q escapes workspace root %q", userPath, rootAbs)
	}
	if requireDir {
		st, err := os.Stat(resolved)
		if err != nil {
			if os.IsNotExist(err) {
				return "", fmt.Errorf("mcp: work_dir %q does not exist", userPath)
			}
			return "", err
		}
		if !st.IsDir() {
			return "", fmt.Errorf("mcp: work_dir %q is not a directory", userPath)
		}
	}
	return resolved, nil
}

// workdirSQLitePaths returns validated nodes/memo paths under workDir (no symlink leaves).
func workdirSQLitePaths(workDir string) (nodesPath, memoPath string, err error) {
	root, err := approvedRoot()
	if err != nil {
		return "", "", err
	}
	nodesPath = filepath.Join(workDir, "nodes.sqlite")
	memoPath = filepath.Join(workDir, "memo.sqlite")
	if err := workdir.RequireNonSymlinkLeaf(root, nodesPath); err != nil {
		return "", "", err
	}
	if err := workdir.RequireNonSymlinkLeaf(root, memoPath); err != nil {
		return "", "", err
	}
	return nodesPath, memoPath, nil
}
```

- [ ] **Step 3: Fix the two direct call sites in `paths_test.go`**

`paths_test.go` has two places that call the functions that just moved:
`TestIsSubpathExactDotDot` calls `isSubpath` directly (now fully covered by
`TestIsSubpath` in `go/trajir/workdir/workdir_test.go`, so it is deleted
here rather than adapted), and `TestRegularSQLiteLeafAllowed` calls
`requireNonSymlinkLeaf` directly (adapted to call the exported package
function).

Add the import at the top of `go/trajir/mcp/paths_test.go`:

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Coder-s-OG-s/Trajectory-IR/go/trajir/workdir"
)
```

In `TestRegularSQLiteLeafAllowed`, change:

```go
	if err := requireNonSymlinkLeaf(root, nodes); err != nil {
		t.Fatal(err)
	}
```

to:

```go
	if err := workdir.RequireNonSymlinkLeaf(root, nodes); err != nil {
		t.Fatal(err)
	}
```

Delete `TestIsSubpathExactDotDot` in its entirety (the whole function, lines
194-205 in the pre-extraction file):

```go
func TestIsSubpathExactDotDot(t *testing.T) {
	root := t.TempDir()
	if !isSubpath(root, filepath.Join(root, "..secrets", "a")) {
		t.Fatal("..secrets should count as under root")
	}
	if isSubpath(root, filepath.Join(root, "..", "outside")) {
		t.Fatal("parent traversal should not count as under root")
	}
	if !isSubpath(root, root) {
		t.Fatal("root itself should be under root")
	}
}
```

- [ ] **Step 4: Run the full Go module test suite**

Run: `cd go && go build ./... && go test ./...`
Expected: PASS across every package, in particular `go/trajir/mcp` (all
existing tests, unmodified assertions) and `go/trajir/workdir` (Task 1's
tests). A green `go build ./...` also confirms `go/cmd/trajir-mcp` still
compiles against the rewired `mcp` package with no call-site changes needed.

- [ ] **Step 5: Add a CHANGELOG entry**

In `CHANGELOG.md`, under the existing `## [Unreleased]` → `### Changed`
section (create the `### Changed` heading under `[Unreleased]` if the
current top entry there is something else — check the file first), add:

```markdown
- Go: extracted generic path-safety helpers (`CanonicalizeDir`,
  `ResolveViaExistingAncestor`, `IsSubpath`, `RequireNonSymlinkLeaf`) out of
  `go/trajir/mcp` into a new `go/trajir/workdir` package. `trajir-mcp`'s
  CWE-73 root-confinement policy is unchanged and now calls the extracted
  package; no external behavior change.
```

- [ ] **Step 6: Commit**

```bash
git add go/trajir/mcp/paths.go go/trajir/mcp/paths_test.go CHANGELOG.md
git commit -s -m "refactor(go): rewire trajir/mcp path confinement onto trajir/workdir"
```

- [ ] **Step 7: Push the branch and open the PR**

```bash
git push -u origin refactor/extract-workdir-package
```

Open the PR with `gh pr create`, following `.github/PULL_REQUEST_TEMPLATE.md`
exactly (Summary, Test plan, Checklist, Safety areas, AI assistance
sections), assign milestone `trajirctl operator CLI`, sign off is already on
both commits. Write the body in plain prose, no em dashes, matching this
repo's established PR style. Wait for required CI checks (DCO, Quality,
Go, etc. — see `docs/maintainer-branch-protection.md`) to go green, then
merge.

- [ ] **Step 8: Sync local `main` and delete the merged branch**

```bash
git checkout main
git pull origin main
git branch -d refactor/extract-workdir-package
```

---

### Task 3: Cut the `go/vX.Y.Z` module tag

This follows the existing release process in
[docs/RELEASE.md](../../RELEASE.md), adding the subdirectory module tag
alongside the usual root tag. Only start this once Task 2's PR is merged
and `main` is green.

**Files:**
- Modify: `pyproject.toml`
- Modify: `CHANGELOG.md`

**Interfaces:**
- Consumes: Task 2 merged to `main` (the code that ships in this tag).
- Produces: git tags `v0.2.2` and `go/v0.2.2` on the same commit, so
  `go get github.com/Coder-s-OG-s/Trajectory-IR/go@v0.2.2` resolves for the
  `trajirctl` repo.

- [ ] **Step 1: Confirm preconditions from `docs/RELEASE.md`**

```bash
git checkout main
git pull origin main
git status
```

Expected: working tree clean, `main` up to date with `origin/main`, and the
latest CI run on `main` green (`gh run list --branch main --limit 3`).

- [ ] **Step 2: Bump the version and move CHANGELOG `[Unreleased]` into `[0.2.2]`**

In `pyproject.toml`, change:

```toml
version = "0.2.1"
```

to:

```toml
version = "0.2.2"
```

In `CHANGELOG.md`, rename the `## [Unreleased]` heading's contents into a
new dated section (keep an empty `## [Unreleased]` above it for future
entries):

```markdown
## [Unreleased]

## [0.2.2] - 2026-09-07

### Added
<!-- move the existing [Unreleased] "Added" bullets here verbatim -->

### Changed
<!-- move the existing [Unreleased] "Changed" bullets here verbatim, including
     the go/trajir/workdir entry from Task 2 -->
```

Read the current `[Unreleased]` section in full before editing — move its
existing bullets as-is under `[0.2.2]`, don't drop or rewrite them.

- [ ] **Step 3: Branch, commit, and open the version-bump PR**

```bash
git checkout -b release/v0.2.2
git add pyproject.toml CHANGELOG.md
git commit -s -m "chore: prepare v0.2.2 release"
git push -u origin release/v0.2.2
```

Open the PR the same way as Task 2 (template filled in, milestone
`trajirctl operator CLI`, plain prose, AI assistance disclosed). Wait for
CI green, then merge.

- [ ] **Step 4: Sync `main` and tag**

```bash
git checkout main
git pull origin main
git branch -d release/v0.2.2

git tag -a v0.2.2 -m "Trajectory IR v0.2.2"
git tag -a go/v0.2.2 -m "Trajectory IR Go module v0.2.2"

git push origin v0.2.2
git push origin go/v0.2.2
```

- [ ] **Step 5: Verify the release workflow and asset attachment**

```bash
gh run list --workflow=release.yml --limit 5
gh release view v0.2.2 --json assets,url
```

Expected: `release.yml` run for the `v0.2.2` tag is green, and the release
has non-empty wheel/sdist assets. `go/v0.2.2` does not match the
workflow's `v*` tag trigger (it doesn't start with `v`), so no separate run
is expected for it — that's correct, it's a plain git tag for Go module
resolution, not a packaged release.

- [ ] **Step 6: Verify the Go module resolves at the new tag**

Run this from a scratch directory (use the session scratchpad, not inside
this repo or the `trajirctl` repo):

```bash
mkdir -p /tmp/verify-go-tag && cd /tmp/verify-go-tag
go mod init verify-go-tag
go get github.com/Coder-s-OG-s/Trajectory-IR/go@v0.2.2
cat go.mod
```

Expected: `go.mod` records
`require github.com/Coder-s-OG-s/Trajectory-IR/go v0.2.2` with no pseudo-version
suffix (no `-0.20260907...-abcdef123456`), confirming `trajirctl` will be able to
depend on a real tagged version rather than a commit SHA.

Clean up the scratch directory afterward — it isn't part of any repo.

---

## Self-Review Notes

- **Spec coverage:** §2.1 (extraction) → Tasks 1-2. §2.2 (tag) → Task 3.
  Both prep-work items from the spec are covered; command-surface,
  repository-structure, and other trajirctl-repo sections of the spec are
  out of scope for this plan by design (they land in the new `trajirctl`
  repo, not here).
- **Compile-breaking detail caught during planning:** `paths_test.go` calls
  `requireNonSymlinkLeaf` and `isSubpath` directly in two tests, not just
  through the confinement-layer functions. Task 2 Step 3 handles both
  explicitly instead of assuming the extraction is purely additive.
- **No behavior change verified structurally:** Task 2 keeps every
  confinement-policy function's name and signature identical, so
  `go/cmd/trajir-mcp` (and anything else importing `go/trajir/mcp`) needs
  zero changes — verified by grepping `tools.go` for call sites before
  writing this plan.
