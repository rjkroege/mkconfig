package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/oauth2"
)

func getFromJson(filename string) (*oauth2.Config, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open file %s: %v", filename, err)
	}
	defer fd.Close()

	decoder := json.NewDecoder(fd)
	var config oauth2.Config

	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("can't decode %s: %v", filename, err)
	}
	return &config, nil
}

// MakePersistentToken gets an OAuth2 token and stores for later use.
// TODO(rjk): Doing the dance for a GCE instance will have to be different.
func MakePersistentToken(clientidfile string) error {
	ctx := context.Background()

	// Setup oauth2.Config with the client id to get auth and refresh tokens.
	conf, err := getFromJson(clientidfile)
	if err != nil {
		return fmt.Errorf("can't read client identity %s: %v", clientidfile, err)
	}

	conf.Scopes = []string{
		// Add more scopes here as needed.
		"https://www.googleapis.com/auth/devstorage.read_only",
	}
	conf.Endpoint = oauth2.Endpoint{
		TokenURL: "https://oauth2.googleapis.com/token",
		AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
	}
	conf.RedirectURL = "urn:ietf:wg:oauth:2.0:oob"

	log.Println(conf)

	// Redirect user to consent page to ask for permission
	// for the scopes specified above.
	url := conf.AuthCodeURL("state", oauth2.AccessTypeOffline)
	fmt.Printf("Visit the URL for the auth dialog: %v\n", url)

	// Use the authorization code that is pushed to the redirect
	// URL. Exchange will do the handshake to retrieve the
	// initial access token. The HTTP Client returned by
	// conf.Client will refresh the token as necessary.
	var code string
	if _, err := fmt.Scan(&code); err != nil {
		return fmt.Errorf("can't read the authorization code %v", err)
	}

	// Use the custom HTTP client when requesting a token.
	httpClient := &http.Client{Timeout: 2 * time.Second}
	ctx = context.WithValue(ctx, oauth2.HTTPClient, httpClient)

	log.Println("config", conf)

	tok, err := conf.Exchange(ctx, code)
	if err != nil {
		return fmt.Errorf("can't complete oauth2 exchange for token %v", err)
	}

	log.Println("retrieved a token via oauth exchange", tok)
	log.Println("now writing the token to keychain or file")

	sad := &SavedAuthData{
		ClientID: conf.ClientID,
		ClientSecret: conf.ClientSecret,
		SavedToken: *tok,
	}
	if err := SaveOauthToken(sad); err != nil {
		return fmt.Errorf("can't save new SavedAuthData: %v", err)
	}
	return nil
}

