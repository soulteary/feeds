package feeds

import (
	"encoding/xml"
	"strings"
	"testing"
)

// Uses pre-existing raw-extension APIs to check Apple's root declaration rule.
func TestRawITunesExtensionsDeclareRootNamespace(t *testing.T) {
	const namespace = "http://www.itunes.com/dtds/podcast-1.0.dtd"
	raw := Extension{XMLName: xml.Name{Space: namespace, Local: "author"}, Content: []ExtensionContent{{Text: "Host"}}}
	for _, f := range []*Feed{{Extensions: []Extension{raw}}, {Items: []*Item{{Extensions: []Extension{raw}}}}} {
		output, err := f.ToRss()
		if err != nil {
			t.Fatal(err)
		}
		decoder := xml.NewDecoder(strings.NewReader(output))
		found := false
		for {
			token, err := decoder.Token()
			if err != nil {
				t.Fatal(err)
			}
			if root, ok := token.(xml.StartElement); ok {
				for _, attr := range root.Attr {
					if attr.Name.Space == "xmlns" && attr.Name.Local == "itunes" && attr.Value == namespace {
						found = true
					}
				}
				break
			}
		}
		if !found {
			t.Fatalf("missing root xmlns:itunes declaration: %s", output)
		}
	}
}
