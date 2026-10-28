package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func TestGenericCategoryMappings(t *testing.T) {
	for _, tc := range []struct {
		name, primary    string
		categories, want []string
	}{
		{"absent", "", nil, nil},
		{"empty", "", []string{}, nil},
		{"legacy single", "Go", nil, []string{"Go"}},
		{"multiple", "", []string{"Go", "RSS"}, []string{"Go", "RSS"}},
		{"merge deduplicate", "Go", []string{"RSS", "Go", "", "RSS", "go"}, []string{"Go", "RSS", "go"}},
		{"unicode escaping whitespace", "技术 & Go", []string{`a"b<c>`, " spaced "}, []string{"技术 & Go", `a"b<c>`, " spaced "}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := &Feed{Id: "urn:feed", Category: tc.primary, Categories: tc.categories, Items: []*Item{{Id: "urn:item", Category: tc.primary, Categories: append([]string(nil), tc.categories...)}}}
			before, _ := json.Marshal(f)
			output := atomXMLForTest(t, f)
			var atom AtomFeed
			if err := xml.Unmarshal([]byte(output), &atom); err != nil {
				t.Fatal(err)
			}
			terms := func(categories []AtomCategory) []string {
				var result []string
				for _, category := range categories {
					result = append(result, category.Term)
				}
				return result
			}
			if !reflect.DeepEqual(terms(atom.Categories), tc.want) || !reflect.DeepEqual(terms(atom.Entries[0].Categories), tc.want) {
				t.Fatalf("Atom categories: %s", output)
			}
			if len(tc.want) > 0 && !strings.Contains(output, `<category term=`) {
				t.Fatalf("Atom category missing term attribute: %s", output)
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
			var r RssFeedXml
			if err := xml.Unmarshal([]byte(rss), &r); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(r.Channel.Categories, tc.want) || !reflect.DeepEqual(r.Channel.Items[0].Categories, tc.want) {
				t.Fatalf("RSS categories: %s", rss)
			}
			j, err := f.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			buf.Reset()
			if err := f.WriteJSON(&buf); err != nil {
				t.Fatal(err)
			}
			var decoded, writer JSONFeed
			if err := json.Unmarshal([]byte(j), &decoded); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(buf.Bytes(), &writer); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(decoded.Items[0].Tags, tc.want) || !reflect.DeepEqual(decoded, writer) {
				t.Fatalf("JSON tags: %s", j)
			}
			var root map[string]json.RawMessage
			if err := json.Unmarshal([]byte(j), &root); err != nil {
				t.Fatal(err)
			}
			if _, exists := root["tags"]; exists {
				t.Fatal("JSON Feed has no feed-level tags")
			}
			after, _ := json.Marshal(f)
			if !bytes.Equal(before, after) {
				t.Fatal("source categories mutated")
			}
		})
	}
}

func TestFeedCategoriesDoNotInherit(t *testing.T) {
	f := &Feed{Category: "feed only", Categories: []string{"another feed category"}, Items: []*Item{{Id: "urn:item"}}}
	if len((&Atom{f}).AtomFeed().Entries[0].Categories) != 0 || len((&Rss{f}).RssFeed().Items[0].Categories) != 0 || len((&JSON{f}).JSONFeed().Items[0].Tags) != 0 {
		t.Fatal("feed categories inherited by item")
	}
}

