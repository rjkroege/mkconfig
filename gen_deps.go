package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Cache map[string]string

// NewLocationCache makes a map of binary target name to path of Git HEAD
// file. This map is used to generate the dependencies below.
func NewLocationCache(roots []string) (Cache, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("no home dir? %v", err)
	}

	// binary target to its Git HEAD file mapping.
	cache := make(Cache)

	for _, root := range roots {
		// A root can use a ~ to mean my home directory for convenience.
		if root[0] == '~' {
			root = filepath.Join(home, root[1:])
		}

		ents, err := os.ReadDir(root)
		if err != nil {
			return nil, fmt.Errorf("%q is an invalid root for tool sources: %v", root, err)
		}

		for _, e := range ents {
			if _, err := os.Stat(filepath.Join(root, e.Name(), "go.mod")); err != nil {
				continue
			}
			stimulatingdep := filepath.Join(root, e.Name(), ".git/HEAD")
			if _, err := os.Stat(stimulatingdep); err != nil {
				continue
			}
			cache[e.Name()] = stimulatingdep
		}
	}
	return cache, nil
}

// genBinDeps assumes that we run inside of mk and generates a set of
// dependencies for the new style Go building approach.
func genBinDeps() error {
	// TODO(rjk): the roots of the location cache might be configurable.
	cache, err := NewLocationCache([]string{"~/tools", "~/tools/_builds"})
	if err != nil {
		return err
	}

	// The invoking mkfile needs to define $binarchive and $gobins. This
	// function uses gobins environment variable from the mkfile.
	gobins := strings.Split(os.Getenv("gobins"), " ")

	for _, bin := range gobins {
		binrepo, ok := cache[bin]
		if !ok {
			return fmt.Errorf("binary %q fails to be a Git-tracked Go tool")
		}

		for _, goos := range []string{"linux", "darwin"} {
			for _, goarch := range []string{"amd64", "arm", "arm64"} {
				fmt.Printf("$binarchive/%s/%s/%s: %s\n", goos, goarch, bin, binrepo)
			}
		}
	}
	return nil
}
