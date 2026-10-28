package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestFeedMultipleLinksByFormat(t *testing.T) {
	home := &Link{Href: "https://example.com/?a=1&b=2"}
	atomSelf := &Link{Href: "https://example.com/atom.xml", Rel: "self", Type: "application/atom+xml"}
	rssSelf := &Link{Href: "https://example.com/rss.xml", Rel: "self", Type: "application/rss+xml"}
	jsonSelf := &Link{Href: "https://example.com/feed.json", Rel: "self", Type: "application/feed+json; charset=utf-8"}
	hub := &Link{Href: "https://example.com/hub", Rel: "hub"}
	f := &Feed{Id: "urn:stable", Link: home, Links: []*Link{nil, {}, home, {Href: home.Href, Rel: "alternate"}, atomSelf, rssSelf, jsonSelf, hub, atomSelf}}
	before, _ := json.Marshal(f)
	output := atomXMLForTest(t, f)
	var atom AtomFeed
	if err := xml.Unmarshal([]byte(output), &atom); err != nil {
		t.Fatal(err)
	}
	all := append([]AtomLink{*atom.Link}, atom.Links...)
	if len(all) != 3 || all[0].Href != home.Href || all[1].Href != atomSelf.Href || all[1].Rel != "self" || all[1].Type != atomSelf.Type || all[2].Rel != "hub" || atom.Id != "urn:stable" {
		t.Fatalf("unexpected Atom links: %+v", all)
	}
	rss, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := f.WriteRss(&buf); err != nil {
		t.Fatal(err)
	}
	if rss != buf.String() {
		t.Fatal("RSS writer disagrees")
	}
	var decoded RssFeedXml
	if err := xml.Unmarshal([]byte(rss), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Channel.Link != home.Href || len(decoded.Channel.AtomLinks) != 2 {
		t.Fatalf("RSS homepage/links: %s", rss)
	}
	for i, want := range []string{rssSelf.Href, hub.Href} {
		link := decoded.Channel.AtomLinks[i]
		if link.XMLName.Space != ns || link.Href != want {
			t.Fatalf("wrong namespace or URL: %+v", link)
		}
	}
	j, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	buf.Reset()
	if err := f.WriteJSON(&buf); err != nil {
		t.Fatal(err)
	}
	var jf, writer JSONFeed
	if err := json.Unmarshal([]byte(j), &jf); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(buf.Bytes(), &writer); err != nil {
		t.Fatal(err)
	}
	if jf.HomePageUrl != home.Href || jf.FeedUrl != jsonSelf.Href || !reflect.DeepEqual(jf, writer) {
		t.Fatalf("JSON URLs: %s", j)
	}
	after, _ := json.Marshal(f)
	if !bytes.Equal(before, after) {
		t.Fatal("source links mutated")
	}
}

func TestLinkSelectionAndEmptyCollections(t *testing.T) {
	for _, tc := range []struct {
		name       string
		primary    *Link
		links      []*Link
		home, self string
	}{
		{"legacy homepage", &Link{Href: "home"}, nil, "home", ""},
		{"empty slice", &Link{Href: "home"}, []*Link{}, "home", ""},
		{"nil and empty", nil, []*Link{nil, {}}, "", ""},
		{"additional homepage", nil, []*Link{{Href: "home", Rel: "alternate", Type: "text/html"}}, "home", ""},
		{"legacy homepage wins", &Link{Href: "old"}, []*Link{{Href: "new", Rel: "alternate"}}, "old", ""},
		{"explicit self is not homepage", &Link{Href: "subscription", Rel: "self"}, nil, "", "subscription"},
		{"primary hub is not homepage", &Link{Href: "hub", Rel: "hub"}, nil, "", ""},
		{"typed self wins", nil, []*Link{{Href: "untyped", Rel: "self"}, {Href: "specific", Rel: "self", Type: "application/atom+xml"}}, "", "specific"},
		{"other format self ignored", nil, []*Link{{Href: "rss", Rel: "self", Type: "application/rss+xml"}}, "", ""},
		{"feed alternate not homepage", nil, []*Link{{Href: "rss", Rel: "alternate", Type: "application/rss+xml"}}, "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Feed{Link: tc.primary, Links: tc.links}
			home := homeLink(f.Link, f.Links)
			gotHome := ""
			if home != nil {
				gotHome = home.Href
			}
			if gotHome != tc.home {
				t.Fatalf("home=%q want %q", gotHome, tc.home)
			}
			self := selfLink(f.Link, f.Links, "application/atom+xml")
			gotSelf := ""
			if self != nil {
				gotSelf = self.Href
			}
			if gotSelf != tc.self {
				t.Fatalf("self=%q want %q", gotSelf, tc.self)
			}
			output := atomXMLForTest(t, f)
			if tc.self == "" && strings.Contains(output, `rel="self"`) {
				t.Fatalf("invented self: %s", output)
			}
			if _, err := f.ToRss(); err != nil {
				t.Fatal(err)
			}
			if _, err := f.ToJSON(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestAtomMultipleLinkRoundtrip(t *testing.T) {
	input := `<feed xmlns="http://www.w3.org/2005/Atom"><title>test</title><id>urn:stable</id><updated>2026-01-01T00:00:00Z</updated><link href="https://example.com/"/><link href="https://example.com/feed" rel="self" type="application/atom+xml" length="42"/><link href="https://example.com/hub" rel="hub"/></feed>`
	var f AtomFeed
	if err := xml.Unmarshal([]byte(input), &f); err != nil {
		t.Fatal(err)
	}
	if f.Link == nil || f.Link.Href != "https://example.com/" || len(f.Links) != 2 || f.Links[0].Length != "42" {
		t.Fatalf("lost links: %+v", f)
	}
	before := append([]AtomLink(nil), f.Links...)
	for _, value := range []interface{}{f, &f} {
		b, err := xml.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var again AtomFeed
		if err := xml.Unmarshal(b, &again); err != nil {
			t.Fatalf("invalid XML %s: %v", b, err)
		}
		if !reflect.DeepEqual(f.Link, again.Link) || !reflect.DeepEqual(f.Links, again.Links) {
			t.Fatalf("roundtrip lost links: %s", b)
		}
	}
	if !reflect.DeepEqual(before, f.Links) {
		t.Fatal("marshal mutated additional links")
	}
	if err := xml.Unmarshal([]byte(`<feed><id>replacement</id></feed>`), &f); err != nil {
		t.Fatal(err)
	}
	if f.Link == nil || len(f.Links) != 2 || f.Title != "test" {
		t.Fatal("missing fields should preserve existing values")
	}
	if err := xml.Unmarshal([]byte(`<feed><link href="new"/></feed>`), &f); err != nil {
		t.Fatal(err)
	}
	if f.Link.Href != "new" || len(f.Links) != 0 {
		t.Fatal("explicit link replacement retained old additional links")
	}
	direct := AtomFeed{Links: []AtomLink{{Href: "one", Rel: "self"}, {Href: "two", Rel: "hub"}}}
	b, err := xml.Marshal(direct)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "<link ") != 2 {
		t.Fatalf("direct additional links lost: %s", b)
	}
}

func TestItemAdditionalLinksAndEnclosure(t *testing.T) {
	enclosure := &Link{Href: "https://example.com/audio.mp3", Rel: "enclosure", Type: "audio/mpeg", Length: "42"}
	item := &Item{Id: "urn:item", Link: &Link{Href: "https://example.com/post"}, Links: []*Link{nil, {}, enclosure, enclosure, {Href: "https://example.com/edit", Rel: "edit"}}, Enclosure: &Enclosure{Url: enclosure.Href, Type: enclosure.Type, Length: enclosure.Length}}
	entry := newAtomEntry(item)
	if len(entry.Links) != 3 || entry.Links[1].Length != "42" || entry.Links[2].Rel != "edit" {
		t.Fatalf("links or enclosure lost/duplicated: %+v", entry.Links)
	}
	if newRssItem(item).Link != item.Link.Href || newJSONItem(item).Url != item.Link.Href {
		t.Fatal("enclosure chosen as permalink")
	}
	item.Link = nil
	item.Links = []*Link{enclosure, {Href: "https://example.com/post", Rel: "alternate", Type: "text/html"}}
	item.Id = ""
	item.Created = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	entry = newAtomEntry(item)
	if newRssItem(item).Link != "https://example.com/post" || newJSONItem(item).Url != "https://example.com/post" || entry.Id != "tag:example.com,2026-01-01:/post" {
		t.Fatalf("additional homepage not used: %+v", entry)
	}
}

func TestRSSPrefixedAtomLinksDoNotReplaceHomePage(t *testing.T) {
	input := `<rss xmlns:atom="http://www.w3.org/2005/Atom"><channel><link>https://example.com/</link><atom:link href="https://example.com/feed" rel="self" type="application/rss+xml"/><atom:link href="https://example.com/hub" rel="hub"/></channel></rss>`
	var decoded RssFeedXml
	if err := xml.Unmarshal([]byte(input), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Channel.Link != "https://example.com/" || len(decoded.Channel.AtomLinks) != 2 || decoded.Channel.AtomLinks[0].Rel != "self" {
		t.Fatalf("namespaced links shadowed homepage: %+v", decoded.Channel)
	}
	output, err := ToXML(decoded.Channel)
	if err != nil {
		t.Fatal(err)
	}
	var again RssFeedXml
	if err := xml.Unmarshal([]byte(output), &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Channel, again.Channel) {
		t.Fatalf("RSS namespaced link roundtrip changed fields: %s", output)
	}
}

func TestAdditionalLinksSupplyFeedIDAndDefaultSelfType(t *testing.T) {
	for _, tc := range []struct {
		name   string
		links  []*Link
		wantID string
	}{
		{"homepage before self", []*Link{{Href: "https://example.com/feed", Rel: "self"}, {Href: "https://example.com/", Rel: "alternate"}}, "https://example.com/"},
		{"self only", []*Link{{Href: "https://example.com/feed", Rel: "self"}}, "https://example.com/feed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Feed{Links: tc.links}
			a := (&Atom{f}).AtomFeed()
			if a.Id != tc.wantID {
				t.Fatalf("id=%q want %q", a.Id, tc.wantID)
			}
			links := append([]AtomLink{*a.Link}, a.Links...)
			found := false
			for _, link := range links {
				if link.Rel == "self" {
					found = true
					if link.Type != "application/atom+xml" {
						t.Fatalf("self type=%q", link.Type)
					}
				}
			}
			if !found {
				t.Fatal("missing self")
			}
			rss := (&Rss{f}).RssFeed()
			if len(rss.AtomLinks) != 1 || rss.AtomLinks[0].Type != "application/rss+xml" {
				t.Fatalf("RSS self type: %+v", rss.AtomLinks)
			}
		})
	}
}
