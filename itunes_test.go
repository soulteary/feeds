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

func itunesBoolForTest(v bool) *bool { return &v }
func itunesFieldsForTest(t *testing.T, extensions []Extension) map[string][]Extension {
	t.Helper()
	fields := make(map[string][]Extension)
	for _, extension := range extensions {
		if extension.XMLName.Space == ITunesNamespace {
			fields[extension.XMLName.Local] = append(fields[extension.XMLName.Local], extension)
		}
	}
	return fields
}
func itunesValueForTest(t *testing.T, fields map[string][]Extension, name, want string) {
	t.Helper()
	elements := fields[name]
	if len(elements) != 1 || len(elements[0].Content) != 1 || elements[0].Content[0].Text != want {
		t.Fatalf("%s: want %q, got %+v", name, want, elements)
	}
}

func TestITunesPodcastRSSMapping(t *testing.T) {
	show := &ITunesFeed{Title: "Show & more", Author: "Host", Subtitle: "Subtitle", Summary: "Summary <html>", Keywords: "one,two", Image: &ITunesImage{Href: "https://example.com/art.jpg?a=1&b=2"}, Categories: []ITunesCategory{{Text: "Society & Culture", Categories: []ITunesCategory{{Text: "Documentary"}}}, {Text: "Technology"}}, Explicit: itunesBoolForTest(false), Type: ITunesFeedTypeSerial, Owner: &ITunesOwner{Name: "Owner & team", Email: "owner@example.com"}, NewFeedURL: "https://example.com/new.xml", Block: true, Complete: true, ApplePodcastsVerify: "verification"}
	episode := &ITunesItem{Title: "Episode", Author: "Guest", Subtitle: "Sub", Summary: "Description", Keywords: "topic", Image: &ITunesImage{Href: "https://example.com/episode.jpg"}, Explicit: itunesBoolForTest(true), Duration: "90", Episode: 12, Season: 2, EpisodeType: ITunesEpisodeTypeFull, Block: true, IsClosedCaptioned: itunesBoolForTest(false), Order: 3}
	item := &Item{Id: "urn:episode:stable", IsPermaLink: "false", Title: "RSS title", Created: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Enclosure: &Enclosure{Url: "https://example.com/episode.mp3", Length: "1234", Type: "audio/mpeg"}, ITunes: episode}
	feed := &Feed{Title: "RSS show", Language: "en-US", ITunes: show, Items: []*Item{item}}
	sourceBefore, _ := json.Marshal(struct {
		Show    *ITunesFeed
		Episode *ITunesItem
	}{show, episode})
	output, err := feed.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	var writer bytes.Buffer
	if err := feed.WriteRss(&writer); err != nil {
		t.Fatal(err)
	}
	if output != writer.String() {
		t.Fatal("RSS writer disagrees")
	}
	if !strings.Contains(output, `xmlns:itunes="`+ITunesNamespace+`"`) || !strings.Contains(output, "<itunes:explicit") {
		t.Fatalf("missing conventional iTunes namespace/prefix: %s", output)
	}
	var decoded RssFeedXml
	if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatalf("invalid RSS: %v\n%s", err, output)
	}
	if decoded.ITunesNamespace != ITunesNamespace {
		t.Fatal("root namespace declaration was not decoded")
	}
	channel := itunesFieldsForTest(t, decoded.Channel.Extensions)
	entry := itunesFieldsForTest(t, decoded.Channel.Items[0].Extensions)
	for name, want := range map[string]string{"title": "Show & more", "author": "Host", "subtitle": "Subtitle", "summary": "Summary <html>", "keywords": "one,two", "explicit": "false", "type": "serial", "new-feed-url": show.NewFeedURL, "block": "Yes", "complete": "Yes", "applepodcastsverify": "verification"} {
		itunesValueForTest(t, channel, name, want)
	}
	for name, want := range map[string]string{"title": "Episode", "author": "Guest", "subtitle": "Sub", "summary": "Description", "keywords": "topic", "explicit": "true", "duration": "90", "episode": "12", "season": "2", "episodeType": "full", "block": "Yes", "isClosedCaptioned": "No", "order": "3"} {
		itunesValueForTest(t, entry, name, want)
	}
	if len(channel["category"]) != 2 || channel["category"][0].Attrs[0].Value != "Society & Culture" || channel["category"][0].Content[0].Element.Attrs[0].Value != "Documentary" {
		t.Fatalf("nested categories lost: %+v", channel["category"])
	}
	if channel["image"][0].Attrs[0].Value != show.Image.Href || entry["image"][0].Attrs[0].Value != episode.Image.Href {
		t.Fatal("image href lost")
	}
	owner := channel["owner"][0]
	if len(owner.Content) != 2 || owner.Content[0].Element.XMLName.Space != ITunesNamespace || owner.Content[1].Element.Content[0].Text != "owner@example.com" {
		t.Fatalf("owner structure lost: %+v", owner)
	}
	rssItem := decoded.Channel.Items[0]
	if rssItem.Guid.Id != item.Id || rssItem.Enclosure.Url != item.Enclosure.Url || rssItem.PubDate == "" || decoded.Channel.Title != "RSS show" || decoded.Channel.Language != "en-US" {
		t.Fatal("core podcast RSS fields changed")
	}
	// Typed metadata is RSS-only and does not modify source objects.
	atom, err := feed.ToAtom()
	if err != nil {
		t.Fatal(err)
	}
	j, err := feed.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(atom, ITunesNamespace) || strings.Contains(j, "itunes") || strings.Contains(j, "verification") {
		t.Fatal("iTunes leaked into another format")
	}
	sourceAfter, _ := json.Marshal(struct {
		Show    *ITunesFeed
		Episode *ITunesItem
	}{show, episode})
	if !bytes.Equal(sourceBefore, sourceAfter) {
		t.Fatal("typed source metadata mutated")
	}
	// Decode keeps metadata as general extensions; re-encoding preserves all tags.
	roundtrip, err := ToXML(decoded.Channel)
	if err != nil {
		t.Fatal(err)
	}
	var again RssFeedXml
	if err := xml.Unmarshal([]byte(roundtrip), &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded.Channel.Extensions, again.Channel.Extensions) || !reflect.DeepEqual(rssItem.Extensions, again.Channel.Items[0].Extensions) {
		t.Fatal("iTunes extensions lost during RSS roundtrip")
	}
}

