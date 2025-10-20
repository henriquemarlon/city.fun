// The config package manages the node configuration, which comes from environment variables.
// The sub-package generate specifies these environment variables.
package configs

import (
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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
	Address        = common.Address
)

// ------------------------------------------------------------------------------------------------
// Auth Kind
// ------------------------------------------------------------------------------------------------

type AuthKind uint8

const (
	AuthKindPrivateKeyVar AuthKind = iota
	AuthKindPrivateKeyFile
	AuthKindMnemonicVar
	AuthKindMnemonicFile
	AuthKindAWS
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

func ToAddressFromString(s string) (Address, error) {
	if len(s) < 3 || (!strings.HasPrefix(s, "0x") && !strings.HasPrefix(s, "0X")) {
		return Address{}, fmt.Errorf("invalid address '%s'", s)
	}
	s = s[2:]
	b, err := hex.DecodeString(s)
	if err != nil {
		return Address{}, err
	}
	return common.BytesToAddress(b), nil
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

func ToSliceStringFromString(s string) ([]string, error) {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		result = append(result, part)
	}
	return result, nil
}

func ToAuthKindFromString(s string) (AuthKind, error) {
	var m = map[string]AuthKind{
		"private_key":      AuthKindPrivateKeyVar,
		"private_key_file": AuthKindPrivateKeyFile,
		"mnemonic":         AuthKindMnemonicVar,
		"mnemonic_file":    AuthKindMnemonicFile,
		"aws":              AuthKindAWS,
	}
	if v, ok := m[s]; ok {
		return v, nil
	} else {
		var zeroValue AuthKind
		return zeroValue, fmt.Errorf("invalid auth kind '%s'", s)
	}
}

// Aliases to be used by the generated functions.
var (
	toBool           = strconv.ParseBool
	toUint64         = ToUint64FromString
	toString         = ToStringFromString
	toDuration       = ToDurationFromSeconds
	toLogLevel       = ToLogLevelFromString
	toRedactedUint   = ToRedactedUint32FromString
	toRedactedString = ToRedactedStringFromString
	toURL            = ToURLFromString
	toSliceString    = ToSliceStringFromString
	toAddress        = ToAddressFromString
	toAuthKind       = ToAuthKindFromString
)

var (
	notDefinedBool           = func() bool { return false }
	notDefinedUint64         = func() uint64 { return 0 }
	notDefinedString         = func() string { return "" }
	notDefinedDuration       = func() time.Duration { return 0 }
	notDefinedLogLevel       = func() slog.Level { return slog.LevelInfo }
	notDefinedRedactedString = func() RedactedString { return RedactedString{""} }
	notDefinedRedactedUint   = func() RedactedUint { return RedactedUint{0} }
	notDefinedURL            = func() URL { return &url.URL{} }
	notDefinedSliceString    = func() []string { return []string{} }
	notDefinedAddress        = func() Address { return common.Address{} }
	notDefinedAuthKind       = func() AuthKind { return AuthKindPrivateKeyVar }
)
