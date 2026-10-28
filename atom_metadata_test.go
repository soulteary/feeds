package feeds

import (
	"bytes"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAtomPublishedDates(t *testing.T) {
	created := time.Date(2026, 10, 20, 9, 30, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	updated := created.Add(48 * time.Hour)
	for _, tc := range []struct {
		name                       string
		created, updated           time.Time
		wantPublished, wantUpdated string
	}{
		{"created only", created, time.Time{}, created.Format(time.RFC3339), created.Format(time.RFC3339)},
		{"distinct created and updated", created, updated, created.Format(time.RFC3339), updated.Format(time.RFC3339)},
		{"updated only", time.Time{}, updated, "", updated.Format(time.RFC3339)},
		{"neither", time.Time{}, time.Time{}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := &Item{Id: "urn:item:stable", Title: "post", Created: tc.created, Updated: tc.updated}
			before := *item
			f := &Feed{Id: "urn:feed:stable", Items: []*Item{item}}
			atom := (&Atom{f}).AtomFeed()
			if atom.Entries[0].Published != tc.wantPublished || atom.Entries[0].Updated != tc.wantUpdated {
				t.Fatalf("published=%q updated=%q; want %q / %q", atom.Entries[0].Published, atom.Entries[0].Updated, tc.wantPublished, tc.wantUpdated)
			}
			output := atomXMLForTest(t, f)
			var decoded AtomFeed
			if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatal(err)
			}
			entry := decoded.Entries[0]
			if entry.Published != tc.wantPublished || entry.Updated != tc.wantUpdated || entry.Id != item.Id {
				t.Fatalf("metadata changed during serialization: %+v", entry)
			}
			if tc.wantPublished == "" && strings.Contains(output, "<published>") {
				t.Fatalf("unknown publication date must be omitted: %s", output)
			}
			if !reflect.DeepEqual(before, *item) {
				t.Fatal("source item mutated")
			}
		})
	}
}

func TestAtomFeedIDPrecedence(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		link     *Link
		want     string
	}{
		{"explicit ID and link", "urn:feed:stable", &Link{Href: "https://example.com/"}, "urn:feed:stable"},
		{"explicit ID without link", "urn:feed:stable", nil, "urn:feed:stable"},
		{"explicit ID with empty link", "urn:feed:stable", &Link{}, "urn:feed:stable"},
		{"link fallback", "", &Link{Href: "https://example.com/"}, "https://example.com/"},
		{"neither", "", nil, ""},
		{"empty link", "", &Link{}, ""},
		{"XML characters", "https://example.com/?a=1&b=2", nil, "https://example.com/?a=1&b=2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Feed{Id: tc.id, Link: tc.link, Items: []*Item{{Id: "urn:item:unchanged"}}}
			before := *f
			var linkBefore Link
			if tc.link != nil {
				linkBefore = *tc.link
			}
			atom := (&Atom{f}).AtomFeed()
			if atom.Id != tc.want {
				t.Fatalf("id=%q; want %q", atom.Id, tc.want)
			}
			output := atomXMLForTest(t, f)
			var decoded AtomFeed
			if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Id != tc.want || decoded.Entries[0].Id != "urn:item:unchanged" {
				t.Fatalf("serialized IDs changed: %s", output)
			}
			wantHref := ""
			if tc.link != nil {
				wantHref = tc.link.Href
			}
			if decoded.Link.Href != wantHref {
				t.Fatalf("ID replaced link: %+v", decoded.Link)
			}
			if !reflect.DeepEqual(before, *f) || (tc.link != nil && !reflect.DeepEqual(linkBefore, *tc.link)) {
				t.Fatal("source feed/link mutated")
			}
		})
	}
}

func TestAtomMetadataLeavesOtherFormatsUnchanged(t *testing.T) {
	created := time.Date(2026, 10, 20, 9, 30, 0, 0, time.UTC)
	item := &Item{Id: "urn:item:stable", Created: created, Updated: created.Add(time.Hour)}
	f := &Feed{Link: &Link{Href: "https://example.com/"}, Items: []*Item{item}}
	rssBefore, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	jsonBefore, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	f.Id = "urn:feed:explicit"
	atomXMLForTest(t, f)
	rssAfter, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	jsonAfter, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if rssBefore != rssAfter || jsonBefore != jsonAfter {
		t.Fatal("Atom metadata altered RSS/JSON output")
	}
	// Low-level Atom callers still control their own dates and IDs.
	direct := &AtomFeed{Id: "urn:direct", Entries: []*AtomEntry{{Id: "urn:entry", Published: "custom-date"}}}
	output, err := ToXML(direct)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "<id>urn:direct</id>") || !strings.Contains(output, "<published>custom-date</published>") {
		t.Fatal(output)
	}
}

func atomXMLForTest(t *testing.T, f *Feed) string {
	t.Helper()
	s, err := f.ToAtom()
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := f.WriteAtom(&b); err != nil {
		t.Fatal(err)
	}
	if s != b.String() {
		t.Fatal("ToAtom and WriteAtom disagree")
	}
	return s
}
