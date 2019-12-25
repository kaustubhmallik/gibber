package service

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
)

const ProjectName = "gibber/"

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
