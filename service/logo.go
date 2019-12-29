package service

import (
	"io/ioutil"
)

const LogoFilePath = "assets/logo.txt"

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
