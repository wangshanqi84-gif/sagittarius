package logger

import (
	"fmt"
	"time"
)

type Level int8

const (
	NoneLevel Level = iota - 1
	DebugLevel
	InfoLevel
	WarnLevel
	ErrorLevel
)

func (l Level) String() string {
	switch l {
	case NoneLevel:
		return ""
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

func (l Level) StringLower() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

func (l Level) isNoneLevel() bool {
	return l == NoneLevel
}

func (l Level) less(dst Level) bool {
	if l < dst {
		return true
	}
	return false
}

const (
	RotationDay  Rotation = "day"
	RotationHour Rotation = "hour"
)

type Rotation string

func (r Rotation) Duration() time.Duration {
	switch r {
	case RotationDay:
		return time.Hour * 24
	case RotationHour:
		return time.Hour
	}
	return time.Hour * 24
}

func (r Rotation) Format() string {
	switch r {
	case RotationDay:
		return "-%Y%m%d.log"
	case RotationHour:
		return "-%Y%m%d%H.log"
	}
	return "-%Y%m%d.log"
}
