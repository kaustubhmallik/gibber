package server

import "io/ioutil"

const LogoFilePath = "server/logo.txt"

func PrintLogo() {
	filePath := ProjectRootPath() + LogoFilePath
	logoData, err := ioutil.ReadFile(filePath)
	if err != nil {
		GetLogger().Printf("reading logo file %s filed failed: %s", filePath, err)
	}
	GetLogger().Println(string(logoData[:]))
}
