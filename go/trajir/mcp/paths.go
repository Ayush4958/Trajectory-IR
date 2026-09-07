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
