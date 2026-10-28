package feeds

import (
	"encoding/xml"
	"fmt"
	"io"
	"testing"
	"time"
)

var perfString string
var perfValue interface{}

func perfFeed(n int, extended bool) *Feed {
	f := &Feed{Title: "Example", Id: "urn:example", Link: &Link{Href: "https://example.com"}, Description: "example feed", Updated: time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)}
	for i := 0; i < n; i++ {
		item := &Item{Title: fmt.Sprintf("Item %d", i), Id: fmt.Sprintf("urn:item:%d", i), Link: &Link{Href: fmt.Sprintf("https://example.com/%d", i)}, Created: f.Updated, Content: "<p>A short article with words &amp; escaped text.</p>"}
		if extended {
			item.Extensions = []Extension{{XMLName: xml.Name{Space: "urn:example:extension", Local: "metadata"}, Attrs: []xml.Attr{{Name: xml.Name{Local: "kind"}, Value: "sample"}}, Content: []ExtensionContent{{Element: &Extension{XMLName: xml.Name{Space: "urn:example:extension", Local: "value"}, Content: []ExtensionContent{{Text: "metadata text"}}}}}}}
		}
		f.Items = append(f.Items, item)
	}
	return f
}
func BenchmarkFeedFormats(b *testing.B) {
	for _, n := range []int{10, 100, 1000} {
		f := perfFeed(n, false)
		a := (&Atom{f}).AtomFeed()
		r := (&Rss{f}).RssFeed()
		for name, run := range map[string]func(){"ToAtom": func() {
			s, e := f.ToAtom()
			if e != nil {
				b.Fatal(e)
			}
			perfString = s
		}, "WriteAtom": func() {
			if e := f.WriteAtom(io.Discard); e != nil {
				b.Fatal(e)
			}
		}, "ToRSS": func() {
			s, e := f.ToRss()
			if e != nil {
				b.Fatal(e)
			}
			perfString = s
		}, "ToJSON": func() {
			s, e := f.ToJSON()
			if e != nil {
				b.Fatal(e)
			}
			perfString = s
		}, "AtomToFeed": func() {
			v, e := a.ToFeed()
			if e != nil {
				b.Fatal(e)
			}
			perfValue = v
		}, "RSSToFeed": func() {
			v, e := r.ToFeed()
			if e != nil {
				b.Fatal(e)
			}
			perfValue = v
		}} {
			b.Run(fmt.Sprintf("%s/%d", name, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					run()
				}
			})
		}
	}
}
func BenchmarkXMLExtensions(b *testing.B) {
	f := perfFeed(100, true)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s, e := f.ToAtom()
		if e != nil {
			b.Fatal(e)
		}
		perfString = s
	}
}
