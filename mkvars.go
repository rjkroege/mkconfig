package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/rjkroege/gocloud/config"
)

// linuxFlavour determines the type of Linux that we're running on based on the package management system.
func linuxFlavour() (string, error) {
	if _, err := os.Stat("/usr/bin/apt"); err == nil {
		// We have apt present.
		return "debian", nil
	}

	if _, err := os.Stat("/sbin/apk"); err == nil {
		// We have apk present so Alpine
		return "alpine", nil
	}

	if _, err := os.Stat("/home/chronos"); err == nil {
		// Container OS or ChromeOS
		return "cos", nil
	}

	// TODO(rjk): gentoo, etc. Whatever I try next.
	return "", fmt.Errorf("can't determine Linux packaging scheme")
}

// printMkVars implements one of the two modes of mkconfig: printing
// mk variables
func printMkVars() {
	fmt.Println("GOOS", "=", runtime.GOOS)
	fmt.Println("GOARCH", "=", runtime.GOARCH)
	fmt.Println("suffix", "=", runtime.GOOS)

	// determines the home directory
	hd, err := os.UserHomeDir()
	if err != nil {
		log.Println("no home directory available: ", err)
	}
	fmt.Println("home", "=", hd)

	platformtargets := make([]string, 0)
	platformtargets = append(platformtargets, runtime.GOOS, runtime.GOARCH)

	// Am I Debian or Alpine. Based it on presence of package management
	if runtime.GOOS == "linux" {
		flavour, err := linuxFlavour()
		if err != nil {
			log.Printf("no package system: %v", err)
		} else {
			fmt.Println("packagesystem", "=", flavour)
		}
	}

	// TODO(rjk): Takes too long if not running on GCP. How can I make it faster?
	// How to know if a machine is a compute node
	if runtime.GOOS == "linux" {
		if config.RunningInGcp(config.NewNodeDirectMetadataClient()) {
			platformtargets = append(platformtargets, "gcp")
		}
	}

	fmt.Println("platformtargets", "=", strings.Join(platformtargets, "_"))

	tp := defaultTargetPath()
	if err := os.MkdirAll(tp, 0755); err != nil {
		log.Printf("can't make %s: %v\n", tp, err)
	}
	fmt.Println("targetpath", "=", tp)
}
