package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	rotate "github.com/lestrrat-go/file-rotatelogs"
)

type Option func(*Logger)

type Logger struct {
	path     string
	name     string
	rotation Rotation
	saveDays int
	level    Level
	// 格式控制
	consoleSeparator string
	format           string
	// 格式函数
	EncodeTime    TimeEncoder
	EncodeCaller  CallerEncoder
	EncoderLevel  LevelEncoder
	EncoderCustom []CustomJsonEncoder
	// 基本参数
	writer Writer
	once   sync.Once
}

func New(name string, opts ...Option) *Logger {
	lgr := &Logger{
		name:             name,
		path:             _defaultPath,
		rotation:         RotationDay,
		saveDays:         _defaultSaveDays,
		format:           ConsoleFormat,
		level:            NoneLevel,
		EncodeCaller:     defaultCallEncoder,
		EncodeTime:       defaultTimeEncoder,
		EncoderLevel:     defaultLevelEncoder,
		consoleSeparator: " ",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(lgr)
		}
	}
	return lgr
}

func NewGroup(level Level, opts ...Option) *Logger {
	lgr := &Logger{
		path:             _defaultPath,
		rotation:         RotationDay,
		saveDays:         _defaultSaveDays,
		format:           ConsoleFormat,
		level:            level,
		EncodeCaller:     defaultCallEncoder,
		EncodeTime:       defaultTimeEncoder,
		EncoderLevel:     defaultLevelEncoder,
		consoleSeparator: " ",
	}
	for _, opt := range opts {
		if opt != nil {
			opt(lgr)
		}
	}
	return lgr
}

func (l *Logger) fullName() string {
	if l.path[len(l.path)-1] != '/' {
		l.path += "/"
	}
	if !strings.Contains(l.name, ".log") {
		if l.name[len(l.name)-1] == '.' {
			l.name += "log"
		} else {
			l.name += ".log"
		}
	}
	return l.path + l.name
}

func (l *Logger) levelName(level Level) string {
	if l.path[len(l.path)-1] != '/' {
		l.path += "/"
	}
	return l.path + level.StringLower() + ".log"
}

func (l *Logger) build() {
	l.once.Do(func() {
		if l.level == NoneLevel {
			r, err := rotate.New(
				strings.Replace(l.fullName(), ".log", "", -1)+l.rotation.Format(),
				rotate.WithLinkName(l.fullName()),
				rotate.WithMaxAge(time.Hour*24*time.Duration(l.saveDays)),
				rotate.WithRotationTime(l.rotation.Duration()),
			)
			if err != nil {
				panic(fmt.Sprintf("init logger new roate_log err:%v", err))
			}
			l.writer = &SingleWriter{r: r, level: l.level}
		} else {
			writer := GroupWriter{}
			for lv := DebugLevel; lv <= ErrorLevel; lv++ {
				if l.level == lv || l.level.less(lv) {
					r, err := rotate.New(
						strings.Replace(l.levelName(lv), ".log", "", -1)+l.rotation.Format(),
						rotate.WithLinkName(l.levelName(lv)),
						rotate.WithMaxAge(time.Hour*24*time.Duration(l.saveDays)),
						rotate.WithRotationTime(l.rotation.Duration()),
					)
					if err != nil {
						panic(fmt.Sprintf("init logger new roate_log err:%v", err))
					}
					writer = append(writer, &SingleWriter{r: r, level: lv})
				}
			}
			l.writer = writer
		}
	})
}

// write use setting level
func (l *Logger) Write(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, l.level, format, args...)
}

func (l *Logger) Debug(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, DebugLevel, format, args...)
}

func (l *Logger) Info(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, InfoLevel, format, args...)
}

func (l *Logger) Warn(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, WarnLevel, format, args...)
}

func (l *Logger) Error(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, ErrorLevel, format, args...)
}

func (l *Logger) Panic(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, ErrorLevel, format, args...)
	panic(fmt.Sprintf(format, args...))
}

func (l *Logger) Fatal(ctx context.Context, format string, args ...interface{}) {
	l.write(ctx, ErrorLevel, format, args...)
	os.Exit(-1)
}

func (l *Logger) write(ctx context.Context, level Level, format string, args ...interface{}) {
	l.build()
	w := l.writer.check(level)

	var buf bytes.Buffer
	// 根据格式不同写入
	switch l.format {
	case JsonFormat:
		data := make(map[string]string)
		if d := l.EncodeTime(time.Now()); d != "" {
			data["time"] = d
		}
		if c := l.EncodeCaller(); c != "" {
			data["caller"] = c
		}
		if lv := level.String(); lv != "" {
			data["level"] = lv
		}
		if format == "" {
			data["message"] = fmt.Sprint(args...)
		} else {
			data["message"] = fmt.Sprintf(format, args...)
		}
		if traceID := traceEncoder(ctx); traceID != "" {
			data["trace_id"] = traceID
		}
		bs, err := json.Marshal(data)
		if err != nil {
			return
		}
		for _, ce := range l.EncoderCustom {
			k, v := ce(ctx)
			data[k] = v
		}
		buf.Write(bs)
	case ConsoleFormat:
		if d := l.EncodeTime(time.Now()); d != "" {
			buf.WriteString(d + l.consoleSeparator)
		}
		if c := l.EncodeCaller(); c != "" {
			buf.WriteString(c + l.consoleSeparator)
		}
		if lvl := l.EncoderLevel(level); lvl != "" {
			buf.WriteString(lvl + l.consoleSeparator)
		}
		var msg string
		if format == "" {
			msg = fmt.Sprint(args...)
		} else {
			msg = fmt.Sprintf(format, args...)
		}
		buf.WriteString(msg + l.consoleSeparator)
		if traceID := traceEncoder(ctx); traceID != "" {
			buf.WriteString("(" + traceID + ")")
		}
	}
	if buf.Len() > 0 {
		buf.WriteString("\n")
	}
	w.Write(buf.Bytes())
}

func SetPath(path string) Option {
	return func(logger *Logger) {
		logger.path = path
	}
}

func SetRotation(r Rotation) Option {
	return func(logger *Logger) {
		logger.rotation = r
	}
}

func SetSaveDays(days int) Option {
	return func(logger *Logger) {
		logger.saveDays = days
	}
}

func SetFormat(format string) Option {
	return func(logger *Logger) {
		if format == JsonFormat || format == ConsoleFormat {
			logger.format = format
		}
	}
}

func SetPathDeep(deep int) Option {
	return func(logger *Logger) {
		PathDeep = deep
	}
}

func SetConsoleSeparator(separator string) Option {
	return func(logger *Logger) {
		logger.consoleSeparator = separator
	}
}

func SetTimeEncoder(encoder TimeEncoder) Option {
	return func(logger *Logger) {
		logger.EncodeTime = encoder
	}
}

func SetCallerEncoder(encoder CallerEncoder) Option {
	return func(logger *Logger) {
		logger.EncodeCaller = encoder
	}
}

func SetLevelEncoder(encoder LevelEncoder) Option {
	return func(logger *Logger) {
		logger.EncoderLevel = encoder
	}
}

func SetCustomJsonEncoder(encoders ...CustomJsonEncoder) Option {
	return func(logger *Logger) {
		if logger.format == JsonFormat {
			logger.EncoderCustom = append(logger.EncoderCustom, encoders...)
		}
	}
}
