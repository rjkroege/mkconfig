package main

import (
	"path/filepath"
	"os"
//	"path"
	"os/user"
	"fmt"
	"strconv"
	"net/http"
	"io/ioutil"
	"log"

	git "gopkg.in/src-d/go-git.v4"
	githttp "gopkg.in/src-d/go-git.v4/plumbing/transport/http"
)

// This code is opinionated.
func BootstrapGcpNode(targetpath, scriptspath string) error {
	// Get user
	// User account (can I read stuffs from the gcp to configure?)
	username, err := readStingFromMetadata("username")
	if err != nil {
		return fmt.Errorf("can't get username %v", err)
	}
	log.Println("username", username)
	
	// Become configured user.
	userinfo, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("can't find user %s: %v", username, err)
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
		return fmt.Errorf("can't chown mk to %s: %v", username, err)
	}
	log.Println("installed mk")

	// Get git credential
	// User account (can I read stuffs from the gcp to configure?)
	gitcred, err := readStingFromMetadata("gitcredential")
	if err != nil {
		return fmt.Errorf("can't get getcredential %v", err)
	}
	log.Println("gitcred", gitcred)

	// Get git tree. Setup in ~username/tools/scripts with binaries in
	// /usr/local/bin
	clonepath := scriptspath
	chownpath := scriptspath
	if !filepath.IsAbs(clonepath) {
		chownpath = filepath.Join(userinfo.HomeDir, scriptspath)
		clonepath = filepath.Join(userinfo.HomeDir, scriptspath, "scripts")
	}
	if err := os.MkdirAll(clonepath, 0755); err != nil {
		return fmt.Errorf("can't make scripts path %q: %v", clonepath, err)
	}

	// TODO(rjk): Read this from configuration eventually.
	const url = "https://git.liqui.org/rjkroege/scripts.git"

	_, err = git.PlainClone(clonepath, false, &git.CloneOptions{
		URL:            url  ,
		Progress:          os.Stdout,
		    Auth: &githttp.BasicAuth{
   		     Username: "abc123", // anything except an empty string
  		      Password: gitcred,
   		 },
	})
	if err != nil {
		return fmt.Errorf("can't checkout clonepath path %q: %v", clonepath, err)
	}
	log.Println("git fetched")
	if err := recursiveChown(chownpath, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", chownpath, err)
	}
	log.Println("recursiveChown scripts")

	// Setup ssh.
	sshdir := filepath.Join(userinfo.HomeDir, ".ssh")
	if err := os.MkdirAll(sshdir, 0755); err != nil {
		return fmt.Errorf("can't make path: %q: %v", sshdir, err)
	}
	log.Println(".ssh made")
	
	sshval, err := readStingFromMetadata("sshkey")
	if err != nil {
		return fmt.Errorf("can't get sshkey %v", err)
	}
	authkeypath := filepath.Join(sshdir, "authorized_keys")
	if err := ioutil.WriteFile(authkeypath, []byte(sshval), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", authkeypath, err)
	}
	log.Println(".ssh/authorized_keys made")

	if err := recursiveChown(sshdir, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", sshdir, err)
	}
	log.Println("recursiveChown .ssh")

	// Setup rclone configuration.
	rclonepath := filepath.Join(userinfo.HomeDir, ".config", "rclone")
	if err := os.MkdirAll(rclonepath, 0755); err != nil {
		return fmt.Errorf("can't make path: %q: %v", rclonepath, err)
	}
	log.Printf("%q made", rclonepath)

	rcloneval, err := readStingFromMetadata("rcloneconfig")
	if err != nil {
		return fmt.Errorf("can't get rcloneconfig %v", err)
	}
	rclonefilepath := filepath.Join(rclonepath, "rclone.conf")
	if err := ioutil.WriteFile(rclonefilepath, []byte(rcloneval), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", rclonefilepath, err)
	}
	log.Printf("%q made", rclonefilepath)

	if err := recursiveChown(filepath.Join(userinfo.HomeDir, ".config"), uid, gid); err != nil {
		return fmt.Errorf("can't chown .config: %v", err)
	}
	log.Println("recursiveChown .config")

	// fix up suoders
	sudoersentry := fmt.Sprintf("%s ALL=(ALL) NOPASSWD: ALL\n", username)
	suoderspath := filepath.Join("/etc/sudoers.d", username)
	if err := ioutil.WriteFile(suoderspath, []byte(sudoersentry), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", suoderspath, err)
	}
	log.Println(suoderspath, "made")

	// fix up git credentials
	gitcredpath := filepath.Join(userinfo.HomeDir, ".git-credentials")
	// TODO(rjk): read this from configuration.
	gitcredentry := fmt.Sprintf("https://%s:%s@git.liqui.org", username, gitcred)
	if err := ioutil.WriteFile(gitcredpath, []byte(gitcredentry), 0600); err != nil {
		return fmt.Errorf("can't write  %q: %v", gitcredpath, err)
	}
	if err := recursiveChown(gitcredpath, uid, gid); err != nil {
		return fmt.Errorf("can't chown %q: %v", gitcredpath, err)
	}
	log.Println("recursiveChown .git-credentials")

	// Exec 'mk' here (as different username)
	// I can do this with su
	// exec mk

	return nil
}

const metabase = "http://metadata.google.internal/computeMetadata/v1/instance/attributes/"

func readStingFromMetadata(entry string) (string, error) {
	path := metabase + entry

	client := &http.Client{}
	req, err := http.NewRequest("GET", path, nil)
	req.Header.Add("Metadata-Flavor", "Google")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("can't fetch metadata %v: %v", path, err)
	}
	
	buffy, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read metadata body %v: %v", path, err)
	}
	return string(buffy), nil
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

