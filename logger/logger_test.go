package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/go-gost/core/logger"
	"github.com/stretchr/testify/assert"
)

func TestLogger_Options(t *testing.T) {
	var buf bytes.Buffer
	name := "testLogger"
	log := NewLogger(
		NameOption(name),
		OutputOption(&buf),
		FormatOption(logger.JSONFormat),
		LevelOption(logger.DebugLevel),
	)

	log.Debug("debug message")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v", err)
	}

	assert.Equal(t, name, entry["logger"], "Logger name should be set")
	assert.Equal(t, "debug", entry["level"], "Log level should be debug")
	assert.Contains(t, entry["msg"], "debug message", "Log message not found")
	assert.Contains(t, entry, "caller", "Caller field should exist at debug level")
	assert.Contains(t, entry, "time", "Time field should exist")
}

func TestLogger_DefaultOptions(t *testing.T) {
	// Note: Default output is stderr, which is hard to capture without
	// redirecting os.Stderr. For this test, we'll rely on checking the level
	// and that methods don't panic, rather than capturing output.
	log := NewLogger()

	// Default level is Info
	assert.Equal(t, logger.InfoLevel, log.GetLevel(), "Default log level should be Info")

	// Default format is JSON (checked by trying to log and ensuring no panic, actual format check is harder without output capture)
	assert.NotPanics(t, func() {
		log.Info("test info")
	}, "Logging with default JSON format should not panic")
}

func TestLogger_TextFormat(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(
		OutputOption(&buf),
		FormatOption(logger.TextFormat),
		LevelOption(logger.InfoLevel),
	)

	log.Info("text format message")
	output := buf.String()

	assert.Contains(t, output, "level=info", "Log level not found in text output")
	assert.Contains(t, output, "msg=\"text format message\"", "Log message not found in text output")
	assert.Contains(t, output, "time=", "Time field not found in text output")
	// Caller is not included by default by logrus.TextFormatter, and our logger doesn't add it for text.
	assert.NotContains(t, output, "caller=", "Caller field should not exist in text output by default")
}

func TestLogger_JSONFormat_DisableHTMLEscape(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(
		OutputOption(&buf),
		FormatOption(logger.JSONFormat), // Default, but explicit for clarity
		LevelOption(logger.InfoLevel),
	)

	testHTMLMessage := "<hello & world>"
	log.Info(testHTMLMessage)

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v", err)
	}
	assert.Equal(t, testHTMLMessage, entry["msg"], "HTML characters should not be escaped")
}

func TestLogger_Levels(t *testing.T) {
	levels := []struct {
		level    logger.LogLevel
		logFunc  func(l logger.Logger, msg string)
		logFuncf func(l logger.Logger, format string, args ...any)
		expected string
	}{
		{logger.TraceLevel, func(l logger.Logger, msg string) { l.Trace(msg) }, func(l logger.Logger, format string, args ...any) { l.Tracef(format, args...) }, "trace"},
		{logger.DebugLevel, func(l logger.Logger, msg string) { l.Debug(msg) }, func(l logger.Logger, format string, args ...any) { l.Debugf(format, args...) }, "debug"},
		{logger.InfoLevel, func(l logger.Logger, msg string) { l.Info(msg) }, func(l logger.Logger, format string, args ...any) { l.Infof(format, args...) }, "info"},
		{logger.WarnLevel, func(l logger.Logger, msg string) { l.Warn(msg) }, func(l logger.Logger, format string, args ...any) { l.Warnf(format, args...) }, "warning"}, // logrus uses "warning" for WarnLevel
		{logger.ErrorLevel, func(l logger.Logger, msg string) { l.Error(msg) }, func(l logger.Logger, format string, args ...any) { l.Errorf(format, args...) }, "error"},
		// Fatal level is skipped as it calls os.Exit(1)
	}

	for _, tc := range levels {
		t.Run(string(tc.level), func(t *testing.T) {
			var buf bytes.Buffer
			log := NewLogger(
				OutputOption(&buf),
				LevelOption(tc.level),
				FormatOption(logger.JSONFormat), // Using JSON for easier parsing
			)

			// Test non-formatted log
			buf.Reset()
			msg := "test " + string(tc.level)
			tc.logFunc(log, msg)

			var entry map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("failed to unmarshal log output for %s: %v. Output: %s", tc.level, err, buf.String())
			}
			assert.Equal(t, tc.expected, entry["level"], "Level string mismatch for %s", tc.level)
			assert.Equal(t, msg, entry["msg"], "Message mismatch for %s", tc.level)
			if tc.level == logger.TraceLevel || tc.level == logger.DebugLevel {
				assert.Contains(t, entry, "caller", "Caller field should exist for %s", tc.level)
			} else {
				assert.NotContains(t, entry, "caller", "Caller field should not exist for %s", tc.level)
			}

			// Test formatted log
			buf.Reset()
			formatMsg := "formatted test %s"
			tc.logFuncf(log, formatMsg, string(tc.level))

			if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
				t.Fatalf("failed to unmarshal formatted log output for %s: %v. Output: %s", tc.level, err, buf.String())
			}
			assert.Equal(t, tc.expected, entry["level"], "Formatted level string mismatch for %s", tc.level)
			assert.Equal(t, "formatted test "+string(tc.level), entry["msg"], "Formatted message mismatch for %s", tc.level)
			if tc.level == logger.TraceLevel || tc.level == logger.DebugLevel {
				assert.Contains(t, entry, "caller", "Caller field should exist for formatted %s", tc.level)
			} else {
				assert.NotContains(t, entry, "caller", "Caller field should not exist for formatted %s", tc.level)
			}
		})
	}
}

