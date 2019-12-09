package service

import "golang.org/x/crypto/bcrypt"

// generates the hash of the given string, using given cost (salt is created automatically by bcrypt)
func GenerateHash(content string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(content), 10) // TODO: ignoring error for now
	return string(hash)
}

// returns nil on match, error otherwise
func MatchHashAndPlainText(hash, plainText string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plainText))
}
