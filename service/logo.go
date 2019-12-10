package service

import (
	"fmt"
	"io/ioutil"
)

const LogoFilePath = "service/logo.txt"

func PrintLogo() {
	filePath := ProjectRootPath() + LogoFilePath
	logoData, err := ioutil.ReadFile(filePath)
	if err != nil {
		Logger().Printf("reading logo file %s filed failed: %s", filePath, err)
	}
	Logger().Println(string(logoData[:]))
	fmt.Println(string(logoData[:]))
}
