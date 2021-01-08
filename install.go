package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"bytes"

	"golang.org/x/oauth2"
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
	p := MakePersister()

	data, err := p.ReadTokens()
	if err != nil {
		return fmt.Errorf("can't read tokens: %v", err)
	}

	
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	sad := new(SavedAuthData)
	if err := decoder.Decode(sad); err != nil {
		return fmt.Errorf("can't decode json payload into SavedAuthData: %v", err)
	}
	token := &sad.SavedToken

	log.Println("using", *token, "will now try to download", args)

	if !token.Valid() {
		log.Println("InstallBinTargets has invalid token")
	}

	ctx := context.Background()

	conf := &oauth2.Config{
	ClientID: sad.ClientID,
	ClientSecret: sad.ClientSecret,
	Scopes: []string{
		// Add more scopes here as needed.
		"https://www.googleapis.com/auth/devstorage.read_only",
	},
	Endpoint: oauth2.Endpoint{
		TokenURL: "https://oauth2.googleapis.com/token",
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
	},
	// don't need I think
	RedirectURL:  "urn:ietf:wg:oauth:2.0:oob",
	}

	client := conf.Client(ctx, token)

	for _, wantedbin := range args {
		finalurl := "https://" + path.Join(urlbase, runtime.GOOS, runtime.GOARCH, wantedbin)
		localpath := filepath.Join(targetpath, wantedbin)
		log.Println(finalurl, " -> ", localpath)

		if err := copyUrl(client, finalurl, localpath); err != nil {
			return fmt.Errorf("InstallBinTargets can't GET %s to %s: %v", finalurl, localpath, err)
		}

		if err := os.Chmod(localpath, 0755); err != nil {
			fmt.Errorf("InstallBinTargets can't make %s executable: %v", localpath, err)
		}
	}

	// get the token to save?
	oatrans, ok := client.Transport.(*oauth2.Transport)
	if  !ok {
		return fmt.Errorf("can't get updated token because client.Transport is not oauth2")
	}
	
	newtoken, err  := oatrans.Source.Token()
	if err != nil {
		return fmt.Errorf("can't get updated token to save: %v", err)
	}

	sad.SavedToken = *newtoken
	if err := SaveOauthToken(sad); err != nil {
		return fmt.Errorf("can't get update SavedAuthData: %v", err)
	}
	return nil
}
