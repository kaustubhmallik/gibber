package service

import (
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	LogoFilePath = "assets/logo.txt"
	ProjectName  = "gibber/"
)

func PrintLogo() (err error) {
	filePath := ProjectRootPath() + LogoFilePath
	logoData, err := ioutil.ReadFile(filePath)
	if err != nil {
		Logger().Printf("reading logo file %s filed failed: %s", filePath, err)
		return
	}
	Logger().Println(string(logoData[:]))
	return
}

func ProjectRootPath() (path string) {
	_, fileStr, _, _ := runtime.Caller(0)
	rootPath := strings.Split(filepath.Dir(fileStr), ProjectName)
	return rootPath[0] + ProjectName
}
