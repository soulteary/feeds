package feeds

// relevant bits from https://github.com/abneptis/GoUUID/blob/master/uuid.go

import (
	"crypto/rand"
	"fmt"
)

// UUID is a 16-byte identifier. NewUUID generates random version 4 values.
type UUID [16]byte

// NewUUID returns a random version 4 UUID using crypto/rand.
// It panics if the random source fails.
func NewUUID() *UUID {
	u := &UUID{}
	_, err := rand.Read(u[:16])
	if err != nil {
		panic(err)
	}

	u[8] = (u[8] | 0x80) & 0xBf
	u[6] = (u[6] | 0x40) & 0x4f
	return u
}

// String returns the UUID bytes as lowercase hexadecimal in 8-4-4-4-12 groups.
// It formats the stored bytes without validating their version or variant.
func (u *UUID) String() string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", u[:4], u[4:6], u[6:8], u[8:10], u[10:])
}
