package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

// Code is from https://github.com/gtank/cryptopasta

// Hash generates a hash of data using HMAC-SHA-512/256. The tag is intended to
// be a natural-language string describing the purpose of the hash, such as
// "hash file for lookup key" or "master secret to client secret".  It serves
// as an HMAC "key" and ensures that different purposes will have different
// hash output. This function is NOT suitable for hashing passwords.
func hashHelper(tag string, data []byte) []byte {
	h := hmac.New(sha512.New512_256, []byte(tag))
	h.Write(data)
	return h.Sum(nil)
}

// Encrypt encrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Output takes the
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Encrypt(plaintext []byte, key *[32]byte) (ciphertext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = io.ReadFull(rand.Reader, nonce)
	if err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts data using 256-bit AES-GCM.  This both hides the content of
// the data and provides a check that it hasn't been altered. Expects input
// form nonce|ciphertext|tag where '|' indicates concatenation.
func Decrypt(ciphertext []byte, key *[32]byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, errors.New("malformed ciphertext")
	}

	return gcm.Open(nil,
		ciphertext[:gcm.NonceSize()],
		ciphertext[gcm.NonceSize():],
		nil,
	)
}

func MakeKeyFromPassphrase(phrase []byte) *[32]byte {
	passphrase := []byte(phrase)
	slicekey := hashHelper("passphrase", passphrase)

	// I might want to assert that slicekey is 32 bytes.
	var key = new([32]byte)
	copy((*key)[:], slicekey[0:32])

	log.Println(len(key), key)
	return key
}

type FilePersister struct{}

func MakePersister() Persister {
	return new(FilePersister)
}

func (_ *FilePersister) ReadTokens() ([]byte, error) {
	// Read the encrypted config into memory.
	filename, _ := mkFileName()
	rdr, err := readFile(filename)
	if err != nil {
		return nil, fmt.Errorf("maybe run mkconfig -token because can't open confg file %s: %v", filename, err)
	}

	encrypted, err := ioutil.ReadAll(rdr)
	if err != nil {
		return nil, fmt.Errorf("can't read confg file %s: %v", filename, err)
	}

	// Get the passphrase. Yes, putting it in the environment is insecure. Sorry.
	// At least the tokens are safe at rest.
	passphrase, exists := os.LookupEnv("passphrase")
	if !exists {
		return nil, fmt.Errorf("no $passphrase set")
	}

	// Make a key
	key := MakeKeyFromPassphrase([]byte(passphrase))

	// Decrypt file
	decrypted, err := Decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("can't decrypt tokens: %v", err)
	}

	return decrypted, nil

}

func (_ *FilePersister) WriteTokens(data []byte) error {
	passphrase, exists := os.LookupEnv("passphrase")
	if !exists {
		return fmt.Errorf("no $passphrase set")
	}

	// Make a key
	key := MakeKeyFromPassphrase([]byte(passphrase))

	encrypted, err := Encrypt(data, key)
	if err != nil {
		return fmt.Errorf("can't encrypt tokens: %v", err)
	}

	// Write enctypted data.
	filename, dir := mkFileName()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("can't make dir %s: %v", dir, err)
	}

	ofd, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("can't open token store %s: %v", filename, err)
	}

	log.Println(encrypted)
	_, werr := ofd.Write(encrypted)
	if cerr := ofd.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return fmt.Errorf("can't write tokens to %s: %v", filename, werr)
	}
	return nil
}

func mkFileName() (string, string) {
	home := "/home/root"
	userinfo, err := user.Current()
	if err == nil {
		home = userinfo.HomeDir
	}

	dirname := filepath.Join(home, ".config", "mkconfig")
	filename := filepath.Join(home, ".config", "mkconfig", "auth.json")

	return filename, dirname
}

func readFile(filename string) (io.Reader, error) {
	fd, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("can't open auth file: %v", err)
	}
	return fd, nil
}
