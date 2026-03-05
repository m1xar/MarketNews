package repository

import "strconv"

func itoa(v int) string {
	return strconv.Itoa(v)
}

func normalizeLimit(v int) int {
	const (
		defaultLimit = 100
		maxLimit     = 1000
	)
	if v <= 0 {
		return defaultLimit
	}
	if v > maxLimit {
		return maxLimit
	}
	return v
}
