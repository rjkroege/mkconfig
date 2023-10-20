package main

import (
	"flag"
	"io"
	"log"
	"os"
)

// defaultTargetPath returns the default target path for the platform.
// I am just going to use /usr/local/bin everywhere. Always.
func defaultTargetPath() string {
	return "/usr/local/bin"
}

// Keep sorted.
// Consider using Kong here?
var accountsetup = flag.Bool("accountsetup", false, "do GCP account setup")
var bootstrap = flag.Bool("bootstrap", false, "do GCP bootstrap")
var clientidfile = flag.String("clientid", "client_info.json", "the client id json file")
var genbindeps = flag.String("bindeps", "", "generate binary deps for mk")
var genmkvars = flag.Bool("vars", false, "print mk vars")
var linuxpkg = flag.Bool("linuxpkg", false, "produces a pkgnotes list for the missing system packages")
var scriptspath = flag.String("scriptspath", "./tools", "pull configuration to this dir")
var targetpath = flag.String("targetpath", defaultTargetPath(), "install binaries here")
var verbose = flag.Bool("log", false, "print more detailed logging messages")

// Makes the state for the mkfile
func main() {
	flag.Parse()
	args := flag.Args()

	// By default, discard all log data during operation unless
	// something goes wrong and needs to be reported.
	if !*verbose {
		log.SetOutput(io.Discard)
	}

	log.Println("mkconfig was executed")

	if *genmkvars {
		log.Println("mkconfig doing printMkVars")
		printMkVars()
	} else if *genbindeps != "" {
		log.Println("mkconfig should generate deps", args)
		if err := genBinDeps(*genbindeps, args); err != nil {
			log.Fatalf("can't generate deps %v", err)
		}
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
		// TODO(rjk): This feature would become obsolete once I switch to new setup/build scheme.
	} else {
		if err := InstallBinTargets(*targetpath, args); err != nil {
			log.SetOutput(os.Stderr)
			log.Fatalf("can't install targets: %v", err)
		}
	}
	os.Exit(0)
}
