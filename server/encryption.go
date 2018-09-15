package server

import (
	"crypto"
	"hash"
	"sync"
)

var sha512Hash hash.Hash
var initHash sync.Once

func initSHA512Hash() {
	sha512Hash = crypto.SHA512.New()
}

func GetSHA512Encrypted(content string) (hashedContent string) {
	initHash.Do(initSHA512Hash)
	sha512Hash.Write([]byte(content))
	hashedContent = string(sha512Hash.Sum(nil))
	sha512Hash.Reset() // resetting the hash builder otherwise it will keep appending to currently hashed data
	return
}
