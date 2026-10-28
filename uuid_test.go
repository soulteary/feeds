package feeds

import (
	"encoding/hex"
	"regexp"
	"strings"
	"testing"
)

func TestUUID(t *testing.T) {
	canonicalV4 := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	seen := make(map[UUID]struct{})
	var previous *UUID
	// Check a sample of values; uniqueness here is a regression check, not proof
	// of randomness or a guarantee that arbitrary UUIDs never collide.
	for i := 0; i < 128; i++ {
		u := NewUUID()
		if u == nil {
			t.Fatal("NewUUID returned nil")
		}
		if len(u) != 16 {
			t.Fatalf("UUID length=%d; want 16", len(u))
		}
		if got := u[6] >> 4; got != 4 {
			t.Fatalf("sample %d: version=%x; want 4 (byte 6=%02x)", i, got, u[6])
		}
		if got := u[8] & 0xc0; got != 0x80 {
			t.Fatalf("sample %d: variant bits=%02x; want 80 (byte 8=%02x)", i, got, u[8])
		}
		text := u.String()
		if len(text) != 36 || !canonicalV4.MatchString(text) {
			t.Fatalf("sample %d: invalid canonical v4 string %q", i, text)
		}
		raw, err := hex.DecodeString(strings.ReplaceAll(text, "-", ""))
		if err != nil {
			t.Fatal(err)
		}
		var decoded UUID
		copy(decoded[:], raw)
		if decoded != *u {
			t.Fatalf("sample %d: string does not represent UUID bytes: %q vs %x", i, text, u[:])
		}
		if previous != nil && *u == *previous {
			t.Fatalf("samples %d and %d have equal UUID values: %s", i-1, i, text)
		}
		if _, duplicate := seen[*u]; duplicate {
			t.Fatalf("sample %d repeats UUID value %s", i, text)
		}
		seen[*u] = struct{}{}
		previous = u
	}
}

func TestUUIDString(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value UUID
		want  string
	}{
		{"zero", UUID{}, "00000000-0000-0000-0000-000000000000"},
		{"known v4", UUID{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x46, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}, "00112233-4455-4677-8899-aabbccddeeff"},
		{"arbitrary bytes", UUID{0xff, 0xee, 0xdd, 0xcc, 0xbb, 0xaa, 0x99, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x00}, "ffeeddcc-bbaa-9988-7766-554433221100"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.value
			if got := tc.value.String(); got != tc.want {
				t.Fatalf("String()=%q; want %q", got, tc.want)
			}
			if tc.value != before {
				t.Fatal("String mutated UUID bytes")
			}
		})
	}
}
