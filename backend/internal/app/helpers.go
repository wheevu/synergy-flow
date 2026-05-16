package app

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
)

// rank returns the numeric rank of a workspace role for permission comparison.
func rank(r string) int {
	return map[string]int{"Viewer": 1, "Member": 2, "Admin": 3, "Owner": 4}[r]
}

// slugify transforms a string into a URL-safe slug (lowercase alphanumeric + hyphens).
func slugify(s string) string {
	return strings.Trim(strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r + 32
		}
		return '-'
	}, s), "-")
}

// hashToken returns the SHA-256 hex digest of a token string.
func hashToken(t string) string {
	h := sha256.Sum256([]byte(t))
	return hex.EncodeToString(h[:])
}

// randString generates a cryptographically random alphanumeric string of the given length.
func randString(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// strp returns a string pointer from a map, or nil if the key is missing or not a string.
func strp(m map[string]any, k string) *string {
	if v, ok := m[k].(string); ok {
		return &v
	}
	return nil
}

// stringSlice converts a []any to []string, or returns nil.
func stringSlice(v any) []string {
	if a, ok := v.([]any); ok {
		out := []string{}
		for _, x := range a {
			out = append(out, fmt.Sprint(x))
		}
		return out
	}
	return nil
}