func TestLogger_FatalLevel(t *testing.T) {
	// Fatal calls os.Exit(1), so we can't test it directly in a normal test run.
	// This is a common challenge. For robust testing, one might use a setup
	// where `os.Exit` is mocked, or test it in a separate process.
	// Here, we'll just ensure it compiles and can be called.
	// We'll also check if the level is set correctly.
	var buf bytes.Buffer
	log := NewLogger(
		OutputOption(&buf),
		LevelOption(logger.FatalLevel),
		FormatOption(logger.JSONFormat),
	)

	assert.Equal(t, logger.FatalLevel, log.GetLevel(), "Log level should be fatal")
	assert.True(t, log.IsLevelEnabled(logger.FatalLevel), "Fatal level should be enabled")

	// We won't call log.Fatal() or log.Fatalf() here to avoid exiting the test.
	// We assume logrus handles the exit correctly if the level is set.
}

func TestLogger_LevelOption_Invalid(t *testing.T) {
	var buf bytes.Buffer
	log := NewLogger(OutputOption(&buf), LevelOption(logger.LogLevel("invalid")))
	// logrus defaults to InfoLevel for invalid levels
	assert.Equal(t, logger.InfoLevel, log.GetLevel(), "Invalid log level should default to Info")

	log.Debug("should not be visible")
	assert.Empty(t, buf.String(), "Debug log should not be visible with Info level")

	log.Info("should be visible")
	assert.NotEmpty(t, buf.String(), "Info log should be visible")
}

func TestLogger_WithFields(t *testing.T) {
	var buf bytes.Buffer
	baseLogger := NewLogger(
		OutputOption(&buf),
		FormatOption(logger.JSONFormat),
		LevelOption(logger.InfoLevel),
	)

	fieldLogger := baseLogger.WithFields(map[string]any{
		"key1": "value1",
		"key2": 123,
	})

	fieldLogger.Info("message with fields")

	var entry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("failed to unmarshal log output: %v", err)
	}

	assert.Equal(t, "value1", entry["key1"], "Field key1 not found or incorrect")
	assert.Equal(t, float64(123), entry["key2"], "Field key2 not found or incorrect (JSON numbers are float64)") // JSON unmarshals numbers to float64
	assert.Equal(t, "message with fields", entry["msg"], "Message not found")

	// Ensure original logger is not affected
	buf.Reset() // Reset buffer for the base logger
	baseLogger.Info("message without fields")

	// It's important to re-declare `entry` or ensure it's properly reassigned for the new JSON.
	var baseEntry map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &baseEntry); err != nil {
		t.Fatalf("failed to unmarshal log output for base logger: %v. Output: %s", err, buf.String())
	}
	assert.NotContains(t, baseEntry, "key1", "Original logger should not have new fields")
	assert.Equal(t, "message without fields", baseEntry["msg"], "Base logger message mismatch")
}

