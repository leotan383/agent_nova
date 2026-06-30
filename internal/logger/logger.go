package logger

import (
	"fmt"
	"os"
	"strings"
)

var debugEnabled bool

func SetDebug(on bool) { debugEnabled = on }

func Debug(format string, args ...any) {
	if !debugEnabled {
		return
	}
	fmt.Fprintf(os.Stderr, "[debug] "+format+"\n", args...)
}

func Info(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
}

func Truncate(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
