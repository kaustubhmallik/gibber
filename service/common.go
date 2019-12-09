package service

import (
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