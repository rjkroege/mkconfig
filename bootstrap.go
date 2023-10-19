package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/rjkroege/gocloud/config"
	git "gopkg.in/src-d/go-git.v4"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// BootstrapGcpNode configures a GCP node. This is executed by the
// `gocloud` tool via the -bootstrap flag to setup node state such as ssh
// keys and git access.
func BootstrapGcpNode(targetpath, scriptspath string) error {
	nb, err := config.GetNodeMetadata(config.NewNodeDirectMetadataClient())
	if err != nil {
		return fmt.Errorf("problem with fetching node metadata: %v", err)
	}

	// Get user name from GCP metadata above.
	// Become configured user.
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

	// Fetch mk
	if err := InstallBinTargets(targetpath, []string{"mk"}); err != nil {
		return fmt.Errorf("can't install mk to %q: %v", targetpath, err)
	}
	if err := os.Chown(filepath.Join(targetpath, "mk"), uid, gid); err != nil {
		return fmt.Errorf("can't chown mk to %s: %v", nb["username"], err)
	}
	log.Println("installed mk")

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

	// Setup rclone support.
	if err := setupRclone(userinfo.HomeDir, uid, gid, nb); err != nil {
		return err
	}

	// fix up suoders
	sudoersentry := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL\n", nb["username"])
	suoderspath := filepath.Join("/etc/sudoers.d", nb["username"])
	if err := ioutil.WriteFile(suoderspath, []byte(sudoersentry), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", suoderspath, err)
	}
	log.Println(suoderspath, "made")

	if err := writegitcred(userinfo, nb["githost"], nb["username"], nb["gitcredential"], uid, gid); err != nil {
		return err
	}

	// Get git tree. Setup in ~username/tools/scripts with binaries in
	// /usr/local/bin
	// TODO(rjk): This could also be configurable.
	clonepath := scriptspath
	chownpath := scriptspath
	if !filepath.IsAbs(clonepath) {
		chownpath = filepath.Join(userinfo.HomeDir, scriptspath)
		clonepath = filepath.Join(userinfo.HomeDir, scriptspath, "scripts")
	}
	if err := os.MkdirAll(clonepath, 0755); err != nil {
		return fmt.Errorf("can't make scripts path %q: %v", clonepath, err)
	}

	githost := nb["githost"]
	gitcred := nb["gitcredential"]

	log.Println("gitcred", gitcred, "githost", githost)

	_, err = git.PlainClone(clonepath, false, &git.CloneOptions{
		URL:      githost,
		Progress: os.Stdout,
		Auth: &githttp.BasicAuth{
			Username: nb["username"],
			Password: gitcred,
		},
		Depth: 4,
	})
	if err != nil {
		return fmt.Errorf("can't checkout clonepath path %q: %v", clonepath, err)
	}
	log.Println("git fetched")
	if err := recursiveChown(chownpath, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", chownpath, err)
	}
	log.Println("recursiveChown scripts")

	// Exec 'mk' here (as different username)
	// I can do this with su
	// exec mk

	// One way to proceed with the knowing when I've finished setting up the
	// node is to write some kind of status. I had the idea of writing a
	// metadata value here. This seems problematic: it's tedious to write the
	// metadata service. Why don't I run something that I can poll for? When
	// I can connect to it, the node is up? What should I run? Instead, I
	// will poll for `ssh` connectivity and config from `ssh` instead.
	return nil
}

func writegitcred(userinfo *user.User, githost, username, gitcred string, uid, gid int) error {
	// fix up git credentials
	gitcredpath := filepath.Join(userinfo.HomeDir, ".git-credentials")
	// TODO(rjk): read the site from configuration.

	// TODO(rjk): I should check that this is valid all the way back in gocloud
	// before sending it to the remote machine.
	giturl, err := url.Parse(githost)
	if err != nil {
		return fmt.Errorf("can't parse githhost  %q: %v", githost, err)
	}
	giturl.User = url.UserPassword(username, gitcred)
	// Strip the path at the end.
	giturl.Path = ""
	giturl.RawQuery = ""
	giturl.Fragment = ""

	if err := ioutil.WriteFile(gitcredpath, []byte(giturl.String()), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", gitcredpath, err)
	}
	if err := recursiveChown(gitcredpath, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", gitcredpath, err)
	}
	log.Println("recursiveChown .git-credentials")
	return nil
}

func recursiveChown(path string, uid, gid int) error {
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err := os.Chown(path, uid, gid); err != nil {
			return fmt.Errorf("can't chown mk to %d: %v", uid, err)
		}
		return nil
	})
	return nil
}

func setupRclone(homedir string, uid, gid int, nb config.NodeMetadata) error {
	// Setup rclone configuration  including for Docker volume use.
	const (
		rclonedockerconfig = "/var/lib/docker-plugins/rclone/config"
		rclonedockercache  = "/var/lib/docker-plugins/rclone/cache"
	)
	rclonepath := filepath.Join(homedir, ".config", "rclone")

	for _, pth := range []string{rclonepath, rclonedockerconfig, rclonedockercache} {
		if err := os.MkdirAll(pth, 0755); err != nil {
			return fmt.Errorf("can't make path: %q: %v", pth, err)
		}
		log.Printf("%q made", pth)
	}

	rclonefilepath := filepath.Join(rclonepath, "rclone.conf")
	if err := ioutil.WriteFile(rclonefilepath, []byte(nb["rcloneconfig"]), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", rclonefilepath, err)
	}
	log.Printf("%q made", rclonefilepath)

	dockerrclonefilepath := filepath.Join(rclonedockerconfig, "rclone.conf")
	if err := ioutil.WriteFile(dockerrclonefilepath, []byte(nb["rcloneconfig"]), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", dockerrclonefilepath, err)
	}
	log.Printf("%q made", dockerrclonefilepath)

	if err := recursiveChown(filepath.Join(homedir, ".config"), uid, gid); err != nil {
		return fmt.Errorf("can't chown .config: %v", err)
	}
	log.Println("recursiveChown .config")

	return nil
}
