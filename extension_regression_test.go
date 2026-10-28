package feeds

import (
	"encoding/xml"
	"reflect"
	"testing"
)

// Uses existing APIs to expose the old foreign/local-name collision.
func TestForeignXMLTitleDoesNotOverrideNativeTitle(t *testing.T) {
	for _, value := range []interface{}{&AtomFeed{}, &RssFeed{}} {
		root := "feed"
		if _, ok := value.(*RssFeed); ok {
			root = "channel"
		}
		input := "<" + root + ` xmlns:x="urn:extension"><title>native title</title><x:title>foreign title</x:title></` + root + ">"
		if err := xml.Unmarshal([]byte(input), value); err != nil {
			t.Fatal(err)
		}
		if reflect.ValueOf(value).Elem().FieldByName("Title").String() != "native title" {
			t.Fatalf("foreign title replaced native field: %s", input)
		}
	}
}
