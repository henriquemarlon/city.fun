// The config package manages the node configuration, which comes from environment variables.
// The sub-package generate specifies these environment variables.
package configs

import (
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Redacted is a wrapper that redacts a given field from the logs.
type Redacted[T any] struct {
	Value T
}

func (r Redacted[T]) String() string {
	return "[REDACTED]"
}

type (
	URL            = *url.URL
	Duration       = time.Duration
	LogLevel       = slog.Level
	RedactedString = Redacted[string]
	RedactedUint   = Redacted[uint32]
)

// ------------------------------------------------------------------------------------------------
// Parsing functions
// ------------------------------------------------------------------------------------------------

func ToUint64FromString(s string) (uint64, error) {
	value, err := strconv.ParseUint(s, 10, 64)
	return value, err
}

func ToUint64FromDecimalOrHexString(s string) (uint64, error) {
	if len(s) >= 2 && (strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X")) {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	return ToUint64FromString(s)
}

func ToStringFromString(s string) (string, error) {
	return s, nil
}

func ToDurationFromSeconds(s string) (time.Duration, error) {
	return time.ParseDuration(s + "s")
}

func ToLogLevelFromString(s string) (LogLevel, error) {
	var m = map[string]LogLevel{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
	}
	if v, ok := m[s]; ok {
		return v, nil
	} else {
		var zeroValue LogLevel
		return zeroValue, fmt.Errorf("invalid log level '%s'", s)
	}
}

func ToRedactedStringFromString(s string) (RedactedString, error) {
	return RedactedString{s}, nil
}

func ToRedactedUint32FromString(s string) (RedactedUint, error) {
	value, err := strconv.ParseUint(s, 10, 32)
	return RedactedUint{uint32(value)}, err
}

func ToURLFromString(s string) (URL, error) {
	result, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("invalid URL [Redacted]")
	}
	return result, nil
}

// Aliases to be used by the generated functions.
var (
	toBool           = strconv.ParseBool
	toString         = ToStringFromString
	toDuration       = ToDurationFromSeconds
	toLogLevel       = ToLogLevelFromString
	toRedactedString = ToRedactedStringFromString
	toURL            = ToURLFromString
)

var (
	notDefinedbool           = func() bool { return false }
	notDefinedstring         = func() string { return "" }
	notDefinedDuration       = func() time.Duration { return 0 }
	notDefinedLogLevel       = func() slog.Level { return slog.LevelInfo }
	notDefinedRedactedString = func() RedactedString { return RedactedString{""} }
	notDefinedURL            = func() URL { return &url.URL{} }
)
