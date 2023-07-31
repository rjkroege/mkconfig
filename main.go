package main

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
)

// defaultTargetPath returns the default target path for the platform.
// I am just going to use /usr/local/bin everywhere. Always.
func defaultTargetPath() string {
	return "/usr/local/bin"
}

var targetpath = flag.String("targetpath", defaultTargetPath(), "install binaries here")
var scriptspath = flag.String("scriptspath", "./tools", "pull configuration to this dir")
var bootstrap = flag.Bool("bootstrap", false, "do GCP bootstrap")
var accountsetup = flag.Bool("accountsetup", false, "do GCP account setup")
var verbose = flag.Bool("log", false, "print more detailed logging messages")
var genmkvars = flag.Bool("vars", false, "print mk vars")
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
	} else if *linuxpkg {
		log.Println("CheckLinuxPackagesInstalled")
		if err := CheckLinuxPackagesInstalled(args); err != nil {
			log.Fatalf("can't determine missing packages: %v\n", err)
		}
	} else if *bootstrap {
		log.Println("BootstrapGcpNode")
		if err := BootstrapGcpNode(*targetpath, *scriptspath); err != nil {
			log.Fatalf("can't bootstrap node: %v\n", err)
		}
	} else if *accountsetup {
		log.Println("SetupGcpAccount")
		if err := SetupGcpAccount(*targetpath, *scriptspath); err != nil {
			log.Fatalf("can't bootstrap node: %v\n", err)
		}
	} else {
		if err := InstallBinTargets(*targetpath, args); err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("can't install targets: %v", err)
		}
	}
	os.Exit(0)
}
