// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.

package internal

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/carlmjohnson/requests"
)

// Resource represents an external resource to be downloaded.
type Resource struct {
	Urls      []string
	Integrity string
	Tags      []string `toml:",omitempty"`
	Filename  string   `toml:",omitempty"`
}

func NewResourceFromUrl(urls []string, algo string, tags []string, filename string) (*Resource, error) {
	if len(urls) < 1 {
		return nil, fmt.Errorf("empty url list")
	}
	url := urls[0]
	ctx := context.Background()
	path, err := GetUrltoTempFile(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get url: %s", err)
	}
	defer os.Remove(path)
	integrity, err := getIntegrityFromFile(path, algo)
	if err != nil {
		return nil, fmt.Errorf("failed to compute ressource integrity: %s", err)
	}
	return &Resource{Urls: urls, Integrity: integrity, Tags: tags, Filename: filename}, nil
}

// getUrl downloads the given resource and returns the path to it.
func getUrl(u string, fileName string, ctx context.Context) (string, error) {
	_, err := url.Parse(u)
	if err != nil {
		return "", fmt.Errorf("invalid url '%s': %s", u, err)
	}
	log.Debug().Str("URL", u).Msg("Downloading")
	err = requests.
		URL(u).
		Header("Accept", "*/*").
		ToFile(fileName).
		Fetch(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to download '%s': %s", u, err)
	}
	log.Debug().Str("URL", u).Msg("Downloaded")
	return fileName, nil
}

// GetUrlToDir downloads the given resource to a temporary file and returns the path to it.
// Modify the GetUrlToDir function to ensure proper file cleanup
func GetUrlToDir(u string, targetDir string, ctx context.Context) 
(string, error) {
	// create temporary name in the target directory.
	h := sha256.New()
	h.Write([]byte(u))
	fileName := filepath.Join(targetDir, fmt.Sprintf(".%s", hex.EncodeToString(h.Sum(nil))))

	// Remove existing file if present
	os.Remove(fileName)

	// Create new file with immediate close
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %v", err)
	}
	file.Close()

	fileName, err = getUrl(u, fileName, ctx)
	if err != nil {
		os.Remove(fileName)
		return "", err
	}

	return fileName, nil
}

// GetUrlWithDir downloads the given resource to a temporary file and returns the path to it.
func GetUrltoTempFile(u string, ctx context.Context) (string, error) {
	file, err := os.CreateTemp("", "prefix")
	if err != nil {
		log.Fatal().Err(err)
	}
	fileName := file.Name()
	return getUrl(u, fileName, ctx)
}

func (l *Resource) Download(dir string, mode os.FileMode, ctx context.Context) error {
	ok := false
	algo, err := getAlgoFromIntegrity(l.Integrity)
	if err != nil {
		return err
	}
	var downloadError error = nil
	for _, u := range l.Urls {
		log.Debug().Str("URL", u).Msg("Downloading")

		// Download file in the target directory so that the call to
		// os.Rename is atomic.
		lpath, err := GetUrlToDir(u, dir, ctx)
		if err != nil {
			downloadError = err
			continue
		}
		err = checkIntegrityFromFile(lpath, algo, l.Integrity, u)
		if err != nil {
			return err
		}

		localName := ""
		if l.Filename != "" {
			localName = l.Filename
		} else {
			localName = path.Base(u)
		}
		resPath := filepath.Join(dir, localName)

		// Check if file exists and is valid
		if ValidateLocalFile(resPath, l.Integrity) {
			log.Debug().Msgf("Using existing validated file: %s", resPath)
			if mode != NoFileMode {
				if err := os.Chmod(resPath, mode.Perm()); err != nil {
					return err
				}
			}
			ok = true
			continue
		}

		// Download file with proper cleanup
		lpath, err := GetUrlToDir(u, dir, ctx)
		if err != nil {
			downloadError = fmt.Errorf("failed to download '%s': %v", u, err)
			continue
		}

		// Validate and move file
		if err := checkIntegrityFromFile(lpath, algo, l.Integrity, u); err != nil {
			os.Remove(lpath)
			downloadError = err
			continue
		}

		// Remove target file if it exists
		os.Remove(resPath)
		if err := os.Rename(lpath, resPath); err != nil {
			os.Remove(lpath)
			return err
		}

		if mode != NoFileMode {
			if err := os.Chmod(resPath, mode.Perm()); err != nil {
				return err
			}
		}
		ok = true
	}

	if !ok && downloadError != nil {
		return downloadError

	if !ok {
		if downloadError != nil {
			return downloadError
		}
		return err
	}
	return nil
}

func (l *Resource) Contains(url string) bool {
	for _, u := range l.Urls {
		if u == url {
			return true
		}
	}
	return false
}

func ValidateLocalFile(filePath string, expectedIntegrity string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	algo, err := getAlgoFromIntegrity(expectedIntegrity)
	if err != nil {
		return false
	}

	fileIntegrity, err := getIntegrityFromFile(filePath, algo)
	if err != nil {
		return false
	}

	return fileIntegrity == expectedIntegrity
}
