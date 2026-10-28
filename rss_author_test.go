package feeds

import (
	"bytes"
	"encoding/xml"
	"net/mail"
	"reflect"
	"strings"
	"testing"
)

func TestRSSAuthorContacts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		author *Author
		want   string
	}{
		{"nil", nil, ""},
		{"empty", &Author{}, ""},
		{"name only", &Author{Name: "Alice"}, ""},
		{"whitespace email", &Author{Name: "Alice", Email: " \t "}, ""},
		{"invalid", &Author{Name: "Alice", Email: "not-an-email"}, ""},
		{"multiple mailboxes", &Author{Name: "Alice", Email: "alice@example.com, bob@example.com"}, ""},
		{"line break", &Author{Name: "Alice", Email: "alice@example.com\r\n"}, ""},
		{"email only", &Author{Email: "alice@example.com"}, "alice@example.com"},
		{"name and email", &Author{Name: "Alice", Email: "alice@example.com"}, "alice@example.com (Alice)"},
		{"trimmed", &Author{Name: " Alice ", Email: " alice@example.com "}, "alice@example.com (Alice)"},
		{"mailbox display name", &Author{Email: "Alice <alice@example.com>"}, "alice@example.com (Alice)"},
		{"explicit name wins", &Author{Name: "Bob", Email: "Alice <alice@example.com>"}, "alice@example.com (Bob)"},
		{"mailbox comment", &Author{Email: "alice@example.com (Alice)"}, "alice@example.com (Alice)"},
		{"quoted local part", &Author{Name: "Alice", Email: `"a b"@example.com`}, `"a b"@example.com (Alice)`},
		{"comment escapes", &Author{Name: `Alice (editor) \ team`, Email: "alice@example.com"}, `alice@example.com (Alice \(editor\) \\ team)`},
		{"unicode and XML", &Author{Name: "作者 <Alice> & Bob", Email: "alice@example.com"}, "alice@example.com (作者 <Alice> & Bob)"},
		{"name line breaks", &Author{Name: "Alice\nEditor", Email: "alice@example.com"}, "alice@example.com (Alice Editor)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before Author
			if tc.author != nil {
				before = *tc.author
			}
			f := &Feed{Title: "test", Link: &Link{Href: "https://example.com"}, Author: tc.author, Items: []*Item{{Title: "post", Author: tc.author}}}
			rss := (&Rss{f}).RssFeed()
			if rss.ManagingEditor != tc.want || rss.Items[0].Author != tc.want {
				t.Fatalf("want %q for both contacts; editor=%q author=%q", tc.want, rss.ManagingEditor, rss.Items[0].Author)
			}
			if tc.want != "" {
				if _, err := mail.ParseAddress(tc.want); err != nil {
					t.Fatalf("generated invalid mailbox %q: %v", tc.want, err)
				}
			}
			output, err := f.ToRss()
			if err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			if err := f.WriteRss(&buf); err != nil {
				t.Fatal(err)
			}
			if output != buf.String() {
				t.Fatal("ToRss and WriteRss disagree")
			}
			var decoded RssFeedXml
			if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Channel.ManagingEditor != tc.want || decoded.Channel.Items[0].Author != tc.want {
				t.Fatalf("XML contact roundtrip failed: %s", output)
			}
			if tc.want == "" && (strings.Contains(output, "<managingEditor>") || strings.Contains(output, "<author>")) {
				t.Fatalf("invalid or absent contacts must be omitted: %s", output)
			}
			if tc.author != nil && !reflect.DeepEqual(before, *tc.author) {
				t.Fatal("source author mutated")
			}
		})
	}
}

func TestRSSAuthorDoesNotInheritOrChangeOtherFormats(t *testing.T) {
	author := &Author{Name: "Alice", Email: "alice@example.com"}
	f := &Feed{Title: "test", Author: author, Items: []*Item{{Title: "no author"}, {Title: "name only", Author: &Author{Name: "Bob"}}}}
	rss := (&Rss{f}).RssFeed()
	for _, item := range rss.Items {
		if item.Author != "" {
			t.Fatalf("item author unexpectedly inherited: %q", item.Author)
		}
	}
	// Name-only authors remain available in Atom and JSON.
	atom := (&Atom{f}).AtomFeed()
	j := (&JSON{f}).JSONFeed()
	if atom.Author.Name != "Alice" || atom.Entries[1].Author.Name != "Bob" || j.Authors[0].Name != "Alice" || j.Items[1].Authors[0].Name != "Bob" {
		t.Fatal("non-RSS authors changed")
	}
	// Callers of the low-level RSS model retain control over raw field strings.
	direct := &RssFeed{ManagingEditor: "custom editor", Items: []*RssItem{{Author: "custom author"}}}
	output, err := ToXML(direct)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "<author>custom author</author>") || !strings.Contains(output, "<managingEditor>custom editor</managingEditor>") {
		t.Fatal(output)
	}
}
