package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func addGoDep(newbins []string, bin, binrepo, binarchive string) []string {
	for _, goos := range []string{"linux", "darwin"} {
		archs := []string{}
		switch goos {
		case "linux":
			archs = []string{"amd64", "arm", "arm64"}
		case "darwin":
			archs = []string{"amd64",  "arm64"}
		}
		for _, goarch := range archs {
			archivetarget := filepath.Join(binarchive, goos, goarch, bin)
			fmt.Printf("%s: %s\n", archivetarget, binrepo)
			newbins = append(newbins, archivetarget)
		}
	}

	return newbins
}

func addSwiftDep(newbins []string, bin, binrepo, binarchive string) []string {
	for _, goarch := range []string{"amd64", "arm64"} {
		archivetarget := filepath.Join(binarchive, "darwin", goarch, bin)
		fmt.Printf("%s: %s\n", archivetarget, binrepo)
		newbins = append(newbins, archivetarget)
	}
	return newbins
}

func findPkgRoot(pth string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(pth, ".git")); err == nil {
			return pth, nil
		}
		if pth == "/" {
			return "", fmt.Errorf("no .git in any parent")
		}
		pth = filepath.Dir(pth)
	}
	// Should never get here.
	return "", nil
}

func isSwift(pkgroot string) bool {
	_, err := os.Stat(filepath.Join(pkgroot, "Package.swift"))
	return err == nil
}

func isGo(pkgroot string) bool {
	_, err := os.Stat(filepath.Join(pkgroot, "go.mod"))
	return err == nil
}

// genBinDeps assumes that we run inside of mk and generates a set of
// dependencies for the new style Go building approach.
func genBinDeps(binarchive string, paths []string) error {
	newbins := make([]string, 0)

	for _, pth := range paths {
		// I might call Abs here to make the path sane

		pkgroot, err := findPkgRoot(pth)
		if err != nil {
			return fmt.Errorf("binary %q is not a Git-tracked toos: %v", pth, err)
		}

		bin := filepath.Base(pth)
		deproot := filepath.Join(pkgroot, ".git", "HEAD")

		switch {
		case isSwift(pkgroot):
			newbins = addSwiftDep(newbins, bin, deproot, binarchive)
		case isGo(pkgroot):
			newbins = addGoDep(newbins, bin, deproot, binarchive)
		default:
			return fmt.Errorf("binary %q does not use a supported language", pth)
		}
	}

	// Print newbins
	fmt.Printf("\nnewbins = \\\n	")
	fmt.Printf(strings.Join(newbins, " \\\n	"))

	return nil
}
