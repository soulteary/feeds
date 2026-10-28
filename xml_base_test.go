package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestXMLBaseMappings(t *testing.T) {
	for _, tc := range []struct{ name, feed, item string }{
		{"absent", "", ""},
		{"feed context", "https://example.com/blog/", ""},
		{"relative item override", "https://example.com/blog/", "../posts/"},
		{"absolute item override", "https://example.com/blog/", "https://cdn.example.com/"},
		{"item only", "", "https://example.com/posts/"},
		{"Unicode and escaping", "https://example.com/技术/?a=1&b=2", `../a"b/`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := &Item{XMLBase: tc.item, Id: "urn:item", Link: &Link{Href: "post.html"}, Enclosure: &Enclosure{Url: "audio.mp3", Type: "audio/mpeg", Length: "42"}}
			f := &Feed{XMLBase: tc.feed, Id: "urn:feed", Link: &Link{Href: "index.html"}, Items: []*Item{item}}
			var jsonBefore string
			var err error
			// JSON has no xml:base equivalent and must not change when these fields vary.
			f.XMLBase = ""
			item.XMLBase = ""
			jsonBefore, err = f.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			f.XMLBase = tc.feed
			item.XMLBase = tc.item
			atom := atomXMLForTest(t, f)
			var decodedAtom AtomFeed
			if err := xml.Unmarshal([]byte(atom), &decodedAtom); err != nil {
				t.Fatal(err)
			}
			if decodedAtom.XMLBase != tc.feed || decodedAtom.Entries[0].XMLBase != tc.item {
				t.Fatalf("Atom bases lost: %s", atom)
			}
			var namespaced struct {
				Base    string `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
				Entries []struct {
					Links []struct {
						Href string `xml:"href,attr"`
					} `xml:"link"`
					Base string `xml:"http://www.w3.org/XML/1998/namespace base,attr"`
				} `xml:"entry"`
			}
			if err := xml.Unmarshal([]byte(atom), &namespaced); err != nil {
				t.Fatal(err)
			}
			if namespaced.Base != tc.feed || namespaced.Entries[0].Base != tc.item {
				t.Fatal("wrong XML base namespace")
			}
			if decodedAtom.Link.Href != "index.html" || namespaced.Entries[0].Links[0].Href != "post.html" || decodedAtom.Entries[0].Id != "urn:item" {
				t.Fatal("Atom links or IDs were rewritten")
			}
			rss, err := f.ToRss()
			if err != nil {
				t.Fatal(err)
			}
			var b bytes.Buffer
			if err := f.WriteRss(&b); err != nil {
				t.Fatal(err)
			}
			if rss != b.String() {
				t.Fatal("RSS writer disagrees")
			}
			var decodedRSS RssFeedXml
			if err := xml.Unmarshal([]byte(rss), &decodedRSS); err != nil {
				t.Fatal(err)
			}
			if decodedRSS.XMLBase != "" || decodedRSS.Channel.XMLBase != tc.feed || decodedRSS.Channel.Items[0].XMLBase != tc.item {
				t.Fatalf("wrong RSS base levels: %s", rss)
			}
			if decodedRSS.Channel.Link != "index.html" || decodedRSS.Channel.Items[0].Link != "post.html" || decodedRSS.Channel.Items[0].Enclosure.Url != "audio.mp3" {
				t.Fatal("RSS URLs were rewritten")
			}
			j, err := f.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			if j != jsonBefore {
				t.Fatal("xml:base changed JSON output")
			}
			if f.XMLBase != tc.feed || item.XMLBase != tc.item {
				t.Fatal("source bases mutated")
			}
			if tc.feed == "" && tc.item == "" && (strings.Contains(atom, "base=") || strings.Contains(rss, "base=")) {
				t.Fatal("empty bases should be omitted")
			}
		})
	}
}

func TestLowLevelXMLBaseRoundtrips(t *testing.T) {
	for _, tc := range []struct {
		input    string
		newValue func() interface{}
	}{
		{`<feed xml:base="https://example.com/"/>`, func() interface{} { return &AtomFeed{} }},
		{`<entry xml:base="../post/"/>`, func() interface{} { return &AtomEntry{} }},
		{`<channel xml:base="https://example.com/"/>`, func() interface{} { return &RssFeed{} }},
		{`<item xml:base="../post/"/>`, func() interface{} { return &RssItem{} }},
	} {
		value := tc.newValue()
		if err := xml.Unmarshal([]byte(tc.input), value); err != nil {
			t.Fatal(err)
		}
		want := reflect.ValueOf(value).Elem().FieldByName("XMLBase").String()
		for _, form := range []interface{}{value, reflect.ValueOf(value).Elem().Interface()} {
			output, err := xml.Marshal(form)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(output), "xml:base=") {
				t.Fatalf("missing xml:base: %s", output)
			}
			again := tc.newValue()
			if err := xml.Unmarshal(output, again); err != nil {
				t.Fatal(err)
			}
			if reflect.ValueOf(again).Elem().FieldByName("XMLBase").String() != want {
				t.Fatal("base roundtrip failed")
			}
		}
	}
}

func TestRSSRootBaseAndNamespaces(t *testing.T) {
	input := `<rss version="2.0" xml:base="https://example.com/root/" xmlns:content="http://purl.org/rss/1.0/modules/content/" xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"><channel xml:base="../show/"><title>show</title><item xml:base="episodes/"><title>post</title></item></channel></rss>`
	var r RssFeedXml
	if err := xml.Unmarshal([]byte(input), &r); err != nil {
		t.Fatal(err)
	}
	if r.XMLBase != "https://example.com/root/" || r.Channel.XMLBase != "../show/" || r.Channel.Items[0].XMLBase != "episodes/" {
		t.Fatal("root/channel/item contexts flattened")
	}
	output, err := ToXML(&r)
	if err != nil {
		t.Fatal(err)
	}
	var b bytes.Buffer
	if err := WriteXML(&r, &b); err != nil {
		t.Fatal(err)
	}
	if output != b.String() {
		t.Fatal("complete RSS writer disagrees")
	}
	var again RssFeedXml
	if err := xml.Unmarshal([]byte(output), &again); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(r, again) {
		t.Fatalf("root bases or namespace declarations lost: %s", output)
	}
	// Rewrapping only Channel intentionally drops root context: callers retaining
	// document-level attributes should pass the complete RssFeedXml to ToXML.
}

func TestXMLBaseNamespaceReuseAndExtensions(t *testing.T) {
	var a AtomFeed
	if err := xml.Unmarshal([]byte(`<feed xml:base="https://example.com/"/>`), &a); err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal([]byte(`<feed base="wrong" xmlns:x="urn:wrong" x:base="wrong"/>`), &a); err != nil {
		t.Fatal(err)
	}
	if a.XMLBase != "https://example.com/" {
		t.Fatal("non-XML base attribute replaced XML base")
	}
	if err := xml.Unmarshal([]byte(`<feed xml:base="relative/"/>`), &a); err != nil {
		t.Fatal(err)
	}
	if a.XMLBase != "relative/" {
		t.Fatal("explicit replacement lost")
	}
	f := &Feed{XMLBase: "https://example.com/", Extensions: []Extension{{XMLName: xml.Name{Space: "urn:payload", Local: "data"}, Attrs: []xml.Attr{{Name: xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "base"}, Value: "nested/"}}}}}
	output := atomXMLForTest(t, f)
	if err := xml.Unmarshal([]byte(output), &a); err != nil {
		t.Fatal(err)
	}
	if len(a.Extensions) != 1 || a.Extensions[0].Attrs[0].Name.Space != "http://www.w3.org/XML/1998/namespace" || a.Extensions[0].Attrs[0].Value != "nested/" {
		t.Fatal("nested extension base lost")
	}
	// Generic XML-only fields are omitted even when the generic object is marshaled directly.
	j, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(j), "XMLBase") {
		t.Fatal("generic XML base leaked into JSON")
	}
}
