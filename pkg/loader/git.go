package loader

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitInfo describes how a git-backed input should be checked out and resolved.
type GitInfo struct {
	Repo         string
	Ref          string
	Path         string
	TempRoot     string
	CheckoutDir  string
	ResolvedPath string
}

// EnsureGitCheckout ensures the git repo is available locally and resolves the effective input path.
func (f *FileEntry) EnsureGitCheckout() error {
	if f.Source != SourceKindGit {
		return nil
	}
	if f.Git == nil {
		return fmt.Errorf("git source %s missing git metadata", f.Name)
	}
	if f.Git.Repo == "" {
		return fmt.Errorf("git source %s missing repo", f.Name)
	}

	if f.Git.CheckoutDir == "" {
		checkoutDir, err := deterministicCheckoutDir(f.Git.Repo, f.Git.Ref, f.Git.TempRoot)
		if err != nil {
			return err
		}
		f.Git.CheckoutDir = checkoutDir
	}

	if err := ensureCheckoutExists(f.Git.CheckoutDir, f.Git.Repo); err != nil {
		return err
	}
	if f.Git.Ref != "" {
		if err := runGitCommand(filepath.Dir(f.Git.CheckoutDir), "-C", f.Git.CheckoutDir, "fetch", "--all", "--tags", "--prune"); err != nil {
			return err
		}
		if err := runGitCommand(filepath.Dir(f.Git.CheckoutDir), "-C", f.Git.CheckoutDir, "checkout", f.Git.Ref); err != nil {
			return err
		}
	}

	resolved, err := resolveGitPath(f.Git.CheckoutDir, f.Git.Path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(resolved); err != nil {
		return fmt.Errorf("git source path does not exist (%s): %w", resolved, err)
	}

	f.Git.ResolvedPath = NormalizeFilepath(resolved)
	f.Name = f.Git.ResolvedPath
	f.isDir = nil
	f.isFile = nil
	return nil
}

func deterministicCheckoutDir(repo, ref, tempRoot string) (string, error) {
	root := strings.TrimSpace(tempRoot)
	if root == "" {
		root = os.TempDir()
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return "", err
	}
	h := sha1.Sum([]byte(repo + "\n" + ref))
	suffix := hex.EncodeToString(h[:])[:12]
	return filepath.Join(root, "git-"+suffix), nil
}

func ensureCheckoutExists(checkoutDir, repo string) error {
	gitDir := filepath.Join(checkoutDir, ".git")
	if ok, err := Exists(gitDir); err != nil {
		return err
	} else if ok {
		return nil
	}
	if ok, err := Exists(checkoutDir); err != nil {
		return err
	} else if !ok {
		if err := os.MkdirAll(filepath.Dir(checkoutDir), 0o755); err != nil {
			return err
		}
	}
	return runGitCommand(filepath.Dir(checkoutDir), "clone", repo, checkoutDir)
}

func runGitCommand(cwd string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

func resolveGitPath(checkoutDir, p string) (string, error) {
	cleanPath := strings.TrimSpace(p)
	if cleanPath == "" {
		return checkoutDir, nil
	}
	if hasDotPathSegment(cleanPath) {
		return "", fmt.Errorf("invalid git path %q: dot path segments are not allowed", p)
	}
	cleanPath = filepath.Clean(filepath.FromSlash(cleanPath))
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid git path %q: cannot escape repository root", p)
	}

	resolved := filepath.Join(checkoutDir, cleanPath)
	absCheckout, err := filepath.Abs(checkoutDir)
	if err != nil {
		return "", err
	}
	absResolved, err := filepath.Abs(resolved)
	if err != nil {
		return "", err
	}
	absCheckout = filepath.Clean(absCheckout)
	absResolved = filepath.Clean(absResolved)

	if absResolved != absCheckout && !strings.HasPrefix(absResolved, absCheckout+string(filepath.Separator)) {
		return "", fmt.Errorf("invalid git path %q: cannot escape repository root", p)
	}
	return absResolved, nil
}

func hasDotPathSegment(p string) bool {
	segments := strings.FieldsFunc(p, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	for _, segment := range segments {
		if segment == "." || segment == ".." {
			return true
		}
	}
	return false
}
