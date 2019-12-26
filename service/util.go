package service

import (
	"encoding/json"
	"math/rand"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const ProjectName = "gibber/"

const (
	letterBytes   = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

func ProjectRootPath() (path string) {
	_, fileStr, _, _ := runtime.Caller(0)
	rootPath := strings.Split(filepath.Dir(fileStr), ProjectName)
	return rootPath[0] + ProjectName
}

func GetMap(data interface{}) (dataMap map[string]interface{}, err error) {
	if data == nil {
		return
	}
	bytes, err := json.Marshal(data)
	if err != nil {
		return
	}
	err = json.Unmarshal(bytes, &dataMap)
	return
}

func RandomString(length int) string {
	b := make([]byte, length)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := length-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}
	return string(b)
}
