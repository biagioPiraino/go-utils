package goutils

// TODO: review tests and understand how to setup test and run
// TODO: publish v0.1.0 on public

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// setup severities
type Severity int

const (
	Emergency Severity = iota
	Alert
	Critical
	Notice
	Debug
	Trace
)

var severityName = map[Severity]string{
	Emergency: "EMERGENCY",
	Alert:     "ALERT",
	Critical:  "CRITICAL",
	Notice:    "NOTICE",
	Debug:     "DEBUG",
	Trace:     "TRACE",
}

func (severity Severity) ToString() string {
	return severityName[severity]
}

// setup logger
type ProcessType int

const (
	OsProcess ProcessType = iota
	GoRoutineProcess
	RequestProcess
)

var processTypeName = map[ProcessType]string{
	OsProcess:        "Operating System",
	GoRoutineProcess: "Goroutine",
	RequestProcess:   "Request",
}

func (p ProcessType) ToString() string {
	return processTypeName[p]
}

type LogEvent struct {
	ProcessType ProcessType
	ProcessId   string
	Event       string
}

type Blogger struct {
	// Leave file open to prevent overhead by keep open it
	// everytime a log is made.

	// To ensure resources are correclty closed on panic
	// remember to defer their closure during recovery
	errorsFile *os.File
	logsFile   *os.File

	errLogger *log.Logger // includes severities 0-2
	stdLogger *log.Logger // includes severities 3-5
}

func NewLogger(logDirectory string, logFilename string, errorFilename string) (*Blogger, error) {
	if errorFilename == "" {
		errorFilename = logFilename
	}

	logsFile, errorsFile, err := openOutputFiles(logDirectory, logFilename, errorFilename)
	if err != nil {
		return nil, err
	}

	logger := Blogger{
		logsFile:   logsFile,
		errorsFile: errorsFile,
		// new logger can be direclty initialised and assigned to a struct
		stdLogger: log.New(logsFile, "", 0),
		errLogger: log.New(errorsFile, "", 0),
	}

	logger.Log(
		Trace,
		LogEvent{ProcessType: OsProcess,
			ProcessId: strconv.Itoa(os.Getpid()),
			Event:     "Logger initialised successfully"})

	return &logger, nil
}

func (b *Blogger) Log(severity Severity, process LogEvent) {
	msg := fmt.Sprintf("%s,%s,%s,%s,%s",
		severityName[severity], nowUTC(), processTypeName[process.ProcessType], process.ProcessId, process.Event)

	switch severity {
	case Emergency, Alert, Critical:
		b.errLogger.Println(msg)
	default:
		b.stdLogger.Println(msg)
	}
}

func (b *Blogger) Close() {
	if b.errorsFile != nil {
		if err := b.errorsFile.Close(); err != nil {
			// log auto redirect to std err
			log.Printf("error while closing error logs file: %v")
		}
	}

	if b.logsFile != nil {
		if err := b.logsFile.Close(); err != nil {
			// log auto redirect to std err
			log.Printf("error while closing logs file: %v")
		}
	}
}

// private functions
func openOutputFiles(logDirectory string, logFilename string, errorFilename string) (*os.File, *os.File, error) {
	logsFileTimeExt := strings.Join([]string{todayUTC(), "-", logFilename, ".csv"}, "")
	errorsFileTimeExt := strings.Join([]string{todayUTC(), "-", errorFilename, ".csv"}, "")

	// creating directory where only app can write and external user can only read and traverse
	if err := os.MkdirAll(logDirectory, 0755); err != nil {
		return nil, nil, err
	}

	// create files, only app the write and read, all the others can read only
	logsFilepath := filepath.Join(logDirectory, logsFileTimeExt)
	logFile, err := os.OpenFile(logsFilepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, nil, err
	}

	errorsFilepath := filepath.Join(logDirectory, errorsFileTimeExt)
	errorFile, err := os.OpenFile(errorsFilepath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		logFile.Close()
		return nil, nil, err
	}
	return logFile, errorFile, nil
}

func todayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
