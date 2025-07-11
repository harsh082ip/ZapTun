package log

import (
	"io"
	"os"

	"github.com/rs/zerolog"
)

const (
	loggerKey = "log_data"
)

var (
	DefaultLoggerAddr *zerolog.Logger
)

type Logger struct {
	Log           *zerolog.Logger
	DefaultLogger *DefaultLog
}

type DefaultLog struct {
	RemoteAddr string `json:"remote_addr,omitempty"`
	Source     string `json:"source,omitempty"`
}

// NewLogger creates and returns a new configured Logger instance.
// This single function replaces the previous InitLogger and the logic from utils.go.
func NewLogger(writer io.Writer, level zerolog.Level, source string) *Logger {
	// If no writer is provided, default to standard output.
	if writer == nil {
		writer = os.Stdout
	}

	zerolog.SetGlobalLevel(level)
	zLogger := zerolog.New(writer).With().Timestamp().Str("source", source).Logger()

	defaultLog := &DefaultLog{Source: source}
	return &Logger{Log: &zLogger, DefaultLogger: defaultLog}
}
func (l Logger) LogInfoMessage() *zerolog.Event {
	return l.Log.Info().Interface(loggerKey, l.DefaultLogger)
}
func (l Logger) LogWarnMessage() *zerolog.Event {
	return l.Log.Warn().Interface(loggerKey, l.DefaultLogger)
}
func (l Logger) LogDebugMessage() *zerolog.Event {
	return l.Log.Debug().Interface(loggerKey, l.DefaultLogger)
}
func (l Logger) LogErrorMessage() *zerolog.Event {
	return l.Log.Error().Interface(loggerKey, l.DefaultLogger)
}
func (l Logger) LogFatalMessage() *zerolog.Event {
	return l.Log.Fatal().Interface(loggerKey, l.DefaultLogger)
}
func (l *Logger) Print(val string) {
	l.Log.Info().Interface(loggerKey, l.DefaultLogger).Msg(val)
}
func (l *Logger) Printf(format string, val string) {
	l.Log.Info().Interface(loggerKey, l.DefaultLogger).Msg(val)
}
func (l *Logger) Println(val string) {
	l.Log.Info().Interface(loggerKey, l.DefaultLogger).Msg(val)
}
