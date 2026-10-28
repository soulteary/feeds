package feeds

import (
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAtomToFeed(t *testing.T) {
	date := "2024-01-02T03:04:05.123+08:00"
	ext := Extension{XMLName: xml.Name{Space: "urn:example", Local: "data"}, Attrs: []xml.Attr{{Name: xml.Name{Local: "id"}, Value: "original"}}, Content: []ExtensionContent{{Element: &Extension{XMLName: xml.Name{Local: "nested"}, Content: []ExtensionContent{{Text: "original"}}}}}}
	a := &AtomFeed{Title: "show", Id: "urn:show", Subtitle: "about", Updated: date, Language: "zh-CN", XMLBase: "https://example.com/", Rights: "copyright", Logo: "logo.png", Icon: "icon.png", Author: &AtomAuthor{AtomPerson: AtomPerson{Name: "Alice", Email: "alice@example.com"}}, Link: &AtomLink{Href: "feed.atom", Rel: "self", Type: "application/atom+xml"}, Links: []AtomLink{{Href: "index.html", Type: "text/html"}, {Href: "hub", Rel: "hub"}}, Category: "one", Categories: []AtomCategory{{Term: "two", Scheme: "urn:scheme"}}, Extensions: []Extension{ext}, Entries: []*AtomEntry{nil, {Title: "episode", Id: "urn:episode", Updated: date, Published: date, XMLBase: "episodes/", Language: "en", Author: &AtomAuthor{AtomPerson: AtomPerson{Name: "Bob"}}, Summary: &AtomSummary{Type: "html", Content: "<p>summary</p>"}, Content: &AtomContent{Type: "html", Content: "<p>body</p>"}, Category: "entry", Categories: []AtomCategory{{Term: "other"}}, Links: []AtomLink{{Href: "episode.html"}, {Href: "audio.mp3", Rel: "enclosure", Type: "audio/mpeg", Length: "42"}}, Extensions: []Extension{ext}}}}
	before, _ := xml.Marshal(a)
	feed, err := a.ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if feed.Title != "show" || feed.Id != "urn:show" || feed.Description != "about" || feed.Subtitle != "about" || feed.Copyright != "copyright" || feed.Image.Url != "logo.png" || feed.Author.Email != "alice@example.com" || feed.Link.Href != "index.html" || feed.XMLBase != a.XMLBase || feed.Language != "zh-CN" || len(feed.Links) != 2 || !reflect.DeepEqual(feed.Categories, []string{"one", "two"}) {
		t.Fatalf("feed mapping: %+v", feed)
	}
	item := feed.Items[0]
	if len(feed.Items) != 1 || item.Title != "episode" || item.Id != "urn:episode" || item.Content != "<p>body</p>" || item.Description != "<p>summary</p>" || item.Enclosure.Url != "audio.mp3" || item.Link.Href != "episode.html" || item.Author.Name != "Bob" || item.XMLBase != "episodes/" || item.Language != "en" || len(item.Categories) != 2 {
		t.Fatalf("item mapping: %+v", item)
	}
	want, _ := time.Parse(time.RFC3339Nano, date)
	if !feed.Updated.Equal(want) || !item.Created.Equal(want) || !item.Updated.Equal(want) {
		t.Fatal("dates changed")
	}
	after, _ := xml.Marshal(a)
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
	feed.Extensions[0].Attrs[0].Value = "changed"
	feed.Extensions[0].Content[0].Element.Content[0].Text = "changed"
	item.Links[0].Href = "changed"
	item.Author.Name = "changed"
	if a.Extensions[0].Attrs[0].Value != "original" || a.Extensions[0].Content[0].Element.Content[0].Text != "original" || a.Entries[1].Links[1].Href != "audio.mp3" || a.Entries[1].Author.Name != "Bob" {
		t.Fatal("result aliases source")
	}
	for _, format := range []func() (string, error){feed.ToAtom, feed.ToRss, feed.ToJSON} {
		if _, err := format(); err != nil {
			t.Fatal(err)
		}
	}
	for _, value := range []*AtomFeed{{Icon: "icon.png"}, {}, {Link: &AtomLink{Href: "self", Rel: "self"}}, {Links: []AtomLink{{Href: "home"}, {Href: "home"}}}} {
		if _, err := value.ToFeed(); err != nil {
			t.Fatal(err)
		}
	}
	icon, _ := (&AtomFeed{Icon: "icon.png"}).ToFeed()
	if icon.Image.Url != "icon.png" {
		t.Fatal("icon fallback")
	}
}

func TestRSSToFeed(t *testing.T) {
	r := &RssFeedXml{XMLBase: "https://example.com/root/", Channel: &RssFeed{XMLBase: "../show/", Title: "show", Link: "index.html", Description: "about", Language: "en", Copyright: "copyright", ManagingEditor: "alice@example.com (Alice)", PubDate: "Tue, 02 Jan 2024 03:04:05 +0000", LastBuildDate: "Tue, 02 Jan 2024 04:04:05 GMT", Category: "one", Categories: []string{"two"}, AtomLinks: []RssAtomLink{{Href: "feed.rss", Rel: "self", Type: "application/rss+xml"}}, Image: &RssImage{Url: "image.png", Title: "image", Link: "index.html", Width: 100, Height: 80}, ITunes: &ITunesFeed{Author: "podcaster"}, Items: []*RssItem{nil, {Title: "episode", Link: "episode.html", Description: "summary", XMLBase: "episodes/", Content: &RssContent{Content: "body"}, Author: "Bob", Category: "item", Categories: []string{"other"}, Guid: &RssGuid{Id: "id", IsPermaLink: "false"}, PubDate: "02 Jan 24 03:04 +0000", Enclosure: &RssEnclosure{Url: "audio", Type: "audio/mpeg", Length: "42"}, Source: "legacy", SourceInfo: &RssSource{URL: "source.rss", Name: "Source"}, ITunes: &ITunesItem{Episode: 2}}}}}
	before, _ := xml.Marshal(r)
	f, err := r.ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if f.XMLBase != "https://example.com/show/" || f.Title != "show" || f.Link.Href != "index.html" || f.Description != "about" || f.Author.Name != "Alice" || f.Author.Email != "alice@example.com" || f.Language != "en" || f.Copyright != "copyright" || f.Image.Width != 100 || f.Image.Height != 80 || f.Image.Title != "image" || len(f.Categories) != 2 || len(f.Links) != 1 || len(f.Extensions) != 1 || f.Created.IsZero() || f.Updated.IsZero() {
		t.Fatalf("feed mapping: %+v", f)
	}
	i := f.Items[0]
	if len(f.Items) != 1 || i.Source.Href != "source.rss" || i.Title != "episode" || i.Link.Href != "episode.html" || i.Description != "summary" || i.Content != "body" || i.XMLBase != "episodes/" || i.Author.Name != "Bob" || i.Id != "id" || i.IsPermaLink != "false" || i.Enclosure.Length != "42" || len(i.Categories) != 2 || len(i.Extensions) != 1 || i.Created.IsZero() || !i.Updated.IsZero() {
		t.Fatalf("item mapping: %+v", i)
	}
	after, _ := xml.Marshal(r)
	if string(before) != string(after) {
		t.Fatal("source mutated")
	}
	f.Image.Url = "changed"
	f.Links[0].Href = "changed"
	i.Enclosure.Url = "changed"
	if r.Channel.Image.Url != "image.png" || r.Channel.AtomLinks[0].Href != "feed.rss" || r.Channel.Items[1].Enclosure.Url != "audio" {
		t.Fatal("alias")
	}
	for _, format := range []func() (string, error){f.ToAtom, f.ToRss, f.ToJSON} {
		if _, err := format(); err != nil {
			t.Fatal(err)
		}
	}
	legacy, _ := (&RssItem{Source: "legacy"}).ToItem()
	if legacy.Source.Href != "legacy" {
		t.Fatal("legacy source")
	}
	bare, _ := (&RssFeedXml{XMLBase: "relative/", Channel: &RssFeed{}}).ToFeed()
	if bare.XMLBase != "relative/" {
		t.Fatal("root base")
	}
	if _, err := (&RssFeedXml{Channel: &RssFeed{}}).ToFeed(); err != nil {
		t.Fatal(err)
	}
	if _, err := (&RssItem{PubDate: "02 Jan 24 03:04 GMT"}).ToItem(); err != nil {
		t.Fatal(err)
	}
}

func TestConversionErrors(t *testing.T) {
	cases := []struct {
		name string
		run  func() error
	}{
		{"nil Atom", func() error { _, err := (*AtomFeed)(nil).ToFeed(); return err }},
		{"nil Atom entry", func() error { _, err := (*AtomEntry)(nil).ToItem(); return err }},
		{"nil RSS", func() error { _, err := (*RssFeed)(nil).ToFeed(); return err }},
		{"nil RSS item", func() error { _, err := (*RssItem)(nil).ToItem(); return err }},
		{"nil document", func() error { _, err := (*RssFeedXml)(nil).ToFeed(); return err }},
		{"no channel", func() error { _, err := (&RssFeedXml{}).ToFeed(); return err }},
		{"Atom updated", func() error { _, err := (&AtomFeed{Updated: "bad"}).ToFeed(); return err }},
		{"Atom entry updated", func() error { _, err := (&AtomFeed{Entries: []*AtomEntry{{Updated: "bad"}}}).ToFeed(); return err }},
		{"Atom published", func() error { _, err := (&AtomEntry{Published: "bad"}).ToItem(); return err }},
		{"RSS pubDate", func() error { _, err := (&RssFeed{PubDate: "bad"}).ToFeed(); return err }},
		{"RSS lastBuildDate", func() error { _, err := (&RssFeed{LastBuildDate: "bad"}).ToFeed(); return err }},
		{"RSS item", func() error { _, err := (&RssFeed{Items: []*RssItem{{PubDate: "bad"}}}).ToFeed(); return err }},
		{"root base", func() error { _, err := (&RssFeedXml{XMLBase: "%", Channel: &RssFeed{}}).ToFeed(); return err }},
		{"channel base", func() error {
			_, err := (&RssFeedXml{XMLBase: "https://example.com", Channel: &RssFeed{XMLBase: "%"}}).ToFeed()
			return err
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := c.run(); err == nil || !strings.Contains(err.Error(), "feeds:") {
				t.Fatalf("missing contextual error: %v", err)
			}
		})
	}
}

func TestConversionDecodedFixtures(t *testing.T) {
	var atom AtomFeed
	if err := xml.Unmarshal([]byte(`<feed xmlns="http://www.w3.org/2005/Atom"><title>test</title><id>id</id><updated>2024-01-02T03:04:05Z</updated><link href="https://example.com"/><entry><title>entry</title><id>item</id><updated>2024-01-02T03:04:05Z</updated><content type="html">&lt;p&gt;body&lt;/p&gt;</content></entry></feed>`), &atom); err != nil {
		t.Fatal(err)
	}
	f, err := atom.ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if f.Items[0].Content != "<p>body</p>" {
		t.Fatal("body lost")
	}
	rss, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	var root RssFeedXml
	if err := xml.Unmarshal([]byte(rss), &root); err != nil {
		t.Fatal(err)
	}
	back, err := root.ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if back.Items[0].Content != f.Items[0].Content || back.Items[0].Id != "item" || back.Title != "test" {
		t.Fatalf("conversion lost shared fields: %+v", back)
	}
}

func TestConversionTextAndTimeSemantics(t *testing.T) {
	for _, kind := range []string{"", "text", "text/plain", "html", "text/html"} {
		entry := &AtomEntry{Content: &AtomContent{Type: kind, Content: "<a>&"}, Summary: &AtomSummary{Type: kind, Content: "<a>&"}}
		item, err := entry.ToItem()
		if err != nil {
			t.Fatal(err)
		}
		want := "<a>&"
		if kind != "html" && kind != "text/html" {
			want = "&lt;a&gt;&amp;"
		}
		if item.Content != want || item.Description != want {
			t.Fatalf("type %s: %+v", kind, item)
		}
	}
	for _, entry := range []*AtomEntry{{Content: &AtomContent{Type: "xhtml"}}, {Summary: &AtomSummary{Type: "application/xml"}}} {
		if _, err := entry.ToItem(); err == nil {
			t.Fatal("unsupported content accepted")
		}
	}
	f, err := (&AtomFeed{Author: &AtomAuthor{AtomPerson: AtomPerson{Name: "Alice"}}, Entries: []*AtomEntry{{}}}).ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if f.Items[0].Author.Name != "Alice" {
		t.Fatal("author inheritance lost")
	}
	f.Items[0].Author.Name = "Bob"
	if f.Author.Name != "Alice" {
		t.Fatal("inherited author aliases feed")
	}
	if _, err := (&AtomFeed{Language: "en", Entries: []*AtomEntry{{LanguageSet: true}}}).ToFeed(); err == nil {
		t.Fatal("unrepresentable reset accepted")
	}
	for zone, offset := range map[string]int{"UT": 0, "UTC": 0, "GMT": 0, "EST": -5, "EDT": -4, "CST": -6, "CDT": -5, "MST": -7, "MDT": -6, "PST": -8, "PDT": -7} {
		item, err := (&RssItem{PubDate: "Tue, 02 Jan 2024 03:04:05 " + zone}).ToItem()
		if err != nil {
			t.Fatal(err)
		}
		_, got := item.Created.Zone()
		if got != offset*3600 {
			t.Fatalf("zone %s offset %d", zone, got)
		}
	}
	if _, err := (&RssItem{PubDate: "Tue, 02 Jan 2024 03:04:05 XYZ"}).ToItem(); err == nil {
		t.Fatal("unknown zone silently mapped to UTC")
	}
}

func TestRelativeRSSBaseNeedsDocumentContext(t *testing.T) {
	if _, err := (&RssFeedXml{XMLBase: "relative/", Channel: &RssFeed{XMLBase: "nested/"}}).ToFeed(); err == nil {
		t.Fatal("ambiguous relative base flattening accepted")
	}
	f, err := (&RssFeedXml{XMLBase: "relative/", Channel: &RssFeed{XMLBase: "https://example.com/channel/"}}).ToFeed()
	if err != nil {
		t.Fatal(err)
	}
	if f.XMLBase != "https://example.com/channel/" {
		t.Fatal(f.XMLBase)
	}
}

func TestConversionWhitespaceDate(t *testing.T) {
	if _, err := (&RssItem{PubDate: " "}).ToItem(); err == nil {
		t.Fatal("whitespace date accepted")
	}
}