func TestITunesOmissionAndExtensionMerging(t *testing.T) {
	for _, f := range []*Feed{{}, {ITunes: &ITunesFeed{}}, {Items: []*Item{{ITunes: &ITunesItem{}}}}} {
		output, err := f.ToRss()
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(output, ITunesNamespace) {
			t.Fatalf("empty metadata changed default output: %s", output)
		}
	}
	var show *ITunesFeed
	var episode *ITunesItem
	if len(show.Extensions()) != 0 || len(episode.Extensions()) != 0 {
		t.Fatal("nil helpers must be safe")
	}
	rawExplicit := Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: "explicit"}, Content: []ExtensionContent{{Text: "legacy"}}}
	rawUnknown := Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: "futureField"}, Content: []ExtensionContent{{Text: "keep"}}}
	r := &RssFeed{ITunes: &ITunesFeed{Explicit: itunesBoolForTest(false)}, Extensions: []Extension{rawExplicit, rawUnknown}, Items: []*RssItem{{ITunes: &ITunesItem{Explicit: itunesBoolForTest(false), Duration: "0", EpisodeType: ITunesEpisodeTypeTrailer}}}}
	before, _ := json.Marshal(r.Extensions)
	output, err := ToXML(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded RssFeedXml
	if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatal(err)
	}
	fields := itunesFieldsForTest(t, decoded.Channel.Extensions)
	itunesValueForTest(t, fields, "explicit", "false")
	itunesValueForTest(t, fields, "futureField", "keep")
	fields = itunesFieldsForTest(t, decoded.Channel.Items[0].Extensions)
	itunesValueForTest(t, fields, "explicit", "false")
	itunesValueForTest(t, fields, "duration", "0")
	itunesValueForTest(t, fields, "episodeType", "trailer")
	after, _ := json.Marshal(r.Extensions)
	if !bytes.Equal(before, after) {
		t.Fatal("merge mutated raw extensions")
	}
	if len((&ITunesFeed{Owner: &ITunesOwner{}, Image: &ITunesImage{}, Categories: []ITunesCategory{{}}}).Extensions()) != 0 {
		t.Fatal("empty nested fields must be omitted")
	}
	for _, kind := range []ITunesEpisodeType{ITunesEpisodeTypeFull, ITunesEpisodeTypeTrailer, ITunesEpisodeTypeBonus} {
		if string(kind) != strings.ToLower(string(kind)) {
			t.Fatal("episode type must be lowercase")
		}
	}
}

func TestITunesPrefixAttributeNamespaceCollision(t *testing.T) {
	extension := Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: "image"}, Prefix: "itunes", Attrs: []xml.Attr{{Name: xml.Name{Space: "https://example.com/itunes", Local: "id"}, Value: "other"}}}
	output, err := xml.Marshal(extension)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Extension
	if err := xml.Unmarshal(output, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.XMLName.Space != ITunesNamespace || decoded.Attrs[0].Name.Space != "https://example.com/itunes" {
		t.Fatalf("prefix collision changed namespaces: %s", output)
	}
	extension.Prefix = "xmlBad"
	if _, err := xml.Marshal(extension); err == nil {
		t.Fatal("reserved prefix accepted")
	}
}

func TestITunesDecodedMetadataReplacesTypedValues(t *testing.T) {
	r := RssFeed{ITunes: &ITunesFeed{Title: "old"}}
	input := `<channel xmlns:i="http://www.itunes.com/dtds/podcast-1.0.dtd"><i:title>new</i:title></channel>`
	if err := xml.Unmarshal([]byte(input), &r); err != nil {
		t.Fatal(err)
	}
	if r.ITunes != nil {
		t.Fatal("stale typed metadata would override decoded extensions")
	}
	output, err := ToXML(&r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(output, ">old<") || !strings.Contains(output, ">new<") {
		t.Fatalf("decoded metadata overridden: %s", output)
	}
}
