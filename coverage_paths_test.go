package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

var errTestIO = errors.New("test I/O failure")

type failingIO struct{}

func (failingIO) Read([]byte) (int, error)  { return 0, errTestIO }
func (failingIO) Write([]byte) (int, error) { return 0, errTestIO }

func TestXMLDecodeFailures(t *testing.T) {
	for _, tc := range []struct {
		input string
		value interface{}
	}{
		{`<feed><entry>`, &AtomFeed{}},
		{`<entry><id>`, &AtomEntry{}},
		{`<channel><item>`, &RssFeed{}},
		{`<item><title>`, &RssItem{}},
		{`<rss><channel>`, &RssFeedXml{}},
		{`<category term="x">`, &AtomCategory{}},
		{`<encoded>`, &RssContent{}},
		{`<feed xmlns:x="urn:x"><x:custom><?unsupported value?></x:custom></feed>`, &AtomFeed{}},
	} {
		if err := xml.Unmarshal([]byte(tc.input), tc.value); err == nil {
			t.Fatalf("%T accepted malformed/unsupported XML: %s", tc.value, tc.input)
		}
	}
}

func TestXMLRejectedExtensionModels(t *testing.T) {
	invalid := Extension{XMLName: xml.Name{Space: ns, Local: "title"}}
	for _, value := range []interface{}{AtomEntry{Extensions: []Extension{invalid}}, RssFeed{Extensions: []Extension{{XMLName: xml.Name{Local: "unqualified"}}}}, RssItem{Extensions: []Extension{{XMLName: xml.Name{Local: "unqualified"}}}}, RssFeed{Extensions: []Extension{{XMLName: xml.Name{Space: ns, Local: "link"}}}}} {
		if _, err := xml.Marshal(value); err == nil {
			t.Fatalf("%T accepted invalid extension model", value)
		}
	}
	child := &Extension{XMLName: xml.Name{Space: "urn:x", Local: "bad:name"}}
	if _, err := xml.Marshal(Extension{XMLName: xml.Name{Space: "urn:x", Local: "root"}, Content: []ExtensionContent{{Element: child}}}); err == nil {
		t.Fatal("invalid child error was not propagated")
	}
}

func TestExtensionTextBufferingAndPrefixCollision(t *testing.T) {
	var x Extension
	if err := xml.Unmarshal([]byte(`<x xmlns="urn:x">one<![CDATA[two]]>three</x>`), &x); err != nil {
		t.Fatal(err)
	}
	if len(x.Content) != 1 || x.Content[0].Text != "onetwothree" {
		t.Fatalf("adjacent text not coalesced: %+v", x.Content)
	}
	x.Prefix = "extattr1"
	x.Attrs = []xml.Attr{{Name: xml.Name{Space: "urn:other", Local: "key"}, Value: "value"}}
	b, err := xml.Marshal(x)
	if err != nil {
		t.Fatal(err)
	}
	var again Extension
	if err := xml.Unmarshal(b, &again); err != nil {
		t.Fatal(err)
	}
	if again.XMLName.Space != "urn:x" || again.Attrs[0].Name.Space != "urn:other" {
		t.Fatalf("prefix collision changed namespace: %s", b)
	}
	// A failed sink must be reported even for a compact mixed-content buffer.
	if err := encodeExtensionContent(failingIO{}, []ExtensionContent{{Text: "payload"}}); !errors.Is(err, errTestIO) {
		t.Fatalf("lost compact-content write failure: %v", err)
	}
	reader := extensionTokenReader{done: true}
	if _, err := reader.Token(); !errors.Is(err, io.EOF) {
		t.Fatalf("completed reader: %v", err)
	}
}

func TestFeedMutationAndOutputErrors(t *testing.T) {
	f := &Feed{}
	item := &Item{Id: "one"}
	f.Add(item)
	if len(f.Items) != 1 || f.Items[0] != item {
		t.Fatal("Add did not retain item pointer")
	}
	if err := f.WriteAtom(failingIO{}); !errors.Is(err, errTestIO) {
		t.Fatalf("XML header write error lost: %v", err)
	}
	if err := f.WriteJSON(failingIO{}); !errors.Is(err, errTestIO) {
		t.Fatalf("JSON write error lost: %v", err)
	}
	invalid := time.Date(10000, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := (&JSONFeed{Items: []*JSONItem{{Id: "one", PublishedDate: &invalid}}}).ToJSON(); err == nil {
		t.Fatal("invalid timestamp encoding error lost")
	}
	source := &Item{Source: &Link{Href: "https://example.com/source"}}
	b, err := json.Marshal(newJSONItem(source))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(b, []byte(`"external_url":"https://example.com/source"`)) {
		t.Fatal(string(b))
	}
}

func TestRemainingOptionalRSSMetadata(t *testing.T) {
	closed := true
	extensions := (&ITunesItem{IsClosedCaptioned: &closed}).Extensions()
	if len(extensions) != 1 || extensions[0].Content[0].Text != "Yes" {
		t.Fatal("closed caption Yes missing")
	}
	child := itunesText("title", "nested")
	outer := Extension{XMLName: xml.Name{Space: "urn:outer", Local: "data"}, Content: []ExtensionContent{{Element: &child}}}
	if !hasITunesExtensions([]Extension{outer}) {
		t.Fatal("nested iTunes namespace not detected")
	}
	r := &RssFeedXml{Version: "2.0", Channel: &RssFeed{}}
	out, err := ToXML(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `xmlns:content="`+rssContentNamespace+`"`) || r.ContentNamespace != "" {
		t.Fatal("root default declaration missing or source mutated")
	}
	i := RssItem{Content: &RssContent{Content: "typed"}, Extensions: []Extension{{XMLName: xml.Name{Space: "urn:keep", Local: "data"}}}}
	out, err = ToXML(&RssFeed{Items: []*RssItem{&i}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "urn:keep") {
		t.Fatal("unrelated extension filtered with content")
	}
}

func TestUUIDEntropyFailure(t *testing.T) {
	defer func() {
		if got := recover(); !errors.Is(asError(got), errTestIO) {
			t.Fatalf("NewUUID entropy failure panic=%v", got)
		}
	}()
	newUUID(failingIO{})
}
func asError(value interface{}) error { err, _ := value.(error); return err }
