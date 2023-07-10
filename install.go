package main

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"

	oauth "golang.org/x/oauth2/google"
)

const urlbase = "storage.googleapis.com/boot-tools-liqui-org"

// copyUrl copies the url using client to path. In essence, wget
func copyUrl(client *http.Client, url string, ofn string) error {
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("copyUrl can't GET %s: %v", url, err)
	}

	// On linux, I need to unlink first.
	os.Remove(ofn)

	ofd, err := os.Create(ofn)
	if err != nil {
		return fmt.Errorf("copyUrl can't open output %s: %v", ofn, err)
	}
	defer ofd.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("copyUrl http sad: %s", resp.Status)
	}

	if _, err := io.Copy(ofd, resp.Body); err != nil {
		return fmt.Errorf("copyUrl Copy %s -> %s failed: %v", url, ofn, err)
	}
	return nil
}

// InstallBinTargets creates an authenticated http client and uses it to
// download all of the desired targets. It looks for an appropriate GCS
// auth token in keychain or in a local file.
func InstallBinTargets(targetpath string, args []string) error {
	ctx := context.Background()

	client, err := oauth.DefaultClient(ctx,
		"https://www.googleapis.com/auth/devstorage.read_only")

	if err != nil {
		return fmt.Errorf("no DefaultClient %v", err)
	}

	if err := os.MkdirAll(targetpath, 0755); err != nil {
		return fmt.Errorf("can't have a target path %q: %v", targetpath, err)
	}

	for _, wantedbin := range args {
		localpath := filepath.Join(targetpath, wantedbin)
		if finalurl, plainfile := isDataFile(urlbase, wantedbin); plainfile {
			log.Println(finalurl, " -> ", localpath)
			if err := copyUrl(client, finalurl, localpath); err != nil {
				return fmt.Errorf("InstallBinTargets can't GET %s to %s: %v", finalurl, localpath, err)
			}
			if err := os.Chmod(localpath, 0644); err != nil {
				fmt.Errorf("InstallBinTargets can't set %q perms: %v", localpath, err)
			}
			continue
		}

		finalurl := "https://" + path.Join(urlbase, runtime.GOOS, runtime.GOARCH, wantedbin)
		log.Println(finalurl, " -> ", localpath)

		if err := copyUrl(client, finalurl, localpath); err != nil {
			return fmt.Errorf("InstallBinTargets can't GET %s to %s: %v", finalurl, localpath, err)
		}

		if err := os.Chmod(localpath, 0755); err != nil {
			fmt.Errorf("InstallBinTargets can't make %s executable: %v", localpath, err)
		}
	}

	return nil
}

func isDataFile(urlbase, wantedbin string) (string, bool) {
	switch filepath.Ext(wantedbin) {
	case ".ttf", "otf":
		return "https://" + path.Join(urlbase, "fonts", wantedbin), true
	case ".1":
		return "https://" + path.Join(urlbase, "mans", wantedbin), true
	default:
		return "", false
	}
}

// TarXZF natively implements 'tar xzf -` from ifd, writing the
// files to targetpath.
func TarXZF(targetpath string, ifd io.Reader) error {
	if err := os.MkdirAll(targetpath, 0755); err != nil {
		return fmt.Errorf("can't create dir %q: %v", targetpath, err)
	}

	zr, err := gzip.NewReader(ifd)
	if err != nil {
		return fmt.Errorf("can't create unzipper: %v", err)
	}

	tr := tar.NewReader(zr)
	for {
		h, err := tr.Next()
		if err != nil && err == io.EOF {
			log.Println("eof?")
			return nil
		} else if err != nil {
			return fmt.Errorf("no header for tar: %v", err)
		}

		path := filepath.Join(targetpath, h.Name)
		log.Printf("tar %q -> %q", h.Name, path)

		switch h.Typeflag {
		case tar.TypeReg:
			dfd, err := os.Create(path)
			if err != nil {
				return fmt.Errorf("can't create dest path %q: %v", path, err)
			}

			if _, err := io.Copy(dfd, tr); err != nil {
				dfd.Close()
				return fmt.Errorf("can't write path %q: %v", path, err)
			}
			dfd.Close()
			// TODO(rjk): Fix mode, timestamps, xattr as needed.
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(h.Mode)); err != nil {
				return fmt.Errorf("can't make dir %q: %v", path, err)
			}
		default:
			// TODO(rjk): Extend as needed to additional types.
			log.Println("can't deal with %q, unsupported entry type", path)
		}
	}
}
