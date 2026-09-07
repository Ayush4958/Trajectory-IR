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
	// A leaf that already exists outside root, reached without a symlink
	// at the leaf itself (e.g. root was computed wrong upstream). Must
	// exist for Lstat to reach the escape check rather than the
	// missing-leaf-is-allowed short circuit.
	outsideFile := filepath.Join(outside, "nodes.sqlite")
	if err := os.WriteFile(outsideFile, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := RequireNonSymlinkLeaf(root, outsideFile)
	if err == nil {
		t.Fatal("expected leaf outside root to be rejected")
	}
	if !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("unexpected err: %v", err)
	}
}
