package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadOptimization(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()
	lockDir := filepath.Join(tmpDir, "001")
	err := os.MkdirAll(lockDir, 0755)
	require.NoError(t, err)

	testUrl := ts.URL + "/test.txt"
	testContent := []byte("test content")
	testFile := filepath.Join(tmpDir, "test.txt")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	t.Run("Valid File Not Redownloaded", func(t *testing.T) {
		lockPath := filepath.Join(lockDir, "grabit.lock")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, RecommendedAlgo, nil, "test.txt")
		require.NoError(t, err)

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})

	t.Run("Invalid File Redownloaded", func(t *testing.T) {
		lockPath := filepath.Join(lockDir, "grabit.lock")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, RecommendedAlgo, nil, "test.txt")
		require.NoError(t, err)

		err = os.WriteFile(testFile, []byte("corrupted"), 0644)
		require.NoError(t, err)

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})

	t.Run("Multiple Resources", func(t *testing.T) {
		lockPath := filepath.Join(lockDir, "grabit.lock")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		urls := []string{
			ts.URL + "/1.txt",
			ts.URL + "/2.txt",
		}

		for _, url := range urls {
			err = lock.AddResource([]string{url}, RecommendedAlgo, nil, "")
			require.NoError(t, err)
		}

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})

	t.Run("Resource Management", func(t *testing.T) {
		lockPath := filepath.Join(lockDir, "grabit.lock")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, RecommendedAlgo, []string{"test"}, "test.txt")
		require.NoError(t, err)

		assert.True(t, lock.Contains(testUrl))
		lock.DeleteResource(testUrl)
		assert.False(t, lock.Contains(testUrl))
	})
}
