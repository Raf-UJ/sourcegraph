package codycontext

import (
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sourcegraph/sourcegraph/internal/api"
	"github.com/sourcegraph/sourcegraph/internal/gitserver"
	"github.com/sourcegraph/sourcegraph/internal/types"
	"github.com/sourcegraph/sourcegraph/lib/errors"

	"github.com/stretchr/testify/require"
)

func TestNewFilter(t *testing.T) {
	repos := []types.RepoIDName{
		{ID: 1, Name: "repo1"},
		{ID: 2, Name: "repo2"},
	}

	t.Run("no ignore files", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("main", api.CommitID("abc123"), nil)
		client.NewFileReaderFunc.SetDefaultReturn(nil, os.ErrNotExist)
		f, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.NoError(t, err)

		chunks := []FileChunkContext{
			{
				RepoName: "repo1",
				RepoID:   1,
				Path:     "/file1.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "/file2.go",
			},
		}

		filtered := f.Filter(chunks)
		require.Equal(t, 2, len(filtered))
	})

	t.Run("filters multiple rules in ignore file", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("main", api.CommitID("abc123"), nil)
		client.NewFileReaderFunc.SetDefaultHook(func(ctx context.Context, rn api.RepoName, ci api.CommitID, s string) (io.ReadCloser, error) {
			if rn == "repo2" {
				return io.NopCloser(strings.NewReader("**/file1.go\nsecret.txt")), nil
			}
			return nil, os.ErrNotExist
		})

		f, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.NoError(t, err)

		chunks := []FileChunkContext{
			{
				RepoName: "repo1",
				RepoID:   1,
				Path:     "file1.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "folder1/file1.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "folder1/folder2/file1.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "secret.txt",
			},
		}

		filtered := f.Filter(chunks)
		require.Equal(t, 1, len(filtered))
		require.Equal(t, api.RepoName("repo1"), filtered[0].RepoName)
	})

	t.Run("uses correct ignore file by repo", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("main", api.CommitID("abc123"), nil)
		client.NewFileReaderFunc.SetDefaultHook(func(ctx context.Context, rn api.RepoName, ci api.CommitID, s string) (io.ReadCloser, error) {
			switch rn {
			case "repo1":
				return io.NopCloser(strings.NewReader("**/file1.go")), nil
			case "repo2":
				return io.NopCloser(strings.NewReader("**/file2.go")), nil
			default:
				return nil, os.ErrNotExist
			}
		})
		f, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.NoError(t, err)

		chunks := []FileChunkContext{
			{
				RepoName: "repo1",
				RepoID:   1,
				Path:     "src/file1.go",
			},
			{
				RepoName: "repo1",
				RepoID:   1,
				Path:     "src/file2.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "src/file1.go",
			},
			{
				RepoName: "repo2",
				RepoID:   2,
				Path:     "src/file2.go",
			},
		}

		filtered := f.Filter(chunks)
		require.Equal(t, 2, len(filtered))
		require.Equal(t, api.RepoName("repo1"), filtered[0].RepoName)
		require.Equal(t, "src/file2.go", filtered[0].Path)
		require.Equal(t, api.RepoName("repo2"), filtered[1].RepoName)
		require.Equal(t, "src/file1.go", filtered[1].Path)
	})

	t.Run("empty repos don't error", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("", api.CommitID(""), nil)
		client.NewFileReaderFunc.SetDefaultHook(func(ctx context.Context, rn api.RepoName, ci api.CommitID, s string) (io.ReadCloser, error) {
			t.Errorf("repos are empty, no files should be read")
			return nil, nil
		})

		_, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.NoError(t, err)
	})

	t.Run("errors checking head do error", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("", api.CommitID(""), errors.New("fail"))
		client.NewFileReaderFunc.SetDefaultHook(func(ctx context.Context, rn api.RepoName, ci api.CommitID, s string) (io.ReadCloser, error) {
			t.Errorf("failed checking head should not continue")
			return nil, nil
		})

		_, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.Error(t, err)
	})

	t.Run("error reading ignore file causes error", func(t *testing.T) {
		client := gitserver.NewMockClient()
		client.GetDefaultBranchFunc.SetDefaultReturn("main", api.CommitID("abc123"), nil)
		client.NewFileReaderFunc.SetDefaultHook(func(ctx context.Context, rn api.RepoName, ci api.CommitID, s string) (io.ReadCloser, error) {
			return nil, errors.New("fail")
		})

		_, err := NewCodyIgnoreFilter(context.Background(), client, repos)
		require.Error(t, err)
	})
}
