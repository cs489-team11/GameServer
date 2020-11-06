// File for the generation of random objects, suitable for testing needs
package server

import (
	"fmt"
	"math/rand"
	"time"
)

const letters = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ"
const digits = "0123456789"
const charset = letters + digits

var names = []string{
	"Alice", "Bob", "Michael", "John", "Jennifer", "Jim", "Luke", "Michelle",
	"Nicole", "Adam", "Marshall", "Lucy", "Robin", "Nick", "Jordan",
}

var seededRand *rand.Rand = rand.New(
	rand.NewSource(time.Now().UnixNano()),
)

func RandStringWithCharset(length int, charset string) string {
	// for English letters, we can just use bytes instead
	// of runes.
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func RandString(length int) string {
	return RandStringWithCharset(length, charset)
}

func RandomUserID() string {
	return fmt.Sprintf("user_%s%s", RandStringWithCharset(3, letters), RandStringWithCharset(3, digits))
}

func RandUsername() username {
	return username(names[seededRand.Intn(len(names))])
}

func RandomGameID() string {
	return fmt.Sprintf("game_%s%s", RandStringWithCharset(3, letters), RandStringWithCharset(3, digits))
}

func RandBool() bool {
	return seededRand.Intn(2) == 1
}
