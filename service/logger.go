package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
)

const LogFileName = "internal.log"
const LogPrefix = "Gibber::Server	"

// for transaction requests logging
var logWriter io.Writer
var logger *log.Logger
var logInit sync.Once

func initLogger() (err error) {
	projectRootPath := ProjectRootPath()
	if err != nil {
		err = fmt.Errorf("getting project root path failed: %s", err)
		return
	}
	logDir := projectRootPath + "internal/generated/"
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		os.Mkdir(logDir, 0755)
	}
	txnLogFile := logDir + LogFileName
	file, err := os.OpenFile(txnLogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		err = fmt.Errorf("opening log file failed: %s", err)
		return
	}
	logWriter = io.MultiWriter(file)
	logger = log.New(logWriter, LogPrefix, log.Ldate|log.Ltime|log.Lmicroseconds|log.LUTC|log.Lshortfile)
	return
}

func WriteLog(log string, params ...interface{}) {
	logInit.Do(func() {
		if err := initLogger(); err != nil {
			panic(fmt.Sprintf("error while initializing internal logger: %s", err))
		}
	})
	logger.Printf(log, params)
}

func GetLogger() *log.Logger {
	logInit.Do(func() {
		if err := initLogger(); err != nil {
			panic(fmt.Sprintf("error while initializing internal logger: %s", err))
		}
	})
	return logger
}

func WriteLogAndReturnError(log string, params ...interface{}) error {
	WriteLog(log, params)
	return fmt.Errorf(log, params)
}