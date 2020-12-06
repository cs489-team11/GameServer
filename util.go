package server

import (
	"encoding/base64"
)

func decodeB64(message string) string {
	base64Text := make([]byte, base64.StdEncoding.DecodedLen(len(message)))
	l, _ := base64.StdEncoding.Decode(base64Text, []byte(message))
	base64Text = base64Text[:l]
	return string(base64Text)
}

func decodeStringsB64(src []string) []string {
	dest := make([]string, len(src))
	for i, s := range src {
		dest[i] = decodeB64(s)
	}
	return dest
}

// 0 <= index <= len(a)
func insertToSlice(a []string, index int, value string) []string {
	if len(a) == index { // nil or empty slice or after last element
		return append(a, value)
	}
	a = append(a[:index+1], a[index:]...) // index < len(a)
	a[index] = value
	return a
}
