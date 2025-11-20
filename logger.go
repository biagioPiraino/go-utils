package goutils

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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

func (severity Severity) Severity() string {
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
	// includes severities 0-2
	errorsFilepath string
	// includes severities 3-5
	logsFilepath string
}

var (
	Logger *Blogger
	once   sync.Once
)

// initialising logger by passing absolute path value
func InitialiseLogger(logDirectory string, logFilename string, errorFilename string) {
	once.Do(func() {
		// same logging file in case error not specified
		if errorFilename == "" {
			errorFilename = logFilename
		}

		logsFilepath, errorsFilepath, err := createOutputFiles(logDirectory, logFilename, errorFilename)
		if err != nil {
			log.Fatalf(Critical.Severity()+": unable to initialise logger, exiting.\nError details: %v", err)
		}

		Logger = &Blogger{
			errorsFilepath: errorsFilepath,
			logsFilepath:   logsFilepath,
		}

		LogRequest(
			Trace,
			LogEvent{
				ProcessType: OsProcess,
				ProcessId:   strconv.Itoa(os.Getpid()),
				Event:       "Logger initialised successfully",
			})
	})
}

func LogRequest(severity Severity, process LogEvent) error {
	var file *os.File
	var err error

	switch severity {
	case Emergency:
	case Alert:
	case Critical:
		file, err = os.OpenFile(Logger.errorsFilepath, os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
	default:
		file, err = os.OpenFile(Logger.logsFilepath, os.O_RDWR|os.O_APPEND, 0644)
		if err != nil {
			return err
		}
	}

	defer closeFile(file)
	log.SetFlags(0)
	log.SetOutput(file)
	log.Printf("%s,%s,%s,%s,%s",
		severityName[severity], nowUTC(), processTypeName[process.ProcessType], process.ProcessId, process.Event)
	file.Sync()
	return nil
}

func closeFile(file *os.File) {
	err := file.Close()
	if err != nil {
		log.Printf("An error occurred while closing the logging file %s\nError details: %v", file.Name(), err)
	}
}

func createOutputFiles(logDirectory string, logFilename string, errorFilename string) (string, string, error) {
	logsFileTimeExt := strings.Join([]string{todayUTC(), "-", logFilename, ".csv"}, "")
	errorsFileTimeExt := strings.Join([]string{todayUTC(), "-", errorFilename, ".csv"}, "")

	// creating directory where only app can write and external user can only read and traverse
	if err := os.MkdirAll(logDirectory, 0755); err != nil {
		return "", "", err
	}

	// create files, only app the write and read, all the others can read only
	logsFilepath := filepath.Join(logDirectory, logsFileTimeExt)
	logFile, err := os.OpenFile(logsFilepath, os.O_CREATE, 0644)
	if err != nil {
		return "", "", err
	}
	defer closeFile(logFile)

	errorsFilepath := filepath.Join(logDirectory, errorsFileTimeExt)
	errorFile, err := os.OpenFile(errorsFilepath, os.O_CREATE, 0644)
	if err != nil {
		return "", "", err
	}
	defer closeFile(errorFile)
	return logsFilepath, errorsFilepath, nil
}

func todayUTC() string {
	return time.Now().UTC().Format("2006-01-02")
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}
