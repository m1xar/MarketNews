package ingest

import (
	"encoding/json"
	"strconv"
	"strings"
)

func NormalizeField(s string) string {
	return strings.TrimSpace(s)
}

func NormalizeName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	return strings.Join(strings.Fields(s), " ")
}

type newsMeta struct {
	SoloURL         string `json:"soloUrl"`
	TrimmedPrefixed string `json:"trimmedPrefixedName"`
	PrefixedName    string `json:"prefixedName"`
}

func DeriveNewsKey(meta string) *string {
	if meta == "" {
		return nil
	}
	var m newsMeta
	if err := json.Unmarshal([]byte(meta), &m); err != nil {
		return nil
	}
	key := strings.TrimSpace(m.SoloURL)
	if key == "" {
		key = strings.TrimSpace(m.TrimmedPrefixed)
	}
	if key == "" {
		key = strings.TrimSpace(m.PrefixedName)
	}
	if key == "" {
		return nil
	}
	key = strings.ToLower(key)
	return &key
}

func ParseValueFloat(s string) *float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	lower := strings.ToLower(s)
	if lower == "n/a" || lower == "na" || lower == "none" {
		return nil
	}

	multiplier := 1.0
	last := s[len(s)-1]
	switch last {
	case 'K', 'k':
		multiplier = 1e3
		s = s[:len(s)-1]
	case 'M', 'm':
		multiplier = 1e6
		s = s[:len(s)-1]
	case 'B', 'b':
		multiplier = 1e9
		s = s[:len(s)-1]
	case 'T', 't':
		multiplier = 1e12
		s = s[:len(s)-1]
	}

	clean := strings.Builder{}
	for _, r := range s {
		if (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '+' {
			clean.WriteRune(r)
		}
	}
	if clean.Len() == 0 {
		return nil
	}
	v, err := strconv.ParseFloat(clean.String(), 64)
	if err != nil {
		return nil
	}
	v = v * multiplier
	return &v
}
