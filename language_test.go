package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestLanguageMappings(t *testing.T) {
	for _, tc := range []struct{ name, feed, item string }{
		{"absent", "", ""},
		{"feed default", "en-US", ""},
		{"item override", "en-US", "zh-Hant-TW"},
		{"item only", "", "ja"},
		{"private use and script", "x-private-company", "sr-Latn-RS"},
		{"no forced validation or normalization", "legacy_CUSTOM", "MiXeD-value"},
		{"escaped strings remain caller controlled", `x-a&b`, `x-"<tag>`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Feed{Id: "urn:feed", Language: tc.feed, Items: []*Item{{Id: "urn:item", Language: tc.item}}}
			before, _ := json.Marshal(f)
			output := atomXMLForTest(t, f)
			var atom AtomFeed
			if err := xml.Unmarshal([]byte(output), &atom); err != nil {
				t.Fatal(err)
			}
			if atom.Language != tc.feed || atom.Entries[0].Language != tc.item {
				t.Fatalf("Atom language lost: %s", output)
			}
			if tc.feed == "" && tc.item == "" && strings.Contains(output, "lang=") {
				t.Fatalf("empty languages should be omitted: %s", output)
			}
			// Verify the reserved XML namespace, not merely an attribute named lang.
			var namespaced struct {
				Language string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
				Entries  []struct {
					Language string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
				} `xml:"entry"`
			}
			if err := xml.Unmarshal([]byte(output), &namespaced); err != nil {
				t.Fatal(err)
			}
			if namespaced.Language != tc.feed || namespaced.Entries[0].Language != tc.item {
				t.Fatal("Atom language used wrong namespace")
			}
			if tc.feed != "" && !strings.Contains(output, `xml:lang=`) {
				t.Fatalf("expected xml:lang: %s", output)
			}
			rss, err := f.ToRss()
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := f.WriteRss(&buf); err != nil {
				t.Fatal(err)
			}
			if buf.String() != rss {
				t.Fatal("RSS writer disagrees")
			}
			var decodedRSS RssFeedXml
			if err := xml.Unmarshal([]byte(rss), &decodedRSS); err != nil {
				t.Fatal(err)
			}
			if decodedRSS.Channel.Language != tc.feed {
				t.Fatalf("RSS language lost: %s", rss)
			}
			var rssItems struct {
				Channel struct {
					Items []struct {
						Language *string `xml:"language"`
					} `xml:"item"`
				} `xml:"channel"`
			}
			if err := xml.Unmarshal([]byte(rss), &rssItems); err != nil {
				t.Fatal(err)
			}
			if rssItems.Channel.Items[0].Language != nil {
				t.Fatalf("invented nonstandard RSS item language: %s", rss)
			}
			j, err := f.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			buf.Reset()
			if err := f.WriteJSON(&buf); err != nil {
				t.Fatal(err)
			}
			var decodedJSON, writer JSONFeed
			if err := json.Unmarshal([]byte(j), &decodedJSON); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(buf.Bytes(), &writer); err != nil {
				t.Fatal(err)
			}
			if decodedJSON.Language != tc.feed || decodedJSON.Items[0].Language != tc.item || !reflect.DeepEqual(decodedJSON, writer) {
				t.Fatalf("JSON language lost: %s", j)
			}
			if tc.feed == "" && tc.item == "" && strings.Contains(j, `"language"`) {
				t.Fatalf("empty JSON language must be omitted: %s", j)
			}
			if tc.item == "" && (atom.Entries[0].Language != "" || decodedJSON.Items[0].Language != "") {
				t.Fatal("default copied into item instead of remaining inherited")
			}
			after, _ := json.Marshal(f)
			if !bytes.Equal(before, after) {
				t.Fatal("source languages mutated")
			}
		})
	}
}

func TestLowLevelLanguageRoundtrips(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		newValue    func() interface{}
	}{
		{"RSS", `<channel><language>en-us</language></channel>`, func() interface{} { return &RssFeed{} }},
		{"Atom feed", `<feed xml:lang="zh-Hant"><id>urn:test</id></feed>`, func() interface{} { return &AtomFeed{} }},
		{"Atom entry", `<entry xml:lang="ja"><id>urn:item</id></entry>`, func() interface{} { return &AtomEntry{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := tc.newValue()
			if err := xml.Unmarshal([]byte(tc.input), value); err != nil {
				t.Fatal(err)
			}
			want := reflect.ValueOf(value).Elem().FieldByName("Language").String()
			if want == "" {
				t.Fatal("language was not decoded")
			}
			for _, form := range []interface{}{value, reflect.ValueOf(value).Elem().Interface()} {
				output, err := xml.Marshal(form)
				if err != nil {
					t.Fatal(err)
				}
				again := tc.newValue()
				if err := xml.Unmarshal(output, again); err != nil {
					t.Fatal(err)
				}
				if reflect.ValueOf(again).Elem().FieldByName("Language").String() != want {
					t.Fatalf("language roundtrip failed: %s", output)
				}
			}
		})
	}
	for _, value := range []interface{}{&JSONFeed{Language: "en-US"}, &JSONItem{Language: "zh-Hant"}} {
		output, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		again := reflect.New(reflect.TypeOf(value).Elem()).Interface()
		if err := json.Unmarshal(output, again); err != nil {
			t.Fatal(err)
		}
		if reflect.ValueOf(again).Elem().FieldByName("Language").String() != reflect.ValueOf(value).Elem().FieldByName("Language").String() {
			t.Fatalf("JSON language roundtrip failed: %s", output)
		}
	}
}

func TestAtomLanguageNamespaceAndReuse(t *testing.T) {
	var feed AtomFeed
	if err := xml.Unmarshal([]byte(`<feed xmlns:xml="http://www.w3.org/XML/1998/namespace" xml:lang="en"/>`), &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Language != "en" {
		t.Fatal("XML language namespace not recognized")
	}
	if err := xml.Unmarshal([]byte(`<feed><title>new</title></feed>`), &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Language != "en" {
		t.Fatal("missing language cleared previous value")
	}
	if err := xml.Unmarshal([]byte(`<feed xml:lang="fr"/>`), &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Language != "fr" {
		t.Fatal("explicit language failed to replace previous value")
	}
	if err := xml.Unmarshal([]byte(`<feed lang="de" xmlns:other="urn:other" other:lang="it"/>`), &feed); err != nil {
		t.Fatal(err)
	}
	if feed.Language != "fr" {
		t.Fatal("non-XML language attribute overrode xml:lang")
	}
}
