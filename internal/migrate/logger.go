package migrate

import "log"

type Logger struct {
	IsVerbose bool
}

func NewLogger(verbose bool) *Logger {
	return &Logger{IsVerbose: verbose}
}

func (l *Logger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *Logger) Verbose() bool {
	return l.IsVerbose
}
