package main

import (
	"io"
	"log"
	"os"

	"github.com/alecthomas/kong"
)

// defaultTargetPath returns the default target path for the platform.
// I am just going to use /usr/local/bin everywhere. Always.
func defaultTargetPath() string {
	return "/usr/local/bin"
}

var CLI struct {
	Verbose      bool   `help:"Enable debugging logging."`

	Vars struct {
	} `cmd help:"print mk vars"`

	Bindeps struct {
		Args []string `arg:"" name:"args" help:"Deps directory and then packages for which to generate mk dependency data."`
	} `cmd help:"generate binary deps for mk"`

	Linuxpkg struct {
		Packages []string `arg:"" name:"packages" help:"List of packages to generate names for."`
	} `cmd help:"Produces a pkgnotes list for the missing system packages."`
	
	Bootstrap struct {
		Targetpath string `arg:"" name:"targetpath" help:"install binaries here"`
		Scriptspath string `arg:"" name:"scriptspath" help:"pull configuration to this dir"`
	} `cmd help:"do GCP bootstrap"`

	Accountsetup struct {
		Targetpath string `arg:"" name:"targetpath" help:"install binaries here"`
		Scriptspath string `arg:"" name:"scriptspath" help:"pull configuration to this dir"`
	} `cmd help:"setup an account on an GCP node"`
}


// Makes the state for the mkfile
func main() {
	ctx := kong.Parse(&CLI)

	// By default, discard all log data during operation unless
	// something goes wrong and needs to be reported.
	if !CLI.Verbose {
		log.SetOutput(io.Discard)
	}

	log.Println("mkconfig was executed")

	switch ctx.Command() {
	case "vars":
		log.Println("mkconfig doing printMkVars")
		printMkVars()
	case "bindeps <args>":
		log.Println("mkconfig should generate deps", CLI.Bindeps.Args)
		// TODO(rjk): This API surface should be fixed at sometime.
		if err := genBinDeps(CLI.Bindeps.Args[0], CLI.Bindeps.Args[1:]); err != nil {
			log.Fatalf("can't generate deps %v", err)
		}
	case "linuxpkg <packages>":
		log.Println("CheckLinuxPackagesInstalled")
		if err := CheckLinuxPackagesInstalled(CLI.Linuxpkg.Packages); err != nil {
			log.Fatalf("can't determine missing packages: %v\n", err)
		}
	case "bootstrap <targetpath> <scriptspath>":
		log.Println("BootstrapGcpNode")
		if err := BootstrapGcpNode(CLI.Bootstrap.Targetpath, CLI.Bootstrap.Scriptspath); err != nil {
			log.Fatalf("can't bootstrap node: %v\n", err)
		}
	case "accountsetup <targetpath> <scriptspath>":
		log.Println("SetupGcpAccount")
		if err := SetupGcpAccount(CLI.Accountsetup.Targetpath, CLI.Accountsetup.Scriptspath); err != nil {
			log.Fatalf("can't bootstrap node: %v\n", err)
		}
// TODO(rjk): This feature is obsolete once I switch to new setup/build
// scheme. But: hasn't that happened?
	}
	os.Exit(0)
}
