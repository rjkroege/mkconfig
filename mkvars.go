package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
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

func isCos() bool {
	if le, err := linuxFlavour(); err == nil && le == "cos" {
		return true
	}
	return false
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

	// TODO(rjk): Verify that an rc install will install enough to support this.
	// Add commands that only make sense if I have plan9 setup.
	p9path := "/usr/local/plan9"
	envp9path, exists := os.LookupEnv("PLAN9")
	if exists {
		p9path = envp9path
	}

	platformtargets := make([]string,0)
	platformtargets = append(platformtargets, runtime.GOOS, runtime.GOARCH)

	if fs, err := os.Stat(p9path); err == nil && fs.IsDir() {
		platformtargets = append(platformtargets, "p9p")
	}

	// TODO(rjk): Add check for corp as needed and add to platformtargets
	// This is for MacOS. I need something different for Linux.
	if fs, err := os.Stat("/usr/local/bin/gcert"); err == nil && fs.Mode() & 0111 != 0 {
		platformtargets = append(platformtargets, "corp")
	} else if fs, err := os.Stat("/usr//bin/gcert"); err == nil && fs.Mode() & 0111 != 0 {
		platformtargets = append(platformtargets, "corp")
	}

	// Am I Debian or Alpine. Based it on presence of package management
	if runtime.GOOS == "linux" {
		flavour, err := linuxFlavour()
		if err != nil {
			log.Printf("no package system: %v", err)
		} else {
			fmt.Println("packagesystem", "=", flavour)
		}
	}

	fmt.Println("platformtargets", "=", strings.Join(platformtargets, "_"))

	tp := defaultTargetPath()
	if err := os.MkdirAll(tp, 0755); err != nil {
		log.Printf("can't make %s: %v\n", tp, err)
	}
	fmt.Println("targetpath", "=", tp)
}
