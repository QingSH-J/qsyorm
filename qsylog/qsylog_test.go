package qsylog

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

// 自定义的Write接口实现，用于测试
type testWriter struct {
	buffer *bytes.Buffer
}

// 实现Write接口的Printf方法
func (tw *testWriter) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
	// 直接写入到buffer而不是从log.Writer()
	fmt.Fprintf(tw.buffer, format, v...)
}

// 测试不同日志级别
func TestLogLevel(t *testing.T) {
	tests := []struct {
		name        string
		level       Loglevel
		logFunc     func(Interface)
		expected    string
		notExpected string
	}{
		{
			name:     "Silent level - no logs",
			level:    Silent,
			logFunc:  func(l Interface) { l.Info("info message"); l.Warn("warn message"); l.Error("error message") },
			expected: "",
		},
		{
			name:        "Error level - only error logs",
			level:       Error,
			logFunc:     func(l Interface) { l.Info("info message"); l.Warn("warn message"); l.Error("error message") },
			expected:    "error message",
			notExpected: "info message",
		},
		{
			name:        "Warning level - warning and error logs",
			level:       Warning,
			logFunc:     func(l Interface) { l.Info("info message"); l.Warn("warn message"); l.Error("error message") },
			expected:    "warn message",
			notExpected: "info message",
		},
		{
			name:     "Info level - all logs",
			level:    Info,
			logFunc:  func(l Interface) { l.Info("info message"); l.Warn("warn message"); l.Error("error message") },
			expected: "info message",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer

			// 创建测试用的writer
			writer := &testWriter{buffer: &buffer}

			// 创建日志实例
			logger := New(writer, Config{
				Colorful: false,
				Loglevel: test.level,
			})

			// 执行日志操作
			test.logFunc(logger)

			output := buffer.String()
			if test.expected != "" && !strings.Contains(output, test.expected) {
				t.Errorf("Expected output to contain %q, but got %q", test.expected, output)
			}

			if test.notExpected != "" && strings.Contains(output, test.notExpected) {
				t.Errorf("Expected output to NOT contain %q, but it did", test.notExpected)
			}
		})
	}
}

// 测试带有参数的日志输出
func TestLogWithParameters(t *testing.T) {
	var buffer bytes.Buffer
	writer := &testWriter{buffer: &buffer}

	logger := New(writer, Config{
		Colorful: false,
		Loglevel: Info,
	})

	tests := []struct {
		name     string
		logFunc  func()
		expected string
	}{
		{
			name:     "Info with parameters",
			logFunc:  func() { logger.Info("info with %d parameters: %s", 2, "test") },
			expected: "info with 2 parameters: test",
		},
		{
			name:     "Warn with parameters",
			logFunc:  func() { logger.Warn("warn with parameter: %s", "warning") },
			expected: "warn with parameter: warning",
		},
		{
			name:     "Error with parameters",
			logFunc:  func() { logger.Error("error code: %d", 500) },
			expected: "error code: 500",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			buffer.Reset()
			test.logFunc()
			output := buffer.String()
			if !strings.Contains(output, test.expected) {
				t.Errorf("Expected output to contain %q, but got %q", test.expected, output)
			}
		})
	}
}

// 测试彩色输出
func TestColorfulOutput(t *testing.T) {
	var buffer bytes.Buffer
	writer := &testWriter{buffer: &buffer}

	colorfulLogger := New(writer, Config{
		Colorful: true,
		Loglevel: Info,
	})

	nonColorfulLogger := New(writer, Config{
		Colorful: false,
		Loglevel: Info,
	})

	// 测试带颜色的日志
	t.Run("Colorful logs", func(t *testing.T) {
		buffer.Reset()
		colorfulLogger.Error("test error")
		output := buffer.String()
		if !strings.Contains(output, Red) {
			t.Errorf("Expected colorful output to contain color code, but got %q", output)
		}
	})

	// 测试不带颜色的日志
	t.Run("Non-colorful logs", func(t *testing.T) {
		buffer.Reset()
		nonColorfulLogger.Error("test error")
		output := buffer.String()
		if strings.Contains(output, Red) {
			t.Errorf("Expected non-colorful output to not contain color code, but got %q", output)
		}
	})
}

// 测试Discard日志实例
func TestDiscardLogger(t *testing.T) {
	// 保存原始输出并在测试后恢复
	originalOutput := log.Writer()
	defer log.SetOutput(originalOutput)

	var buffer bytes.Buffer
	log.SetOutput(&buffer)

	// 使用Discard日志记录器
	Discard.Info("This should be discarded")
	Discard.Warn("This should be discarded")
	Discard.Error("This should be discarded")

	output := buffer.String()
	if strings.Contains(output, "should be discarded") {
		t.Errorf("Expected Discard logger to not output anything, but got: %s", output)
	}
}

// 测试Default日志实例
func TestDefaultLogger(t *testing.T) {
	var buffer bytes.Buffer
	writer := &testWriter{buffer: &buffer}

	// 创建一个临时自定义Default日志实例用于测试
	originalDefault := Default
	defer func() { Default = originalDefault }()

	Default = New(writer, Config{
		Colorful: true,
		Loglevel: Warning,
	})

	// 测试Warning级别应该能输出
	buffer.Reset()
	Default.Warn("default warning")
	if !strings.Contains(buffer.String(), "default warning") {
		t.Errorf("Expected Default logger to output warning, but got: %s", buffer.String())
	}

	// 测试Info级别不应该输出
	buffer.Reset()
	Default.Info("default info")
	if strings.Contains(buffer.String(), "default info") {
		t.Errorf("Expected Default logger to not output info, but got: %s", buffer.String())
	}

	// 测试Error级别应该能输出
	buffer.Reset()
	Default.Error("default error")
	if !strings.Contains(buffer.String(), "default error") {
		t.Errorf("Expected Default logger to output error, but got: %s", buffer.String())
	}
}
