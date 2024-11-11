package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cisco-open/grabit/test"
	"github.com/stretchr/testify/require"
)

func TestOptimization(t *testing.T) {
	// Setup test server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("test content"))
	}))
	defer ts.Close()

	t.Run("Valid File Not Redownloaded", func(t *testing.T) {
		tmpDir := test.TmpDir(t)
		lockDir := filepath.Join(tmpDir, "valid_test")
		err := os.MkdirAll(lockDir, 0755)
		require.NoError(t, err)

		testUrl := ts.URL + "/valid_test.txt"
		testFile := test.TmpFile(t, "test content")

		lockPath := test.TmpFile(t, "")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, RecommendedAlgo, nil, filepath.Base(testFile))
		require.NoError(t, err)

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})

	t.Run("Invalid File Redownloaded", func(t *testing.T) {
		tmpDir := test.TmpDir(t)
		lockDir := filepath.Join(tmpDir, "invalid_test")
		err := os.MkdirAll(lockDir, 0755)
		require.NoError(t, err)

		testUrl := ts.URL + "/invalid_test.txt"
		testFile := test.TmpFile(t, "test content")

		lockPath := test.TmpFile(t, "")
		lock, err := NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, RecommendedAlgo, nil, filepath.Base(testFile))
		require.NoError(t, err)

		err = os.WriteFile(testFile, []byte("corrupted"), 0644)
		require.NoError(t, err)

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})
}
