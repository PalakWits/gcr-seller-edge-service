package log

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

const (
	Stdout destinationType = "stdout"
	File   destinationType = "file"
)

const (
	DebugLevel level = "debug"
	InfoLevel  level = "info"
	WarnLevel  level = "warn"
	ErrorLevel level = "error"
	FatalLevel level = "fatal"
	PanicLevel level = "panic"
)

type destinationType string
type level string
type contextKey string

type Config struct {
	Level        level         `json:"level"`
	Destinations []Destination `json:"destinations"`
	ContextKeys  []contextKey  `json:"context_keys,omitempty"`
}

type Destination struct {
	Type   destinationType   `json:"type"`
	Config map[string]string `json:"config,omitempty"`
}

var logLevels = map[level]zerolog.Level{
	DebugLevel: zerolog.DebugLevel,
	InfoLevel:  zerolog.InfoLevel,
	WarnLevel:  zerolog.WarnLevel,
	ErrorLevel: zerolog.ErrorLevel,
	FatalLevel: zerolog.FatalLevel,
	PanicLevel: zerolog.PanicLevel,
}

var (
	logger zerolog.Logger
	cfg    Config
	once   sync.Once
)

var defaultConfig = Config{
	Level: InfoLevel,
	Destinations: []Destination{
		{Type: Stdout},
	},
}

func init() {
	logger, _ = getLogger(defaultConfig)
}

func getLogger(config Config) (zerolog.Logger, error) {
	var newLogger zerolog.Logger
	var writers []io.Writer
	for _, dest := range config.Destinations {
		switch dest.Type {
		case Stdout:
			writers = append(writers, os.Stdout)
		}
	}
	multiwriter := io.MultiWriter(writers...)
	newLogger = zerolog.New(multiwriter).
		Level(logLevels[config.Level]).
		With().
		Timestamp().
		Logger()

	cfg = config
	return newLogger, nil
}

func InitLogger(c Config) error {
	var initErr error
	once.Do(func() {
		logger, initErr = getLogger(c)
	})
	return initErr
}

func Debug(ctx context.Context, msg string) {
	logEvent(ctx, zerolog.DebugLevel, msg, nil)
}

func Debugf(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.DebugLevel, msg, nil)
}

func Info(ctx context.Context, msg string) {
	logEvent(ctx, zerolog.InfoLevel, msg, nil)
}

func Infof(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.InfoLevel, msg, nil)
}

func Warn(ctx context.Context, msg string) {
	logEvent(ctx, zerolog.WarnLevel, msg, nil)
}

func Warnf(ctx context.Context, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.WarnLevel, msg, nil)
}

func Error(ctx context.Context, err error, msg string) {
	logEvent(ctx, zerolog.ErrorLevel, msg, err)
}

func Errorf(ctx context.Context, err error, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.ErrorLevel, msg, err)
}

func Fatal(ctx context.Context, err error, msg string) {
	logEvent(ctx, zerolog.FatalLevel, msg, err)
}

func Fatalf(ctx context.Context, err error, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.FatalLevel, msg, err)
}

func Panic(ctx context.Context, err error, msg string) {
	logEvent(ctx, zerolog.PanicLevel, msg, err)
}

func Panicf(ctx context.Context, err error, format string, v ...any) {
	msg := fmt.Sprintf(format, v...)
	logEvent(ctx, zerolog.PanicLevel, msg, err)
}

func logEvent(ctx context.Context, level zerolog.Level, msg string, err error) {
	event := logger.WithLevel(level)

	if err != nil {
		event = event.Err(err)
	}
	addCtx(ctx, event)
	event.Msg(msg)
}

func addCtx(ctx context.Context, event *zerolog.Event) {
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqIDStr, ok := requestID.(string); ok {
			event.Str("request_id", reqIDStr)
		}
	}

	for _, key := range cfg.ContextKeys {
		val, ok := ctx.Value(key).(string)
		if !ok {
			continue
		}
		keyStr := string(key)
		event.Any(keyStr, val)
	}
}

func RequestStart(ctx context.Context, r *http.Request, body []byte) {
	event := logger.Info()
	addCtx(ctx, event)

	event.Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("path", r.URL.Path).
		Str("query", r.URL.RawQuery).
		Str("user_agent", r.UserAgent()).
		Str("remote_addr", r.RemoteAddr).
		Int("content_length", int(r.ContentLength))

	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			if k != "Authorization" && k != "Cookie" && k != "X-Api-Key" {
				headers[k] = v[0]
			}
		}
	}
	event.Interface("headers", headers)

	if len(body) > 0 && len(body) < 1024 {
		event.Str("body", string(body))
	} else if len(body) > 0 {
		event.Int("body_size", len(body))
	}

	event.Msg("Request started")
}

func RequestEnd(ctx context.Context, r *http.Request, statusCode int, responseTime time.Duration, responseSize int) {
	var event *zerolog.Event
	if statusCode >= 500 {
		event = logger.Error()
	} else if statusCode >= 400 {
		event = logger.Warn()
	} else {
		event = logger.Info()
	}

	addCtx(ctx, event)
	event.Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("path", r.URL.Path).
		Int("status_code", statusCode).
		Dur("response_time", responseTime).
		Int("response_size", responseSize).
		Msg("Request completed")
}

func ErrorWithStack(ctx context.Context, err error, msg string) {
	event := logger.Error()
	addCtx(ctx, event)

	if err != nil {
		event = event.Err(err)
	}

	pc, file, line, ok := runtime.Caller(1)
	if ok {
		fn := runtime.FuncForPC(pc)
		event.Str("file", file).
			Int("line", line).
			Str("function", fn.Name())
	}

	event.Msg(msg)
}

func PanicLog(ctx context.Context, r *http.Request, panicValue interface{}) {
	event := logger.Error()
	addCtx(ctx, event)

	event.Interface("panic", panicValue).
		Str("method", r.Method).
		Str("url", r.URL.String()).
		Str("path", r.URL.Path).
		Str("remote_addr", r.RemoteAddr)

	buf := make([]byte, 1024*16)
	n := runtime.Stack(buf, false)
	event.Str("stack_trace", string(buf[:n]))

	event.Msg("Panic recovered")
}
