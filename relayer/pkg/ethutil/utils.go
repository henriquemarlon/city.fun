package ethutil

import (
	"strings"
)

func TrimHex(s string) string {
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	s = strings.TrimSpace(s)
	return s
}