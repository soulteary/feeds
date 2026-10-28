package feeds

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
)

func FuzzExtensionNameCompatibility(f *testing.F) {
	for _, name := range []string{"", "value", "_name", "a-1.2", "1name", "-name", ".name", "a?", "a\x00", "a:b", "中文", "é", "1中文", "a\u0301", "😀", "a\xff"} {
		f.Add(name)
	}
	f.Fuzz(func(t *testing.T, name string) {
		want := false
		if name != "" && !strings.ContainsAny(name, ": \t\r\n<>/&=\"'") {
			_, err := xml.NewDecoder(strings.NewReader("<" + name + "/>")).Token()
			want = err == nil
		}
		if got := validExtensionName(xml.Name{Local: name}); got != want {
			t.Fatalf("name %q: got %v want %v", name, got, want)
		}
	})
}

func FuzzExtensionTextCompatibility(f *testing.F) {
	for _, text := range []string{"", "plain", "<&>\"'", "a\nb", "a\tb\rc", "中文 café", "&#xA;", "\x00\xff"} {
		f.Add(text)
	}
	f.Fuzz(func(t *testing.T, text string) {
		var got, want bytes.Buffer
		if err := encodeExtensionContent(&got, []ExtensionContent{{Text: text}}); err != nil {
			t.Fatal(err)
		}
		encoder := xml.NewEncoder(&want)
		if err := encoder.EncodeToken(xml.CharData(text)); err != nil {
			t.Fatal(err)
		}
		if err := encoder.Flush(); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got.Bytes(), want.Bytes()) {
			t.Fatalf("text %q: got %q want %q", text, got.String(), want.String())
		}
	})
}

type shortExtensionWriter struct{ calls int }

func (w *shortExtensionWriter) Write(p []byte) (int, error) { w.calls++; return 0, nil }
func TestExtensionTextWriterCompatibility(t *testing.T) {
	w := &shortExtensionWriter{}
	if err := encodeExtensionContent(w, []ExtensionContent{{Text: "text"}}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("short write: %v", err)
	}
	w.calls = 0
	if err := encodeExtensionContent(w, []ExtensionContent{{}}); err != nil || w.calls != 0 {
		t.Fatalf("empty text changed writer behavior: err=%v calls=%d", err, w.calls)
	}
}
