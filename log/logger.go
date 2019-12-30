package log

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

const (
	repo = "gibber/"
)

const (
	FileName = "debug.log"
	Prefix   = "Gibber::Server	"
)

// for transaction requests logging
var logWriter io.Writer
var logger *log.Logger
var logInit sync.Once

func initLogger() (err error) {
	projectRootPath := projectRootPath()
	logDir := projectRootPath + "generated/"
	if _, err = os.Stat(logDir); os.IsNotExist(err) {
		err = os.Mkdir(logDir, 0755)
		if err != nil {
			err = fmt.Errorf("error creating log directory %s: %s", logDir, err)
			return
		}
	}
	txnLogFile := logDir + FileName
	file, err := os.OpenFile(txnLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		err = fmt.Errorf("opening log file failed: %s", err)
		return
	}
	logWriter = io.MultiWriter(file)
	logger = log.New(logWriter, Prefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile)
	return
}

func WriteLog(log string, params ...interface{}) {
	logInit.Do(func() {
		if err := initLogger(); err != nil {
			panic(fmt.Sprintf("error while initializing internal logger: %s", err))
		}
	})
	logger.Print(log, params)
}

func Logger() *log.Logger {
	logInit.Do(func() {
		if err := initLogger(); err != nil {
			panic(fmt.Sprintf("error while initializing internal logger: %s", err))
		}
	})
	return logger
}

func WriteLogAndReturnError(log string, params ...interface{}) error {
	WriteLog(log, params)
	return fmt.Errorf(log, params...)
}

func projectRootPath() (path string) {
	_, fileStr, _, _ := runtime.Caller(0)
	rootPath := strings.Split(filepath.Dir(fileStr), repo)
	return rootPath[0] + repo
}