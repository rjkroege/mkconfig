package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"golang.org/x/oauth2"
)

// We need to save the client id and secret along with the refresh token.

type SavedAuthData struct {
	ClientID string
	ClientSecret string
	SavedToken oauth2.Token
}


type Persister interface {
ReadTokens() ([]byte, error)
WriteTokens(data []byte) error
}


// SaveOauthToken writes out an OAuth token.
func SaveOauthToken(sad *SavedAuthData) error {
	buffy := &bytes.Buffer{}
	encoder := json.NewEncoder(buffy)

	if err := encoder.Encode(sad); err != nil {
		return fmt.Errorf("can't encode SavedAuthData: %v", err)
	}

	p := MakePersister()

	if err := p.WriteTokens(buffy.Bytes()); err != nil {
		return fmt.Errorf("SaveOauthToken can't write tokens: %v", err)
	}

	return nil
}