func TestLogger_CallerField(t *testing.T) {
	var buf bytes.Buffer
	// Set logger level to Debug, to ensure Debug messages are logged.
	log := NewLogger(
		OutputOption(&buf),
		FormatOption(logger.JSONFormat),
		LevelOption(logger.DebugLevel),
	)

	// Log at Debug level - should include caller and be visible
	buf.Reset()
	log.Debug("test caller with debug")
	var entryDebug map[string]interface{}
	// Ensure there's output before trying to unmarshal
	debugOutput := buf.String()
	if debugOutput == "" {
		t.Fatalf("Debug log output was empty when logger level is Debug")
	}
	if err := json.Unmarshal([]byte(debugOutput), &entryDebug); err != nil {
		t.Fatalf("failed to unmarshal debug log output: %v. Output: %s", err, debugOutput)
	}
	callerDebug, okDebug := entryDebug["caller"].(string)
	assert.True(t, okDebug, "Caller field should be a string for Debug message")
	assert.Contains(t, callerDebug, "logger/logger_test.go:", "Caller info for Debug seems incorrect, got: %s", callerDebug)

	// Log at Trace level - should include caller and be visible if logger is TraceLevel
	traceLogger := NewLogger(
		OutputOption(&buf),
		FormatOption(logger.JSONFormat),
		LevelOption(logger.TraceLevel),
	)
	buf.Reset()
	traceLogger.Trace("test caller with trace")
	traceOutput := buf.String()
	if traceOutput == "" {
		t.Fatalf("Trace log output was empty when logger level is Trace")
	}
	var entryTrace map[string]interface{}
	if err := json.Unmarshal([]byte(traceOutput), &entryTrace); err != nil {
		t.Fatalf("failed to unmarshal trace log output: %v. Output: %s", err, traceOutput)
	}
	callerTrace, okTrace := entryTrace["caller"].(string)
	assert.True(t, okTrace, "Caller field should be a string for Trace message")
	assert.Contains(t, callerTrace, "logger/logger_test.go:", "Caller info for Trace seems incorrect, got: %s", callerTrace)


	// Log at Info level (using the 'log' instance which is at DebugLevel) - should NOT include caller, but IS visible
	buf.Reset()
	log.Info("test no caller for info")
	infoOutput := buf.String()
	if infoOutput == "" {
		t.Fatalf("Info log output was empty when logger level is Debug")
	}
	var entryInfo map[string]interface{}
	if err := json.Unmarshal([]byte(infoOutput), &entryInfo); err != nil {
		t.Fatalf("failed to unmarshal info log output: %v. Output: %s", err, infoOutput)
	}
	assert.NotContains(t, entryInfo, "caller", "Caller field should not exist at info level")
}


func TestLogger_IsLevelEnabled(t *testing.T) {
	testCases := []struct {
		name            string
		configLevel     logger.LogLevel
		checkLevel      logger.LogLevel
		shouldBeEnabled bool
	}{
		// Higher severity levels (lower numerical value in logrus) mean more logs are skipped.
		// Lower severity levels (higher numerical value in logrus) mean more logs are shown.
		// Logrus levels: Panic=0, Fatal=1, Error=2, Warn=3, Info=4, Debug=5, Trace=6

		// Logger set to DEBUG: should log DEBUG, INFO, WARN, ERROR, FATAL. Not TRACE (if Trace > Debug).
		// Oh, my previous understanding of logrus levels was off. Trace is the *highest* verbosity.
		// PanicLevel, FatalLevel, ErrorLevel, WarnLevel, InfoLevel, DebugLevel, TraceLevel.
		// So if current level is FOO, it logs FOO and all levels *less verbose* than FOO (i.e. numerically smaller or equal).

		{"ConfigTrace_CheckTrace", logger.TraceLevel, logger.TraceLevel, true},
		{"ConfigTrace_CheckDebug", logger.TraceLevel, logger.DebugLevel, true},
		{"ConfigTrace_CheckInfo", logger.TraceLevel, logger.InfoLevel, true},

		{"ConfigDebug_CheckTrace", logger.DebugLevel, logger.TraceLevel, false}, // Cannot see more verbose
		{"ConfigDebug_CheckDebug", logger.DebugLevel, logger.DebugLevel, true},
		{"ConfigDebug_CheckInfo", logger.DebugLevel, logger.InfoLevel, true},  // Can see less verbose
		{"ConfigDebug_CheckWarn", logger.DebugLevel, logger.WarnLevel, true},
		{"ConfigDebug_CheckError", logger.DebugLevel, logger.ErrorLevel, true},


		{"ConfigInfo_CheckTrace", logger.InfoLevel, logger.TraceLevel, false},
		{"ConfigInfo_CheckDebug", logger.InfoLevel, logger.DebugLevel, false},
		{"ConfigInfo_CheckInfo", logger.InfoLevel, logger.InfoLevel, true},
		{"ConfigInfo_CheckWarn", logger.InfoLevel, logger.WarnLevel, true},
		{"ConfigInfo_CheckError", logger.InfoLevel, logger.ErrorLevel, true},


		{"ConfigWarn_CheckInfo", logger.WarnLevel, logger.InfoLevel, false},
		{"ConfigWarn_CheckWarn", logger.WarnLevel, logger.WarnLevel, true},
		{"ConfigWarn_CheckError", logger.WarnLevel, logger.ErrorLevel, true},

		{"ConfigError_CheckWarn", logger.ErrorLevel, logger.WarnLevel, false},
		{"ConfigError_CheckError", logger.ErrorLevel, logger.ErrorLevel, true},
		{"ConfigError_CheckFatal", logger.ErrorLevel, logger.FatalLevel, true},


		{"ConfigFatal_CheckError", logger.FatalLevel, logger.ErrorLevel, false},
		{"ConfigFatal_CheckFatal", logger.FatalLevel, logger.FatalLevel, true},


		{"InvalidConfigLevel_CheckInfo", logger.LogLevel("invalid"), logger.InfoLevel, true},  // logrus defaults config to Info
		{"ConfigInfo_CheckInvalid", logger.InfoLevel, logger.LogLevel("invalid"), false}, // IsLevelEnabled now returns false for invalid checkLevel
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			log := NewLogger(LevelOption(tc.configLevel))
			actualResult := log.IsLevelEnabled(tc.checkLevel)
			assert.Equal(t, tc.shouldBeEnabled, actualResult, "Mismatch for config level '%s', check level '%s'", tc.configLevel, tc.checkLevel)
		})
	}
}


