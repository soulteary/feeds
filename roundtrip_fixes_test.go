package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestAtomEntryLinksRoundtrip(t *testing.T) {
	input := `<feed xmlns="http://www.w3.org/2005/Atom"><entry><id>urn:item</id><link href="post" rel="alternate"/><link href="audio" rel="enclosure" type="audio/mpeg" length="42"/></entry></feed>`
	var f AtomFeed
	if err := xml.Unmarshal([]byte(input), &f); err != nil {
		t.Fatal(err)
	}
	if len(f.Entries[0].Links) != 2 || f.Entries[0].Links[1].Length != "42" {
		t.Fatalf("links lost: %+v", f.Entries[0])
	}
	output, err := ToXML(&f)
	if err != nil {
		t.Fatal(err)
	}
	var again AtomFeed
	if err := xml.Unmarshal([]byte(output), &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(f.Entries[0].Links, again.Entries[0].Links) {
		t.Fatalf("roundtrip links differ: %s", output)
	}
	entry := f.Entries[0]
	if err := xml.Unmarshal([]byte(`<entry><title>update</title></entry>`), entry); err != nil {
		t.Fatal(err)
	}
	if len(entry.Links) != 2 {
		t.Fatal("missing links cleared old values")
	}
	if err := xml.Unmarshal([]byte(`<entry><link href="new"/></entry>`), entry); err != nil {
		t.Fatal(err)
	}
	if len(entry.Links) != 1 || entry.Links[0].Href != "new" {
		t.Fatal("explicit links did not replace old links")
	}
}

func TestRSSContentAndSourceRoundtrip(t *testing.T) {
	input := `<rss xmlns:c="http://purl.org/rss/1.0/modules/content/"><channel><item><c:encoded><![CDATA[<p>body & more</p>]]></c:encoded><source url="https://example.com/feed?a=1&amp;b=2">Source name</source></item></channel></rss>`
	var r RssFeedXml
	if err := xml.Unmarshal([]byte(input), &r); err != nil {
		t.Fatal(err)
	}
	i := r.Channel.Items[0]
	if i.Content == nil || i.Content.Content != "<p>body & more</p>" || len(i.Extensions) != 0 {
		t.Fatalf("typed content missing or duplicated: %+v", i)
	}
	if i.SourceInfo == nil || i.SourceInfo.URL != "https://example.com/feed?a=1&b=2" || i.Source != "Source name" {
		t.Fatalf("source lost: %+v", i)
	}
	// A stale raw content extension must not duplicate the typed content.
	i.Extensions = []Extension{{XMLName: xml.Name{Space: rssContentNamespace, Local: "encoded"}, Content: []ExtensionContent{{Text: "stale"}}}}
	output, err := ToXML(r.Channel)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(output, "<content:encoded") != 1 || strings.Contains(output, "stale") {
		t.Fatalf("content emitted twice: %s", output)
	}
	var again RssFeedXml
	if err := xml.Unmarshal([]byte(output), &again); err != nil {
		t.Fatal(err)
	}
	if again.Channel.Items[0].Content.Content != i.Content.Content || !reflect.DeepEqual(i.SourceInfo, again.Channel.Items[0].SourceInfo) {
		t.Fatal("RSS data changed on roundtrip")
	}
	if len(i.Extensions) != 1 {
		t.Fatal("encoding mutated source extensions")
	}
	legacy := &Feed{Items: []*Item{{Source: &Link{Href: "https://example.com/source"}}}}
	output, err = legacy.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, `<source url="https://example.com/source">https://example.com/source</source>`) {
		t.Fatal(output)
	}
	reused := RssItem{Source: "legacy", SourceInfo: &RssSource{URL: "old", Name: "old"}}
	if err := xml.Unmarshal([]byte(`<item/>`), &reused); err != nil {
		t.Fatal(err)
	}
	if reused.SourceInfo.URL != "old" {
		t.Fatal("missing source cleared old value")
	}
	if err := xml.Unmarshal([]byte(`<item><source url="new">new name</source></item>`), &reused); err != nil {
		t.Fatal(err)
	}
	if reused.Source != "new name" || reused.SourceInfo.URL != "new" {
		t.Fatal("explicit source replacement failed")
	}
}

func TestAtomEmptyLanguageReset(t *testing.T) {
	input := `<feed xmlns="http://www.w3.org/2005/Atom" xml:lang="en"><entry xml:lang=""><id>urn:one</id></entry><entry><id>urn:two</id></entry></feed>`
	var f AtomFeed
	if err := xml.Unmarshal([]byte(input), &f); err != nil {
		t.Fatal(err)
	}
	if !f.Entries[0].LanguageSet || f.Entries[1].LanguageSet {
		t.Fatal("explicit empty and absent language confused")
	}
	output, err := ToXML(&f)
	if err != nil {
		t.Fatal(err)
	}
	var again AtomFeed
	if err := xml.Unmarshal([]byte(output), &again); err != nil {
		t.Fatal(err)
	}
	if !again.Entries[0].LanguageSet || again.Entries[0].Language != "" || again.Entries[1].LanguageSet {
		t.Fatalf("reset lost: %s", output)
	}
	empty := AtomFeed{LanguageSet: true}
	b, err := xml.Marshal(empty)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`xml:lang=""`)) {
		t.Fatal(string(b))
	}
}

func TestNilItemsAreSkippedWithoutMutation(t *testing.T) {
	f := &Feed{Id: "urn:feed", Items: []*Item{nil, {Id: "one"}, nil, {Id: "two"}, nil}}
	atom := atomXMLForTest(t, f)
	var a AtomFeed
	if err := xml.Unmarshal([]byte(atom), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Entries) != 2 || a.Entries[0].Id != "one" || a.Entries[1].Id != "two" {
		t.Fatal(atom)
	}
	rss, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	var writer bytes.Buffer
	if err := f.WriteRss(&writer); err != nil {
		t.Fatal(err)
	}
	if rss != writer.String() {
		t.Fatal("RSS writer disagrees")
	}
	var r RssFeedXml
	if err := xml.Unmarshal([]byte(rss), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Channel.Items) != 2 {
		t.Fatal(rss)
	}
	j, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	writer.Reset()
	if err := f.WriteJSON(&writer); err != nil {
		t.Fatal(err)
	}
	var parsed JSONFeed
	if err := json.Unmarshal([]byte(j), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Items) != 2 || parsed.Items[1].Id != "two" {
		t.Fatal(j)
	}
	if len(f.Items) != 5 || f.Items[0] != nil || f.Items[2] != nil {
		t.Fatal("source items mutated")
	}
	onlyNil := &Feed{Items: []*Item{nil}}
	j, err = onlyNil.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(j, `"items": []`) {
		t.Fatal(j)
	}
}

func TestContentModuleStaysAnExtensionOutsideItems(t *testing.T) {
	var r RssFeedXml
	input := `<rss xmlns:c="http://purl.org/rss/1.0/modules/content/"><channel><c:encoded>channel payload</c:encoded><item><c:encoded>item body</c:encoded></item></channel></rss>`
	if err := xml.Unmarshal([]byte(input), &r); err != nil {
		t.Fatal(err)
	}
	if len(r.Channel.Extensions) != 1 || r.Channel.Extensions[0].Content[0].Text != "channel payload" || r.Channel.Items[0].Content.Content != "item body" {
		t.Fatal("content-module element routed to the wrong model")
	}
}
