package logger

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"path/filepath"
)

var (
	infoLogger    = log.New(os.Stdout, "", 0)
	errorLogger   = log.New(os.Stderr, "", 0)
	warningLogger = log.New(os.Stdout, "", 0)
	successLogger = log.New(os.Stdout, "", 0)
	debugLogger   = log.New(os.Stdout, "", 0)
	
	verbose = false
	
	// ANSI color codes
	useColors = true
	reset     = "\033[0m"
	red       = "\033[31m"
	green     = "\033[32m"
	yellow    = "\033[33m"
	blue      = "\033[34m"
	magenta   = "\033[35m"
	cyan      = "\033[36m"
)

func init() {
	// Disable colors on Windows command prompt (cmd.exe)
	// but leave them enabled for PowerShell, WSL, etc.
	if runtime.GOOS == "windows" && os.Getenv("TERM") == "" && os.Getenv("WT_SESSION") == "" {
		useColors = false
	}
}

func SetVerbose(v bool) {
	verbose = v
}

func colored(color, format string, args ...interface{}) string {
	message := fmt.Sprintf(format, args...)
	
	if useColors {
		return color + message + reset
	}
	
	return message
}

func Info(format string, args ...interface{}) {
	message := colored(blue, "ℹ️ "+format, args...)
	infoLogger.Println(message)
}

func Warning(format string, args ...interface{}) {
	message := colored(yellow, "⚠️ "+format, args...)
	warningLogger.Println(message)
}

func Error(format string, args ...interface{}) {
	message := colored(red, "❌ "+format, args...)
	errorLogger.Println(message)
}

func Success(format string, args ...interface{}) {
	message := colored(green, "✅ "+format, args...)
	successLogger.Println(message)
}

func Debug(format string, args ...interface{}) {
	if !verbose {
		return
	}
	
	message := colored(magenta, "🔍 "+format, args...)
	debugLogger.Println(message)
}

func Fatal(format string, args ...interface{}) {
	message := colored(red, "💥 FATAL: "+format, args...)
	errorLogger.Println(message)
	os.Exit(1)
}

func RepoHeader(repoPath string) {
	displayPath := repoPath
	cwd, err := os.Getwd()
	if err == nil {
		if rel, err := filepath.Rel(cwd, repoPath); err == nil && !strings.HasPrefix(rel, "..") {
			displayPath = rel
		}
	}
	
	fmt.Println()
	message := colored(cyan, "📁 %s", displayPath)
	infoLogger.Println(message)
}
