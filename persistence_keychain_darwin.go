package main

import (
	"fmt"
	"os/user"

	"github.com/keybase/go-keychain"
)

// readKeyChain reads a value from the keychain identified by service and
// accessgroup for username and returns the read value, true if there was
// a read value and an error if one occurred.
func readKeyChain(service, username, accessgroup string) ([]byte, bool, error) {
	query := keychain.NewItem()

	// Generic password type. I want this kind
	query.SetSecClass(keychain.SecClassGenericPassword)

	// The service name. I'm using gcs.liqui.org. Which is sort of made-up
	query.SetService(service)

	// The name of the current user.
	query.SetAccount(username)

	// This is suppose to be the team id (from signing / notarization) with
	// .org.liqui.mkconfig appended. I have made it up.
	query.SetAccessGroup(accessgroup)

	// We only want one result
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, false,
			fmt.Errorf("tried to read keychain: %s,%s,%s didn't works: %v", service, username, accessgroup, err)
	} else if len(results) != 1 {
		return nil, false, nil
	}
	return results[0].Data, true, nil
}

// writeKeyChain writes data encrypted into KeyChain or backing file.
func writeKeyChain(service, username, accessgroup string, data []byte) error {
	item := keychain.NewItem()

	// Generic password type. I want this kind
	item.SetSecClass(keychain.SecClassGenericPassword)

	// The service name. I'm using gcs.liqui.org. Which is sort of made-up
	item.SetService(service)

	// The name of the current user.
	item.SetAccount(username)

	// This is suppose to be the team id (from signing / notarization) with
	// .org.liqui.mkconfig appended. I have made it up.
	item.SetAccessGroup(accessgroup)

	item.SetData(data)
	item.SetSynchronizable(keychain.SynchronizableNo)
	item.SetAccessible(keychain.AccessibleWhenUnlocked)

	if err := keychain.AddItem(item); err == keychain.ErrorDuplicateItem {
		// Try deleting the duplicate first.
		ditem := keychain.NewItem()
		ditem.SetSecClass(keychain.SecClassGenericPassword)
		ditem.SetService(service)
		ditem.SetAccount(username)
		if err := keychain.DeleteItem(item); err != nil {
			return fmt.Errorf("can't delete old password: %v", err)
		}

		if err := keychain.AddItem(item); err != nil {
			return fmt.Errorf("can't write keychain item: %v", err)
		}

	} else if err != nil {
		return fmt.Errorf("can't write keychain item: %v", err)
	}
	return nil
}

type KeyChainPersister struct{}

func MakePersister() Persister {
	return new(KeyChainPersister)
}

func  (_ *KeyChainPersister) WriteTokens(data []byte) error {
	userinfo, err := user.Current()
	if err != nil {
		return fmt.Errorf("can't get the user name: %v", err)
	}

	if err := writeKeyChain("gcsbin.liqui.org", userinfo.Username, "groovy.org.liqui.mkconfig", data); err != nil {
			return fmt.Errorf("can't write keychain: %v", err)
	}
	return nil

}

func  (_ *KeyChainPersister) ReadTokens() ( []byte, error) {
	userinfo, err := user.Current()
	if err != nil {
		return nil, fmt.Errorf("can't get the user name: %v", err)
	}

	data, exists, err := readKeyChain("gcsbin.liqui.org", userinfo.Username, "groovy.org.liqui.mkconfig")
	if err != nil {
			return nil, fmt.Errorf("can't read keychain: %v", err)
	} else if  !exists {
			return nil, fmt.Errorf("try running mkconfig -token -- no tokens in keychain: %v", err)
	}

	return data, nil
}
