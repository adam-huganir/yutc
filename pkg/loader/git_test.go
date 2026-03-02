package loader

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveGitPath_RejectsDotSegments(t *testing.T) {
	checkoutDir := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	tests := []string{
		".",
		"..",
		"templates/./app.tmpl",
		"templates/../app.tmpl",
		"./templates/app.tmpl",
		"templates\\.\\app.tmpl",
		"templates\\..\\app.tmpl",
	}

	for _, input := range tests {
		_, err := resolveGitPath(checkoutDir, input)
		require.Error(t, err, "expected %q to be rejected", input)
		assert.Contains(t, err.Error(), "dot path segments are not allowed")
	}
}

func TestResolveGitPath_AllowsNormalSubpath(t *testing.T) {
	checkoutDir := filepath.Join(t.TempDir(), "checkout")
	require.NoError(t, os.MkdirAll(checkoutDir, 0o755))

	resolved, err := resolveGitPath(checkoutDir, "templates/app.tmpl")
	require.NoError(t, err)

	absCheckout, err := filepath.Abs(checkoutDir)
	require.NoError(t, err)
	expected := filepath.Join(absCheckout, "templates", "app.tmpl")
	assert.Equal(t, expected, resolved)
}

func TestGitSource_LoadAndContainer(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary is required for git source tests")
	}

	repo := filepath.Join(t.TempDir(), "repo")
	assert.NoError(t, os.MkdirAll(filepath.Join(repo, "templates"), 0o755))
	assert.NoError(t, os.WriteFile(filepath.Join(repo, "templates", "app.tmpl"), []byte("hello"), 0o644))
	assert.NoError(t, os.WriteFile(filepath.Join(repo, "values.yaml"), []byte("name: test"), 0o644))

	runGitTestCommand(t, repo, "init")
	runGitTestCommand(t, repo, "add", ".")
	runGitTestCommand(t, repo, "-c", "user.email=test@example.com", "-c", "user.name=Test", "commit", "-m", "initial")
	runGitTestCommand(t, repo, "tag", "v1")

	checkoutRoot := filepath.Join(t.TempDir(), "checkouts")

	fileEntry := NewFileEntry(repo, WithGitSource(repo, "v1", "values.yaml", checkoutRoot))
	err := fileEntry.Load()
	assert.NoError(t, err)
	assert.Contains(t, string(fileEntry.Content.Data), "name: test")
	assert.Equal(t, SourceKindGit, fileEntry.Source)
	assert.NotNil(t, fileEntry.Git)
	assert.NotEmpty(t, fileEntry.Git.CheckoutDir)
	assert.NotEmpty(t, fileEntry.Git.ResolvedPath)

	dirEntry := NewFileEntry(repo, WithGitSource(repo, "v1", "templates", checkoutRoot))
	isDir, err := dirEntry.IsDir()
	assert.NoError(t, err)
	assert.True(t, isDir)

	entries, err := GetEntries(dirEntry, nil)
	assert.NoError(t, err)
	if assert.NotEmpty(t, entries) {
		found := false
		for _, e := range entries {
			if filepath.Base(e.Name) == "app.tmpl" {
				found = true
				break
			}
		}
		assert.True(t, found, "expected templates/app.tmpl in git container entries")
	}
}

func runGitTestCommand(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v: %s", args, err, string(out))
	}
}
