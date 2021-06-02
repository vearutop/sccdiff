package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_runAtGitRef(t *testing.T) {
	dir := t.TempDir()
	fooPath := filepath.Join(dir, "foo")
	err := ioutil.WriteFile(fooPath, []byte("OG content"), 0o600)
	require.NoError(t, err)
	mustGit(t, dir, "init")
	mustGit(t, dir, "add", "foo")
	mustGit(t, dir, "commit", "-m", "ignore me")
	untrackedPath := filepath.Join(dir, "untracked")

	err = ioutil.WriteFile(untrackedPath, []byte("untracked"), 0o600)
	require.NoError(t, err)

	err = ioutil.WriteFile(fooPath, []byte("new content"), 0o600)
	require.NoError(t, err)

	fn := func(workDir string) {
		var got []byte

		untrackedPath := filepath.Join(workDir, "untracked")

		_, err = ioutil.ReadFile(untrackedPath)
		require.Error(t, err)

		wdFooPath := filepath.Join(workDir, "foo")

		got, err = ioutil.ReadFile(wdFooPath)
		require.NoError(t, err)

		require.Equal(t, "OG content", string(got))
	}

	err = runAtGitRef(nil, "git", dir, "HEAD", fn)
	require.NoError(t, err)

	got, err := ioutil.ReadFile(fooPath)
	require.NoError(t, err)

	require.Equal(t, "new content", string(got))
}

func mustSetEnv(t *testing.T, env map[string]string) {
	t.Helper()

	for k, v := range env {
		assert.NoError(t, os.Setenv(k, v))
	}
}

func mustGit(t *testing.T, repoPath string, args ...string) []byte {
	t.Helper()
	mustSetEnv(t, map[string]string{
		"GIT_AUTHOR_NAME":     "author",
		"GIT_AUTHOR_EMAIL":    "author@localhost",
		"GIT_COMMITTER_NAME":  "committer",
		"GIT_COMMITTER_EMAIL": "committer@localhost",
	})

	got, err := runGitCmd(nil, "git", repoPath, args...)
	assert.NoErrorf(t, err, "error running git:\noutput: %v", string(got))

	return got
}
