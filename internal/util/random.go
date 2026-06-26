package util

import (
	"math/rand"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

// RandomInt generates a random integer between min and max (inclusive).
func RandomInt(min, max int64) int64 {
	return min + rand.Int63n(max-min+1)
}

// RandomString generates a random string of length n.
func RandomString(n int) string {
	var sb strings.Builder
	k := len(alphabet)

	for i := 0; i < n; i++ {
		c := alphabet[rand.Intn(k)]
		sb.WriteByte(c)
	}

	return sb.String()
}

// RandomEmail generates a random email address.
func RandomEmail() string {
	return RandomString(int(RandomInt(3, 8))) + "@" + RandomString(5) + ".com"
}