func TestLowLevelCategoryRoundtrips(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		newValue    func() interface{}
	}{
		{"RSS feed", `<channel><category>one</category><category>two</category></channel>`, func() interface{} { return &RssFeed{} }},
		{"RSS item", `<item><category>one</category><category>two</category></item>`, func() interface{} { return &RssItem{} }},
		{"Atom feed", `<feed xmlns="http://www.w3.org/2005/Atom"><category term="one" scheme="https://example.com/taxonomy" label="One &amp; only"/><category term="two"/></feed>`, func() interface{} { return &AtomFeed{} }},
		{"Atom entry", `<entry xmlns="http://www.w3.org/2005/Atom"><category term="one" scheme="https://example.com/taxonomy" label="One &amp; only"/><category term="two"/></entry>`, func() interface{} { return &AtomEntry{} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			value := tc.newValue()
			if err := xml.Unmarshal([]byte(tc.input), value); err != nil {
				t.Fatal(err)
			}
			if reflect.ValueOf(value).Elem().FieldByName("Category").String() != "one" {
				t.Fatal("legacy Category does not mirror first term")
			}
			for _, form := range []interface{}{value, reflect.ValueOf(value).Elem().Interface()} {
				b, err := xml.Marshal(form)
				if err != nil {
					t.Fatal(err)
				}
				if strings.Count(string(b), "<category") != 2 {
					t.Fatalf("duplicate/lost categories: %s", b)
				}
				again := tc.newValue()
				if err := xml.Unmarshal(b, again); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(reflect.ValueOf(value).Elem().FieldByName("Categories").Interface(), reflect.ValueOf(again).Elem().FieldByName("Categories").Interface()) {
					t.Fatalf("roundtrip lost category metadata: %s", b)
				}
			}
			// Missing categories preserve reused values; present categories replace them.
			root := strings.Split(tc.input, ">")[0] + ">"
			end := "</" + strings.Fields(strings.TrimPrefix(root, "<"))[0]
			end = strings.TrimSuffix(end, ">") + ">"
			if err := xml.Unmarshal([]byte(root+end), value); err != nil {
				t.Fatal(err)
			}
			if reflect.ValueOf(value).Elem().FieldByName("Category").String() != "one" {
				t.Fatal("missing category cleared old value")
			}
			rootName := reflect.ValueOf(value).Elem().FieldByName("XMLName").Interface().(xml.Name).Local
			replacement := "<" + rootName + "><category>replacement</category></" + rootName + ">"
			if err := xml.Unmarshal([]byte(replacement), value); err != nil {
				t.Fatal(err)
			}
			if reflect.ValueOf(value).Elem().FieldByName("Category").String() != "replacement" || reflect.ValueOf(value).Elem().FieldByName("Categories").Len() != 1 {
				t.Fatal("explicit category replacement retained stale values")
			}

		})
	}
}

func TestLegacyAtomCategoryAndRichCategories(t *testing.T) {
	for _, value := range []interface{}{&AtomFeed{Category: "old & term"}, &AtomEntry{Category: "old & term"}} {
		b, err := xml.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(b), `<category term="old &amp; term"`) {
			t.Fatalf("legacy Category not mapped to term: %s", b)
		}
	}
	var category AtomCategory
	if err := xml.Unmarshal([]byte(`<category>legacy &amp; text</category>`), &category); err != nil {
		t.Fatal(err)
	}
	if category.Term != "legacy & text" {
		t.Fatalf("legacy text lost: %+v", category)
	}
	if err := xml.Unmarshal([]byte(`<category term="standard">ignored legacy text</category>`), &category); err != nil {
		t.Fatal(err)
	}
	if category.Term != "standard" {
		t.Fatal("text overrode term")
	}
	value := AtomFeed{Category: "same", Categories: []AtomCategory{{Term: "same", Scheme: "https://example.com/one", Label: "One"}, {Term: "same", Scheme: "https://example.com/two", Label: "Two"}, {Term: ""}, {Term: "same", Scheme: "https://example.com/one", Label: "One"}}}
	before, _ := json.Marshal(value)
	b, err := xml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(b), "<category") != 2 || !strings.Contains(string(b), `scheme="https://example.com/one"`) {
		t.Fatalf("rich categories lost or duplicated: %s", b)
	}
	after, _ := json.Marshal(value)
	if !bytes.Equal(before, after) {
		t.Fatal("source rich categories mutated")
	}
}
