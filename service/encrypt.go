package service

import (
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

// generates the hash of the given string, using given cost (salt is created automatically by bcrypt)
func GenerateHash(content string) (hashContent string, err error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(content), 10)
	if err != nil {
		err = fmt.Errorf("hash generation failed: %s", err)
		return
	}
	hashContent = string(hash)
	return
}

// returns nil on match, error otherwise
func MatchHashAndPlainText(hash, plainText string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainText))
}
