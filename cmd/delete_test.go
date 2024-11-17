package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRunDelete(t *testing.T) {
	// Create a unique temporary directory for this test
	tempDir := filepath.Join(os.TempDir(), fmt.Sprintf("grabit-test-%d", time.Now().UnixNano()))
	err := os.MkdirAll(tempDir, 0755)
	assert.Nil(t, err)
	defer os.RemoveAll(tempDir)

	// Create lock file in the unique directory
	testfilepath := filepath.Join(tempDir, "grabit.lock")
	content := `
    [[Resource]]
    Urls = ['http://localhost:123456/test.html']
    Integrity = 'sha256-asdasdasd'
    Tags = ['tag1', 'tag2']
`
	err = os.WriteFile(testfilepath, []byte(content), 0644)
	assert.Nil(t, err)

	// Execute delete command
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"-f", testfilepath, "delete", "http://localhost:123456/test.html"})
	err = cmd.Execute()
	assert.Nil(t, err)
}
