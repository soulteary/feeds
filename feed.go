package feeds

import (
	"encoding/json"
	"encoding/xml"
	"io"
	"sort"
	"time"
)

// Link describes a primary or additional link, including its relation, media
// type, and optional enclosure length.
type Link struct {
	Href, Rel, Type, Length string
}

// Author identifies the author of a feed or item.
type Author struct {
	Name, Email string
}

// Image describes a feed image. RSS uses all fields; Atom logo and JSON Feed
// icon use Url.
type Image struct {
	Url, Title, Link string
	Width, Height    int
}

// Enclosure describes a related media resource, with a URL, byte length, and
// MIME type.
type Enclosure struct {
	Url, Length, Type string
}

// Item represents one entry in a feed. Format-specific conversion methods map
// its shared metadata to RSS, Atom, and JSON Feed.
type Item struct {
	XMLBase     string      `json:"-"` // Optional XML base URI for the entry/item.
	ITunes      *ITunesItem `json:"-"` // Optional RSS-only podcast episode metadata.
	Title       string
	Link        *Link
	Source      *Link
	Author      *Author
	Description string // used as description in rss, summary in atom
	Id          string // used as guid in rss, id in atom
	IsPermaLink string // an optional parameter for guid in rss
	Updated     time.Time
	Created     time.Time
	Enclosure   *Enclosure
	Content     string
	Links       []*Link     // Additional links; Link remains the primary link.
	Language    string      // Item language override for Atom and JSON.
	Category    string      // Primary category.
	Categories  []string    // Categories combined with Category in stable order.
	Extensions  []Extension `json:"-"` // Foreign XML children for RSS and Atom.
}

// Feed represents shared syndication metadata and an ordered collection of items.
type Feed struct {
	XMLBase     string      `json:"-"` // Optional XML base URI for the feed/channel.
	ITunes      *ITunesFeed `json:"-"` // Optional RSS-only podcast show metadata.
	Title       string
	Link        *Link
	Description string
	Author      *Author
	Updated     time.Time
	Created     time.Time
	Id          string
	Subtitle    string
	Items       []*Item
	Copyright   string
	Image       *Image
	Links       []*Link     // Additional links, including explicit rel=self subscription URLs.
	Language    string      // Primary language of the feed.
	Category    string      // Primary category (RSS and Atom).
	Categories  []string    // Feed categories (RSS and Atom); not inherited by items.
	Extensions  []Extension `json:"-"` // Foreign XML children for RSS and Atom.
}

// Add appends item to the feed without copying it.
func (f *Feed) Add(item *Item) {
	f.Items = append(f.Items, item)
}

// anyTimeFormat returns the first nonzero time in the requested format, or "".
func anyTimeFormat(format string, times ...time.Time) string {
	for _, t := range times {
		if !t.IsZero() {
			return t.Format(format)
		}
	}
	return ""
}

// XmlFeed supplies the XML-ready representation used by ToXML and WriteXML.
// It is implemented by the RSS and Atom conversion and low-level models.
type XmlFeed interface {
	FeedXml() interface{}
}

// ToXML returns the indented XML representation of feed, preceded by the XML
// declaration. It returns an error if XML marshaling fails.
func ToXML(feed XmlFeed) (string, error) {
	x := feed.FeedXml()
	data, err := xml.MarshalIndent(x, "", "  ")
	if err != nil {
		return "", err
	}
	// strip empty line from default xml header
	s := xml.Header[:len(xml.Header)-1] + string(data)
	return s, nil
}

// WriteXML writes the XML declaration and indented representation of feed to w.
// It returns the first error from the writer or XML encoder; partial output may
// have been written when an error is returned.
func WriteXML(feed XmlFeed, w io.Writer) error {
	x := feed.FeedXml()
	// write default xml header, without the newline
	if _, err := w.Write([]byte(xml.Header[:len(xml.Header)-1])); err != nil {
		return err
	}
	e := xml.NewEncoder(w)
	e.Indent("", "  ")
	return e.Encode(x)
}

// ToAtom returns the feed as Atom XML, including the XML declaration.
func (f *Feed) ToAtom() (string, error) {
	a := &Atom{f}
	return ToXML(a)
}

// WriteAtom writes an Atom representation of this feed to the writer.
func (f *Feed) WriteAtom(w io.Writer) error {
	return WriteXML(&Atom{f}, w)
}

// ToRss returns the feed as RSS 2.0 XML, including the XML declaration.
func (f *Feed) ToRss() (string, error) {
	r := &Rss{f}
	return ToXML(r)
}

// WriteRss writes an RSS representation of this feed to the writer.
func (f *Feed) WriteRss(w io.Writer) error {
	return WriteXML(&Rss{f}, w)
}

// ToJSON returns the feed as indented JSON Feed 1.1.
func (f *Feed) ToJSON() (string, error) {
	j := &JSON{f}
	return j.ToJSON()
}

// WriteJSON writes indented JSON Feed 1.1 to w, followed by a newline.
// It returns errors from JSON encoding or the writer.
func (f *Feed) WriteJSON(w io.Writer) error {
	j := &JSON{f}
	feed := j.JSONFeed()

	e := json.NewEncoder(w)
	e.SetIndent("", "  ")
	return e.Encode(feed)
}

// Sort stably sorts the feed's Items in place using less.
// Items that compare equal retain their original order.
func (f *Feed) Sort(less func(a, b *Item) bool) {
	lessFunc := func(i, j int) bool {
		return less(f.Items[i], f.Items[j])
	}
	sort.SliceStable(f.Items, lessFunc)
}
