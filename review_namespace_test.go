package feeds

import (
	"encoding/xml"
	"io"
	"strings"
	"testing"
)

func TestRSSContentPrefixBinding(t *testing.T) {
	for _, bindings := range []string{`xmlns:content="urn:other" xmlns:c="http://purl.org/rss/1.0/modules/content/"`, `xmlns:c="http://purl.org/rss/1.0/modules/content/"`} {
		input := `<rss version="2.0" ` + bindings + `><channel><title>test</title><link>https://example.com</link><description>test</description><item><title>entry</title><c:encoded><![CDATA[<p>body & text</p>]]></c:encoded></item></channel></rss>`
		var root RssFeedXml
		if err := xml.Unmarshal([]byte(input), &root); err != nil {
			t.Fatal(err)
		}
		for name, value := range map[string]interface{}{"root": root.FeedXml(), "channel": root.Channel, "item": root.Channel.Items[0], "content": root.Channel.Items[0].Content} {
			t.Run(name+bindings, func(t *testing.T) {
				data, err := xml.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				decoder := xml.NewDecoder(strings.NewReader(string(data)))
				found := false
				for {
					token, err := decoder.Token()
					if err == io.EOF {
						break
					}
					if err != nil {
						t.Fatal(err)
					}
					if start, ok := token.(xml.StartElement); ok && start.Name.Local == "encoded" {
						found = true
						if start.Name.Space != rssContentNamespace {
							t.Fatalf("wrong content namespace: %q", start.Name.Space)
						}
					}
				}
				if !found {
					t.Fatalf("content missing: %s", data)
				}
			})
		}
		output, err := ToXML(&root)
		if err != nil {
			t.Fatal(err)
		}
		var again RssFeedXml
		if err := xml.Unmarshal([]byte(output), &again); err != nil {
			t.Fatal(err)
		}
		if got := again.Channel.Items[0].Content; got == nil || got.Content != "<p>body & text</p>" {
			t.Fatalf("body lost: %#v", got)
		}
		if root.ContentNamespace != "" && root.ContentNamespace != "urn:other" {
			t.Fatal("input mutated")
		}
	}
}
