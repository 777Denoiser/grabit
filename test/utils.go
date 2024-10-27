// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.

package test

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TmpFile(t *testing.T, content string) string {
	dir := t.TempDir()
	name := filepath.Join(dir, fmt.Sprintf("test%d", rand.Int()))

	// Write content directly without keeping file handle open
	err := os.WriteFile(name, []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}

	return name
}

func TmpDir(t *testing.T) string {
	dir, err := os.MkdirTemp(t.TempDir(), "test")
	if err != nil {
		log.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

func HttpHandler(handler http.HandlerFunc) (int, *httptest.Server) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	s := httptest.NewUnstartedServer(http.HandlerFunc(handler))
	s.Listener.Close()
	s.Listener = l
	s.Start()
	return l.Addr().(*net.TCPAddr).Port, s
}

// TestHttpHandler creates a new HTTP server and returns the port and serves
// the given content. Its lifetime is tied to the given testing.T object.
func TestHttpHandler(content string, t *testing.T) int {
	handler := func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(content))
		if err != nil {
			t.Fatal(err)
		}
	}
	port, server := HttpHandler(handler)
	t.Cleanup(func() { server.Close() })
	return port
}

// AssertFileContains asserts that the file at the given path exists and
// contains the given content.
func AssertFileContains(t *testing.T, path, content string) {
	assert.FileExists(t, path)
	fileContent, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, content, string(fileContent))
}
