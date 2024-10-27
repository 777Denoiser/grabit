// Copyright (c) 2023 Cisco Systems, Inc. and its affiliates
// All rights reserved.

package internal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	toml "github.com/pelletier/go-toml/v2"
)

var COMMENT_PREFIX = "//"

// Lock represents a grabit lockfile.
type Lock struct {
	path string
	conf config
}

type config struct {
	Resource []Resource
}

func NewLock(path string, create bool) (*Lock, error) {
	if !create {
		_, err := os.Stat(path)
		if err != nil {
			return nil, err
		}
	}

	var conf config
	if !create {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("cannot open file '%s': %s", path, err)
		}
		err = toml.Unmarshal(data, &conf)
		if err != nil {
			return nil, fmt.Errorf("invalid lock file '%s': %s", path, err)
		}
	}

	return &Lock{path: path, conf: conf}, nil
}

func (l *Lock) AddResource(paths []string, algo string, tags []string, filename string) error {
	for _, u := range paths {
		if l.Contains(u) {
			return fmt.Errorf("resource '%s' is already present", u)
		}
	}
	r, err := NewResourceFromUrl(paths, algo, tags, filename)
	if err != nil {
		return err
	}
	l.conf.Resource = append(l.conf.Resource, *r)
	return nil
}

func (l *Lock) DeleteResource(path string) {
	newStatements := []Resource{}
	for _, r := range l.conf.Resource {
		if !r.Contains(path) {
			newStatements = append(newStatements, r)
		}
	}
	l.conf.Resource = newStatements
}

const NoFileMode = os.FileMode(0)

// strToFileMode converts a string to a os.FileMode.
func strToFileMode(perm string) (os.FileMode, error) {
	if perm == "" {
		return NoFileMode, nil
	}
	parsed, err := strconv.ParseUint(perm, 8, 32)
	if err != nil {
		return NoFileMode, err
	}
	return os.FileMode(parsed), nil
}

// Download gets all the resources in this lock file and moves them to
// the destination directory.
func (l *Lock) Download(dir string, tags []string, notags []string, perm string) error {
	if stat, err := os.Stat(dir); err != nil || !stat.IsDir() {
		return fmt.Errorf("'%s' is not a directory", dir)
	}
	mode, err := strToFileMode(perm)
	if err != nil {
		return fmt.Errorf("'%s' is not a valid permission definition", perm)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Filter in the resources that have all the required tags.
	tagFilteredResources := []Resource{}
	if len(tags) > 0 {
		for _, r := range l.conf.Resource {
			hasAllTags := true
			for _, tag := range tags {
				hasTag := false
				for _, rtag := range r.Tags {
					if tag == rtag {
						hasTag = true
						break
					}
				}
				if !hasTag {
					hasAllTags = false
					break
				}
			}
			if hasAllTags {
				tagFilteredResources = append(tagFilteredResources, r)
			}
		}
	} else {
		tagFilteredResources = l.conf.Resource
	}
	// Filter out the resources that have any 'notag' tag.
	filteredResources := []Resource{}
	if len(notags) > 0 {
		for _, r := range tagFilteredResources {
			hasTag := false
			for _, notag := range notags {
				for _, rtag := range r.Tags {
					if notag == rtag {
						hasTag = true
					}
				}
			}
			if !hasTag {
				filteredResources = append(filteredResources, r)
			}
		}
	} else {
		filteredResources = tagFilteredResources
	}

	total := len(filteredResources)
	if total == 0 {
		return fmt.Errorf("nothing to download")
	}
	errorCh := make(chan error, total)
	for _, r := range filteredResources {
		resource := r
		go func() {
			err := resource.Download(dir, mode, ctx)
			errorCh <- err
		}()
	}
	done := 0
	errs := []error{}
	for range total {
		err = <-errorCh
		if err != nil {
			errs = append(errs, err)
		} else {
			done += 1
		}
	}
	if done == total {
		return nil
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// Save this lock file to disk.
func (l *Lock) Save() error {
	res, err := toml.Marshal(l.conf)
	if err != nil {
		return err
	}

	dir := filepath.Dir(l.path)
	tmpFile := filepath.Join(dir, fmt.Sprintf(".%s.tmp", filepath.Base(l.path)))

	if err := os.WriteFile(tmpFile, res, 0644); err != nil {
		return err
	}

	if err := os.Rename(tmpFile, l.path); err != nil {
		os.Remove(tmpFile) // Clean up temp file if rename fails
		return err
	}

	return nil
}

// Contains returns true if this lock file contains the
// given resource url.
func (l *Lock) Contains(url string) bool {
	for _, r := range l.conf.Resource {
		for _, u := range r.Urls {
			if url == u {
				return true
			}
		}
	}
	return false
}
