package log

import (
	"fmt"
	"io"
	"os"

	opentracing "github.com/opentracing/opentracing-go"

	"github.com/getsentry/raven-go"
	"github.com/pkg/errors"
	"github.com/uber/jaeger-client-go/config"
	jaegerzap "github.com/uber/jaeger-client-go/log/zap"
	"github.com/uber/jaeger-lib/metrics"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

// Logger including sentry
type Logger struct {
	*zap.Logger
	Sentry *raven.Client
	closer io.Closer
	Tracer opentracing.Tracer
}

// Close the Tracer
func (log *Logger) Close() {
	log.closer.Close()
}

// WithFields wrapper around zap.With
func (log *Logger) WithFields(fields ...zapcore.Field) *Logger {
	log.Logger = log.Logger.With(fields...)
	return log
}

// New Logger including sentry and jaeger
func New(name, dsn string, dbg bool) *Logger {
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	sentry, err := raven.New(dsn)
	if err != nil {
		panic(err)
	}

	consoleDebugging := zapcore.Lock(os.Stdout)
	consoleErrors := zapcore.Lock(os.Stderr)
	consoleConfig := zap.NewDevelopmentEncoderConfig()
	consoleEncoder := zapcore.NewConsoleEncoder(consoleConfig)
	sentryEncoder := NewSentryEncoder(sentry)
	var core zapcore.Core
	if dbg {
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
		)
	} else {
		core = zapcore.NewTee(
			zapcore.NewCore(consoleEncoder, consoleErrors, highPriority),
			zapcore.NewCore(consoleEncoder, consoleDebugging, lowPriority),
			zapcore.NewCore(sentryEncoder, consoleErrors, highPriority),
		)
	}

	logger := zap.New(core)
	if dbg {
		logger = logger.WithOptions(
			zap.AddCaller(),
			zap.AddStacktrace(zap.ErrorLevel),
		)
	} else {
		logger = logger.WithOptions(
			zap.AddStacktrace(zap.FatalLevel),
		)
	}

	cfg := config.Configuration{}

	jMetricsFactory := metrics.NullFactory

	tracer, closer, err := cfg.New(
		name,
		config.Logger(jaegerzap.NewLogger(logger)),
		config.Metrics(jMetricsFactory),
	)
	if err != nil {
		panic(fmt.Sprintf("cannot init jaeger: %v\n", err))
	}
	log := &Logger{
		Logger: logger,
		Sentry: sentry,
		closer: closer,
		Tracer: tracer,
	}

	return log
}

// NewNop returns Logger doing nothing
func NewNop() *Logger {
	sentry, err := raven.New("")
	if err != nil {
		panic(err)
	}

	logger := zap.NewNop()
	cfg := config.Configuration{}

	jMetricsFactory := metrics.NullFactory

	tracer, closer, err := cfg.New(
		"nop",
		config.Logger(jaegerzap.NewLogger(logger)),
		config.Metrics(jMetricsFactory),
	)
	if err != nil {
		panic(fmt.Sprintf("cannot init jaeger: %v\n", err))
	}
	log := &Logger{
		Logger: logger,
		Sentry: sentry,
		closer: closer,
		Tracer: tracer,
	}

	return log
}

// NewSentryEncoder with dsn
func NewSentryEncoder(client *raven.Client) zapcore.Encoder {
	return newSentryEncoder(client)
}

func newSentryEncoder(client *raven.Client) *sentryEncoder {
	enc := &sentryEncoder{}
	enc.Sentry = client
	return enc
}

type sentryEncoder struct {
	zapcore.ObjectEncoder
	dsn    string
	Sentry *raven.Client
}

// Clone .
func (s *sentryEncoder) Clone() zapcore.Encoder {
	return newSentryEncoder(s.Sentry)
}

// EncodeEntry .
func (s *sentryEncoder) EncodeEntry(e zapcore.Entry, fields []zapcore.Field) (*buffer.Buffer, error) {
	buf := buffer.NewPool().Get()
	if e.Level == zapcore.ErrorLevel {
		tags := make(map[string]string)
		var err error
		for _, f := range fields {
			var tag string
			switch f.Type {
			case zapcore.StringType:
				tag = f.String
			case zapcore.Int16Type, zapcore.Int32Type, zapcore.Int64Type:
				tag = fmt.Sprintf("%v", f.Integer)
			case zapcore.ErrorType:
				err = f.Interface.(error)
			}
			tags[f.Key] = tag

		}
		if err == nil {
			s.Sentry.CaptureMessage(e.Message, tags)
			return buf, nil
		}
		s.Sentry.CaptureError(errors.Wrap(err, e.Message), tags)
	}
	return buf, nil
}

func (s *sentryEncoder) AddString(key, val string) {
	tags := s.Sentry.Tags
	if tags == nil {
		tags = make(map[string]string)
	}
	tags[key] = val
	s.Sentry.SetTagsContext(tags)
}
