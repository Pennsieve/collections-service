package dbmigrate

import "log"

// Logger implements migrate.Logger; if we don't pass migrate.Migrate one of
// these, it won't do its internal logging.
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
