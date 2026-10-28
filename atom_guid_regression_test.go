package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"testing"
)

func TestConvertedAtomIdentifierRSSGUID(t *testing.T) {
	for _, tc := range []struct{ name, id, href string }{
		{"opaque URN", "urn:uuid:12345678-1234-1234-1234-123456789abc", "https://example.com/article"},
		{"HTTP identifier differs from page", "https://example.com/ids/42", "https://example.com/article"},
		{"HTTP identifier matches page", "https://example.com/article", "https://example.com/article"},
		{"no page", "urn:example:42", ""},
		{"XML escaping", "urn:example:a&b", "https://example.com/article?a=1&b=2"},
		{"missing identifier", "", "https://example.com/article"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			entry := &AtomEntry{Title: "entry", Id: tc.id}
			if tc.href != "" {
				entry.Links = []AtomLink{{Href: tc.href, Rel: "alternate"}}
			}
			before := *entry
			before.Links = append([]AtomLink(nil), entry.Links...)
			item, err := entry.ToItem()
			if err != nil {
				t.Fatal(err)
			}
			expected := ""
			if tc.id != "" {
				expected = "false"
			}
			if item.Id != tc.id || item.IsPermaLink != expected {
				t.Fatalf("converted identifier: id=%q isPermaLink=%q", item.Id, item.IsPermaLink)
			}
			source := &AtomFeed{Title: "feed", Id: "urn:example:feed", Entries: []*AtomEntry{entry}}
			feed, err := source.ToFeed()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(item, feed.Items[0]) {
				t.Fatal("ToFeed differs from ToItem")
			}
			output, err := feed.ToRss()
			if err != nil {
				t.Fatal(err)
			}
			var written bytes.Buffer
			if err := feed.WriteRss(&written); err != nil {
				t.Fatal(err)
			}
			if written.String() != output {
				t.Fatal("writer output differs")
			}
			var rss RssFeedXml
			if err := xml.Unmarshal([]byte(output), &rss); err != nil {
				t.Fatal(err)
			}
			guid := rss.Channel.Items[0].Guid
			if tc.id == "" {
				if guid != nil {
					t.Fatalf("invented GUID: %+v", guid)
				}
			} else if guid == nil || guid.Id != tc.id || guid.IsPermaLink != "false" {
				t.Fatalf("RSS GUID: %+v", guid)
			}
			if rss.Channel.Items[0].Link != tc.href {
				t.Fatalf("article link changed: %q", rss.Channel.Items[0].Link)
			}
			atomOutput, err := feed.ToAtom()
			if err != nil {
				t.Fatal(err)
			}
			var atom AtomFeed
			if err := xml.Unmarshal([]byte(atomOutput), &atom); err != nil {
				t.Fatal(err)
			}
			if tc.id != "" && atom.Entries[0].Id != tc.id {
				t.Fatal("Atom identifier changed")
			}
			jsonOutput, err := feed.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			var jsonFeed JSONFeed
			if err := json.Unmarshal([]byte(jsonOutput), &jsonFeed); err != nil {
				t.Fatal(err)
			}
			if jsonFeed.Items[0].Id != tc.id {
				t.Fatal("JSON identifier changed")
			}
			if !reflect.DeepEqual(*entry, before) {
				t.Fatal("source entry mutated")
			}
		})
	}
}

func TestExplicitRSSPermalinkControlsUnchanged(t *testing.T) {
	for _, value := range []string{"", "true", "false"} {
		source := &RssItem{Guid: &RssGuid{Id: "https://example.com/id", IsPermaLink: value}}
		item, err := source.ToItem()
		if err != nil {
			t.Fatal(err)
		}
		if item.IsPermaLink != value {
			t.Fatalf("RSS flag changed from %q to %q", value, item.IsPermaLink)
		}
		feed := &Feed{Items: []*Item{{Id: "https://example.com/id", IsPermaLink: value}}}
		data, err := feed.ToRss()
		if err != nil {
			t.Fatal(err)
		}
		var rss RssFeedXml
		if err := xml.Unmarshal([]byte(data), &rss); err != nil {
			t.Fatal(err)
		}
		if rss.Channel.Items[0].Guid.IsPermaLink != value {
			t.Fatalf("explicit generic flag %q changed", value)
		}
	}
}
