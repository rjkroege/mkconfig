package main

import (
	"os"
	"path/filepath"
	//	"path"
	"fmt"
	"io/ioutil"
	"log"
	"os/user"
	"strconv"

	"github.com/rjkroege/gocloud/config"
)

// SetupGcpAccount is a subset of bootstrap for building out an account
// on a GCP node when the node is actually getting built by GCP's gcloud
// tool.
// TODO(rjk): It's conceivable that there is code duplication here that I
// could remove.
func SetupGcpAccount(targetpath, scriptspath string) error {
	nb, err := config.GetNodeMetadata(config.NewNodeDirectMetadataClient())
	if err != nil {
		return fmt.Errorf("problem with fetching node metadata: %v", err)
	}

	// Get infos about the users.
	userinfo, err := user.Lookup(nb["username"])
	if err != nil {
		return fmt.Errorf("can't find user %s: %v", nb["username"], err)
	}

	// Code works only on UNIX. TODO(rjk): generalize as needed.
	uid, err := strconv.Atoi(userinfo.Uid)
	if err != nil {
		return fmt.Errorf("can't make numeric uid %s: %v", userinfo.Uid, err)
	}
	gid, err := strconv.Atoi(userinfo.Gid)
	if err != nil {
		return fmt.Errorf("can't make numeric gid %s: %v", userinfo.Gid, err)
	}
	log.Println("uid", uid, "gid", gid)

	// Make a home directory
	homedir := userinfo.HomeDir
	if err := os.MkdirAll(homedir, 0755); err != nil {
		return fmt.Errorf("can't make path: %q: %v", homedir, err)
	}
	log.Println("homedir made")

	// This is going to be slow.
	if err := recursiveChown(homedir, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", homedir, err)
	}
	log.Println("recursiveChown homedir")

	// Setup ssh.
	sshdir := filepath.Join(userinfo.HomeDir, ".ssh")
	if err := os.MkdirAll(sshdir, 0755); err != nil {
		return fmt.Errorf("can't make path: %q: %v", sshdir, err)
	}
	log.Println(".ssh made")

	authkeypath := filepath.Join(sshdir, "authorized_keys")
	if err := ioutil.WriteFile(authkeypath, []byte(nb["sshkey"]), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", authkeypath, err)
	}
	log.Println(".ssh/authorized_keys made")

	if err := recursiveChown(sshdir, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", sshdir, err)
	}
	log.Println("recursiveChown .ssh")

	// fix up suoders
	sudoersentry := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL\n", nb["username"])
	suoderspath := filepath.Join("/etc/sudoers.d", nb["username"])
	if err := ioutil.WriteFile(suoderspath, []byte(sudoersentry), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", suoderspath, err)
	}
	log.Println(suoderspath, "made")

	return nil
}
