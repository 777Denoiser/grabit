package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"github.com/cisco-open/grabit/internal"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/cisco-open/grabit/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func getSha256Integrity(content string) string {
	hasher := sha256.New()
	hasher.Write([]byte(content))
	return fmt.Sprintf("sha256-%s", base64.StdEncoding.EncodeToString(hasher.Sum(nil)))
}

func TestRunDownload(t *testing.T) {
	content := `abcdef`
	contentIntegrity := getSha256Integrity(content)
	port := test.TestHttpHandler(content, t)
	testfilepath := test.TmpFile(t, fmt.Sprintf(`
	[[Resource]]
	Urls = ['http://localhost:%d/test.html']
	Integrity = '%s'

	[[Resource]]
	Urls = ['http://localhost:%d/test3.html']
	Integrity = '%s'
`, port, contentIntegrity, port, contentIntegrity))
	outputDir := test.TmpDir(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download", "--dir", outputDir})
	err := cmd.Execute()
	assert.Nil(t, err)
	for _, file := range []string{"test.html", "test3.html"} {
		test.AssertFileContains(t, fmt.Sprintf("%s/%s", outputDir, file), content)
	}
}

func TestRunDownloadWithTags(t *testing.T) {
	content := `abcdef`
	contentIntegrity := getSha256Integrity(content)
	port := test.TestHttpHandler(content, t)
	testfilepath := test.TmpFile(t, fmt.Sprintf(`
	[[Resource]]
	Urls = ['http://localhost:%d/test.html']
	Integrity = '%s'
	Tags = ['tag']

	[[Resource]]
	Urls = ['http://localhost:%d/test2.html']
	Integrity = '%s'
	Tags = ['tag1', 'tag2']
`, port, contentIntegrity, port, contentIntegrity))
	outputDir := test.TmpDir(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download", "--tag", "tag", "--dir", outputDir})
	err := cmd.Execute()
	assert.Nil(t, err)
	for _, file := range []string{"test.html"} {
		test.AssertFileContains(t, fmt.Sprintf("%s/%s", outputDir, file), content)
	}
}

func TestRunDownloadWithoutTags(t *testing.T) {
	content := `abcdef`
	contentIntegrity := getSha256Integrity(content)
	port := test.TestHttpHandler(content, t)
	testfilepath := test.TmpFile(t, fmt.Sprintf(`
	[[Resource]]
	Urls = ['http://localhost:%d/test.html']
	Integrity = '%s'
	Tags = ['tag']

	[[Resource]]
	Urls = ['http://localhost:%d/test2.html']
	Integrity = '%s'
	Tags = ['tag1', 'tag2']
`, port, contentIntegrity, port, contentIntegrity))
	outputDir := test.TmpDir(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download", "--notag", "tag", "--dir", outputDir})
	err := cmd.Execute()
	assert.Nil(t, err)
	for _, file := range []string{"test2.html"} {
		test.AssertFileContains(t, fmt.Sprintf("%s/%s", outputDir, file), content)
	}
}

func TestRunDownloadMultipleErrors(t *testing.T) {
	testfilepath := test.TmpFile(t, `
	[[Resource]]
	Urls = ['http://localhost:1234/test.html']
	Integrity = 'sha256-unused'

	[[Resource]]
	Urls = ['http://cannot-be-resolved.no:12/test.html']
	Integrity = 'sha256-unused'
`)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download"})
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "failed to download")
	assert.Contains(t, err.Error(), "connection refused")
	assert.Contains(t, err.Error(), "no such host")
}

func TestRunDownloadFailsIntegrityTest(t *testing.T) {
	content := `abcdef`
	port := test.TestHttpHandler(content, t)
	testfilepath := test.TmpFile(t, fmt.Sprintf(`
	[[Resource]]
	Urls = ['http://localhost:%d/test.html']
	Integrity = 'sha256-bogus'
`, port))
	outputDir := test.TmpDir(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download", "--dir", outputDir})
	err := cmd.Execute()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "integrity mismatch")
}

main
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
		lock, err := internal.NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, internal.RecommendedAlgo, nil, filepath.Base(testFile))
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
		lock, err := internal.NewLock(lockPath, true)
		require.NoError(t, err)

		err = lock.AddResource([]string{testUrl}, internal.RecommendedAlgo, nil, filepath.Base(testFile))
		require.NoError(t, err)

		err = os.WriteFile(testFile, []byte("corrupted"), 0644)
		require.NoError(t, err)

		err = lock.Download(tmpDir, nil, nil, "")
		require.NoError(t, err)
	})

func TestRunDownloadTriesAllUrls(t *testing.T) {
	content := `abcdef`
	contentIntegrity := getSha256Integrity(content)
	port := test.TestHttpHandler(content, t)
	testfilepath := test.TmpFile(t, fmt.Sprintf(`
	[[Resource]]
	Urls = ['http://cannot-be-resolved.no:12/test.html', 'http://localhost:%d/test.html']
	Integrity = '%s'
`, port, contentIntegrity))
	outputDir := test.TmpDir(t)
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "download", "--dir", outputDir})
	err := cmd.Execute()
	assert.Nil(t, err)
	for _, file := range []string{"test.html"} {
		test.AssertFileContains(t, fmt.Sprintf("%s/%s", outputDir, file), content)
	}
main
}
