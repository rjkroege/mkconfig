package main

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os/exec"
	"strings"
)

// CheckAlpinePackagesInstalled generates pkgnotes variable by checking
// which of the desired packages are already installed and creates an
// Alpine Linux suffixed list of the missing packages that should be
// installed.
func CheckAlpinePackagesInstalled(args []string) error {
	log.Println(args)
	cmdline := []string{"apk", "info", "-e"}
	cmdline = append(cmdline, args...)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)

	// We wouldn't be here if there wasn't a command. A non-0 exit status
	// means that one or more of the packages is missing.
	out, _ := cmd.Output()

	gotmap := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		gotmap[string(bytes.TrimSpace(line))] = struct{}{}
	}

	log.Println("map of installed", gotmap)

	need := make([]string, 0)
	for _, s := range args {
		if _, ok := gotmap[s]; !ok {
			need = append(need, s+".alpine")
		}
	}
	log.Println("need array", need)

	// Generate the mk variable.
	fmt.Println("pkgnotes = ", strings.Join(need, " "))
	return nil
}

// CheckDebianPackagesInstalled generates pkgnotes variable by checking
// which of the desired packages are already installed and creates a
// Debian Linux suffixed list of the missing packages that should be
// installed.
func CheckDebianPackagesInstalled(args []string) error {
	log.Println(args)
	cmdline := []string{"apt", "-qq", "list"}
	cmdline = append(cmdline, args...)
	cmd := exec.Command(cmdline[0], cmdline[1:]...)

	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("cmd.Output fail: %v", err)
	}

	// Output looks like "package version platform status". One line per
	// package. Status is last column. Assumption: I split into lines.
	gotmap := make(map[string]struct{})
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Bytes()
		cols := bytes.Split(line, []byte{' '})
		i := bytes.IndexByte(cols[0], '/')
		pkg := string(cols[0][0:i])
		if len(cols) > 3 && bytes.Index(cols[3], []byte("installed")) >= 0 {
			gotmap[pkg] = struct{}{}
		}
	}

	log.Println("gotmap", gotmap)
	need := make([]string, 0)
	for _, s := range args {
		if _, ok := gotmap[s]; !ok {
			need = append(need, s+".debian")
		}
	}
	log.Println("need array", need)

	// Generate the mk variable.
	fmt.Println("pkgnotes = ", strings.Join(need, " "))
	return nil
}

// CheckLinuxPackagesInstalled creates the pkgnotes variable.
func CheckLinuxPackagesInstalled(args []string) error {
	fl, err := linuxFlavour()
	if err != nil {
		return fmt.Errorf("unsupported linux: %v", err)
	}

	switch {
	case fl == "debian":
		return CheckDebianPackagesInstalled(args)
	case fl == "alpine":
		return CheckAlpinePackagesInstalled(args)
	case fl == "cos":
		// No packages to install on Cos. 
		return nil
	}
	// Shouldn't get here.
	return nil
}
