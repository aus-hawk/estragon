package logging

import (
	"io"
	"log"
	"os"
)

type multiWriter struct {
	logFile io.Writer
}

func (m multiWriter) Write(p []byte) (n int, err error) {
	n, err = m.logFile.Write(p)
	if err != nil {
		return
	}

	n, err = os.Stderr.Write(p)
	return
}

var InfoLogger, ErrorLogger *log.Logger

func SetLogFile(w io.Writer, verbose bool) {
	var writer io.Writer = multiWriter{w}
	ErrorLogger = log.New(writer, "ERROR ", log.LstdFlags)

	if !verbose {
		writer = w
	}
	InfoLogger = log.New(writer, "INFO ", log.LstdFlags)
}
