package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// defaultTargetPath returns the default target path for the platform.
func defaultTargetPath() string {
	s := *targetpath
	if runtime.GOOS != "darwin" {
		h := os.ExpandEnv("$HOME")
		if h != "" {
			s = filepath.Join(h, "bin")
		}
	}
	return s
}

var targetpath = flag.String("targetpath", "/usr/local/bin", "help message for flagname")
var verbose = flag.Bool("log", false, "print more detailed logging messages")
var genmkvars = flag.Bool("vars", false, "print mk vars")
var mktoken = flag.Bool("token", false, "create and persist authtoken")
var clientidfile = flag.String("clientid", "client_info.json", "the client id json file")
var linuxpkg = flag.Bool("linuxpkg", false, "produces a pkgnotes list for the missing system packages")

// Makes the state for the mkfile
func main() {
	flag.Parse()
	args := flag.Args()

	// By default, discard all log data during operation unless
	// something goes wrong and needs to be reported.
	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}

	log.Println("mkconfig was executed")

	if *genmkvars {
		log.Println("mkconfig doing printMkVars")
		printMkVars()
	} else if *mktoken {
		if err := MakePersistentToken(*clientidfile); err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("can't create auth token: %v", err)
		}
	} else if *linuxpkg {
		log.Println("CheckLinuxPackagesInstalled")
		if err := CheckLinuxPackagesInstalled(args); err != nil {
			log.Fatalf("can't determine missing packages: %v\n", err)
		}
	} else {
		if err := InstallBinTargets(defaultTargetPath(), args); err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("can't install targets: %v", err)
		}
	}
	os.Exit(0)
}
