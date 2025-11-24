package goutils__test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	goutils "github.com/biagioPiraino/go-utils"
)

const (
	logsName   = "app_logs"
	errorsName = "app_errors"
)

// Helper to clean up artifacts after tests
func cleanup(dir string) {
	os.RemoveAll(dir)
}

// Helper to generate expected filenames based on the logic in logger.go
func getExpectedFilenames(dir, logName, errName string) (string, string) {
	date := time.Now().UTC().Format("2006-01-02")
	logFile := filepath.Join(dir, fmt.Sprintf("%s-%s.csv", date, logName))
	errFile := filepath.Join(dir, fmt.Sprintf("%s-%s.csv", date, errName))
	return logFile, errFile
}

// Test 1: Verify Enum String Conversions
// Ensures the maps for Severity and ProcessType return correct strings.
func TestEnumStringConversions(t *testing.T) {
	tests := []struct {
		severity     goutils.Severity
		expectedSev  string
		process      goutils.ProcessType
		expectedProc string
	}{
		{goutils.Emergency, "EMERGENCY", goutils.OsProcess, "Operating System"},
		{goutils.Debug, "DEBUG", goutils.GoRoutineProcess, "Goroutine"},
		{goutils.Trace, "TRACE", goutils.RequestProcess, "Request"},
	}

	for _, tc := range tests {
		if got := tc.severity.ToString(); got != tc.expectedSev {
			t.Errorf("Expected severity %s, got %s", tc.expectedSev, got)
		}
		if got := tc.process.ToString(); got != tc.expectedProc {
			t.Errorf("Expected process %s, got %s", tc.expectedProc, got)
		}
	}
}

// Test 2: Integration Test (Initialization, Routing, and Content)
// This checks if files are created and if logs go to the right place.
func TestLoggerIntegration(t *testing.T) {
	// 1. Setup Temporary Directory
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup(tempDir)

	// 2. Initialize Logger
	logger, err := goutils.NewLogger(tempDir, logsName, errorsName)
	if err != nil {
		t.Fatal("Logger was not initialized (nil)")
	}

	// 3. Generate Expected File Paths
	expectedLogPath, expectedErrPath := getExpectedFilenames(tempDir, logsName, errorsName)

	// 4. Test Routing: CRITICAL (Should go to Error File)
	critMsg := "Database connection failed"
	logger.Log(goutils.Critical, goutils.LogEvent{
		ProcessType: goutils.OsProcess,
		ProcessId:   "101",
		Event:       critMsg,
	})

	// 5. Test Routing: DEBUG (Should go to Standard Log File)
	debugMsg := "Calculation started"
	logger.Log(goutils.Debug, goutils.LogEvent{
		ProcessType: goutils.GoRoutineProcess,
		ProcessId:   "202",
		Event:       debugMsg,
	})

	// 6. Verify Error File Content
	contentErr, err := os.ReadFile(expectedErrPath)
	if err != nil {
		t.Fatalf("Could not read error file at %s: %v", expectedErrPath, err)
	}
	strContentErr := string(contentErr)
	if !strings.Contains(strContentErr, "CRITICAL") || !strings.Contains(strContentErr, critMsg) {
		t.Errorf("Error file missing expected content. Got:\n%s", strContentErr)
	}
	// Verify standard logs didn't leak into error file
	if strings.Contains(strContentErr, debugMsg) {
		t.Error("Error file contains Debug messages (leaked content).")
	}

	// 7. Verify Log File Content
	// Note: Initialization logs a TRACE event, so we expect that + our DEBUG message
	contentLog, err := os.ReadFile(expectedLogPath)
	if err != nil {
		t.Fatalf("Could not read log file at %s: %v", expectedLogPath, err)
	}
	strContentLog := string(contentLog)
	if !strings.Contains(strContentLog, "DEBUG") || !strings.Contains(strContentLog, debugMsg) {
		t.Errorf("Log file missing expected debug content. Got:\n%s", strContentLog)
	}
	// Verify critical logs didn't leak into standard file
	if strings.Contains(strContentLog, critMsg) {
		t.Error("Log file contains Critical messages (leaked content).")
	}
}

// Test 3: CSV Format Validation
// Ensures the CSV structure matches: Severity,Timestamp,ProcessType,ID,Message
func TestLogFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup(tempDir)

	// 2. Initialize Logger
	logger, err := goutils.NewLogger(tempDir, logsName, errorsName)
	if err != nil {
		t.Fatal("Logger was not initialized (nil)")
	}

	testEvent := "FormattingTest"
	logger.Log(goutils.Notice, goutils.LogEvent{
		ProcessType: goutils.RequestProcess,
		ProcessId:   "999",
		Event:       testEvent,
	})

	// Read the log file (path stored in the global Logger)
	content, _ := os.ReadFile(logger.LogsFile.Name())
	lines := strings.Split(string(content), "\n")

	// Find our specific line
	var foundLine string
	for _, line := range lines {
		if strings.Contains(line, testEvent) {
			foundLine = line
			break
		}
	}

	if foundLine == "" {
		t.Fatal("Could not find the formatting test log line")
	}

	// Parse CSV Line
	parts := strings.Split(foundLine, ",")
	if len(parts) < 5 {
		t.Fatalf("Log line does not have enough CSV fields: %s", foundLine)
	}

	// Validate Specific Fields
	if parts[0] != "NOTICE" {
		t.Errorf("Expected NOTICE, got %s", parts[0])
	}
	// Check Timestamp (RFC3339)
	_, err = time.Parse(time.RFC3339, parts[1])
	if err != nil {
		t.Errorf("Invalid timestamp format: %s", parts[1])
	}
	if parts[2] != "Request" { // Mapped from RequestProcess
		t.Errorf("Expected Request, got %s", parts[2])
	}
	if parts[3] != "999" {
		t.Errorf("Expected ProcessId 999, got %s", parts[3])
	}
	// Handle potential commas in the message by joining the rest
	msg := strings.Join(parts[4:], ",")
	if msg != testEvent {
		t.Errorf("Expected message %s, got %s", testEvent, msg)
	}
}

// Test 4: Concurrency Safety
// Runs multiple goroutines writing logs simultaneously to ensure no panics or race conditions.
func TestConcurrency(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "logger_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer cleanup(tempDir)

	logger, err := goutils.NewLogger(tempDir, logsName, errorsName)
	if err != nil {
		t.Fatal("Logger not initialised correclty")
	}

	var wg sync.WaitGroup
	routines := 50

	wg.Add(routines)
	for i := 0; i < routines; i++ {
		go func(val int) {
			defer wg.Done()
			logger.Log(goutils.Trace, goutils.LogEvent{
				ProcessType: goutils.GoRoutineProcess,
				ProcessId:   fmt.Sprintf("%d", val),
				Event:       "Concurrent write test",
			})
		}(i)
	}
	wg.Wait()
}
