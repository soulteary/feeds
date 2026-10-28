package feeds

import (
	"encoding/xml"
	"testing"
)

// Uses existing low-level APIs to expose loss of incoming xml:base attributes.
func TestXMLBaseInputSurvivesRoundtrip(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		value       interface{}
	}{
		{"Atom feed", `<feed xml:base="https://example.com/"/>`, &AtomFeed{}},
		{"Atom entry", `<entry xml:base="../post/"/>`, &AtomEntry{}},
		{"RSS channel", `<channel xml:base="https://example.com/"/>`, &RssFeed{}},
		{"RSS item", `<item xml:base="../post/"/>`, &RssItem{}},
		{"RSS root", `<rss version="2.0" xml:base="https://example.com/"/>`, &RssFeedXml{ContentNamespace: "http://purl.org/rss/1.0/modules/content/"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := xml.Unmarshal([]byte(tc.input), tc.value); err != nil {
				t.Fatal(err)
			}
			output, err := xml.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			var original, decoded struct {
				Base string `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
			}
			if err := xml.Unmarshal([]byte(tc.input), &original); err != nil {
				t.Fatal(err)
			}
			if err := xml.Unmarshal(output, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Base != original.Base {
				t.Fatalf("xml:base lost: input %s, output %s", tc.input, output)
			}
		})
	}
}
