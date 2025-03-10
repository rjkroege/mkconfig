package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func addGoZigDep(newbins, deproots []string, bin, binarchive string) []string {
	for _, goos := range []string{"linux", "darwin"} {
		archs := []string{}
		switch goos {
		case "linux":
			archs = []string{"amd64", "arm", "arm64"}
		case "darwin":
			archs = []string{"amd64", "arm64"}
		}
		for _, goarch := range archs {
			archivetarget := filepath.Join(binarchive, goos, goarch, bin)
			for _, dr := range deproots {
				fmt.Printf("%s: %s\n", archivetarget, dr)
			}
			newbins = append(newbins, archivetarget)
		}
	}

	return newbins
}

func addSwiftDep(newbins, deproots []string, bin, binarchive string) []string {
	for _, goarch := range []string{"amd64", "arm64"} {
		archivetarget := filepath.Join(binarchive, "darwin", goarch, bin)
		for _, dr := range deproots {
			fmt.Printf("%s: %s\n", archivetarget, dr)
		}
		newbins = append(newbins, archivetarget)
	}
	return newbins
}

func findPkgRoot(pth, wantedfile string) (string, error) {
	for {
		if _, err := os.Stat(filepath.Join(pth, wantedfile)); err == nil {
			return pth, nil
		}
		if pth == "/" {
			return "", fmt.Errorf("no %s in any parent", wantedfile)
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

func isZig(pkgroot string) bool {
	_, err := os.Stat(filepath.Join(pkgroot, "build.zig"))
	return err == nil
}

// genBinDeps assumes that we run inside of mk and generates a set of
// dependencies for the new style Go building approach.
func genBinDeps(binarchive string, paths []string) error {
	newbins := make([]string, 0)

	for _, pth := range paths {
		// I might call Abs here to make the path sane

		pkgroot, err := findPkgRoot(pth, ".git")
		if err != nil {
			return fmt.Errorf("binary %q is not a Git-tracked tool: %v", pth, err)
		}

		bin := filepath.Base(pth)
		deproots := []string{
			filepath.Join(pkgroot, ".git", "HEAD"),
			filepath.Join(pkgroot, ".git", "index"),
		}

		switch {
		case isSwift(pkgroot):
			newbins = addSwiftDep(newbins, deproots, bin, binarchive)
		case isGo(pkgroot):
			newbins = addGoZigDep(newbins, deproots, bin, binarchive)
		case isZig(pkgroot):
			newbins = addGoZigDep(newbins, deproots, bin, binarchive)
		default:
			return fmt.Errorf("binary %q does not use a supported language", pth)
		}
	}

	// Print newbins
	fmt.Printf("\nnewbins = \\\n	")
	fmt.Printf(strings.Join(newbins, " \\\n	"))

	return nil
}