// --- NopLogger Tests ---

func TestNopLogger(t *testing.T) {
	nopLog := Nop()

	assert.NotNil(t, nopLog, "Nop logger should not be nil")

	// Check all methods to ensure they don't panic
	assert.NotPanics(t, func() {
		nopLog.Trace("trace message")
		nopLog.Tracef("tracef %s", "message")
		nopLog.Debug("debug message")
		nopLog.Debugf("debugf %s", "message")
		nopLog.Info("info message")
		nopLog.Infof("infof %s", "message")
		nopLog.Warn("warn message")
		nopLog.Warnf("warnf %s", "message")
		nopLog.Error("error message")
		nopLog.Errorf("errorf %s", "message")
		// NopLogger's Fatal/Fatalf are also no-ops and don't exit.
		nopLog.Fatal("fatal message")
		nopLog.Fatalf("fatalf %s", "message")
	}, "NopLogger methods should not panic")
}

func TestNopLogger_WithFields(t *testing.T) {
	nopLog := Nop()
	nopLogWithFields := nopLog.WithFields(map[string]any{"key": "value"})

	// Nop logger should return itself for WithFields
	assert.Same(t, nopLog, nopLogWithFields, "WithFields on NopLogger should return the same instance")
}

func TestNopLogger_GetLevel(t *testing.T) {
	nopLog := Nop()
	assert.Equal(t, logger.LogLevel(""), nopLog.GetLevel(), "NopLogger GetLevel should return empty string")
}

func TestNopLogger_IsLevelEnabled(t *testing.T) {
	nopLog := Nop()
	assert.False(t, nopLog.IsLevelEnabled(logger.DebugLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.InfoLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.TraceLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.WarnLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.ErrorLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.FatalLevel), "NopLogger should always have levels disabled")
	assert.False(t, nopLog.IsLevelEnabled(logger.LogLevel("invalid")), "NopLogger should always have levels disabled")
}

func TestLogger_CallerField_PathTrimming(t *testing.T) {
    // This test is to ensure the caller path trimming logic works as expected.
    var buf bytes.Buffer
    log := NewLogger(
        OutputOption(&buf),
        FormatOption(logger.JSONFormat),
        LevelOption(logger.TraceLevel), // Ensure caller is logged
    )

    log.Trace("testing caller path trimming") // Use Trace which should have caller

    var entry map[string]interface{}
    if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
        t.Fatalf("failed to unmarshal log output: %v. Output: %s", err, buf.String())
    }

    caller, ok := entry["caller"].(string)
    assert.True(t, ok, "Caller field should be a string")
    // Example: "logger/logger_test.go:350"
    // We are looking for the pattern "some_dir/file.go:line"
    assert.Regexp(t, `[^/]+/[^/]+\.go:\d+$`, caller, "Caller path format is unexpected: %s", caller)
    // More specific check if we know the last two path components
    assert.True(t, strings.Contains(caller, "logger/logger_test.go:"), "Caller path does not contain expected file path suffix: %s", caller)
}
