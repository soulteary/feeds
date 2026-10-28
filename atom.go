package feeds

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"time"
)

// Generates Atom feed as XML

const ns = "http://www.w3.org/2005/Atom"

type AtomPerson struct {
	Name  string `xml:"name,omitempty"`
	Uri   string `xml:"uri,omitempty"`
	Email string `xml:"email,omitempty"`
}

type AtomSummary struct {
	XMLName xml.Name `xml:"summary"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomContent struct {
	XMLName xml.Name `xml:"content"`
	Content string   `xml:",chardata"`
	Type    string   `xml:"type,attr"`
}

type AtomAuthor struct {
	XMLName xml.Name `xml:"author"`
	AtomPerson
}

type AtomContributor struct {
	XMLName xml.Name `xml:"contributor"`
	AtomPerson
}

type AtomEntry struct {
	XMLBase     string         `xml:"http://www.w3.org/XML/1998/namespace base,attr,omitempty"`
	Extensions  []Extension    `xml:"-"`
	XMLName     xml.Name       `xml:"entry"`
	Xmlns       string         `xml:"xmlns,attr,omitempty"`
	Language    string         `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Title       string         `xml:"title"`   // required
	Updated     string         `xml:"updated"` // required
	Id          string         `xml:"id"`      // required
	Category    string         `xml:"-"`
	Categories  []AtomCategory `xml:"category,omitempty"`
	Content     *AtomContent
	Rights      string `xml:"rights,omitempty"`
	Source      string `xml:"source,omitempty"`
	Published   string `xml:"published,omitempty"`
	Contributor *AtomContributor
	Links       []AtomLink   // required if no child 'content' elements
	Summary     *AtomSummary // required if content has src or content is base64
	Author      *AtomAuthor  // required if feed lacks an author
}

// Multiple links with different rel can coexist
type AtomLink struct {
	//Atom 1.0 <link rel="enclosure" type="audio/mpeg" title="MP3" href="http://www.example.org/myaudiofile.mp3" length="1234" />
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Length  string   `xml:"length,attr,omitempty"`
}

type AtomFeed struct {
	XMLBase     string         `xml:"http://www.w3.org/XML/1998/namespace base,attr,omitempty"`
	Extensions  []Extension    `xml:"-"`
	XMLName     xml.Name       `xml:"feed"`
	Xmlns       string         `xml:"xmlns,attr"`
	Language    string         `xml:"http://www.w3.org/XML/1998/namespace lang,attr,omitempty"`
	Title       string         `xml:"title"`   // required
	Id          string         `xml:"id"`      // required
	Updated     string         `xml:"updated"` // required
	Category    string         `xml:"-"`
	Categories  []AtomCategory `xml:"category,omitempty"`
	Icon        string         `xml:"icon,omitempty"`
	Logo        string         `xml:"logo,omitempty"`
	Rights      string         `xml:"rights,omitempty"` // copyright used
	Subtitle    string         `xml:"subtitle,omitempty"`
	Link        *AtomLink      `xml:"-"`              // First link, retained for compatibility.
	Links       []AtomLink     `xml:"link,omitempty"` // Additional links.
	Author      *AtomAuthor    `xml:"author,omitempty"`
	Contributor *AtomContributor
	Entries     []*AtomEntry `xml:"entry"`
}

type Atom struct {
	*Feed
}

func newAtomEntry(i *Item) *AtomEntry {
	id := i.Id
	link := i.Link
	if link == nil {
		link = homeLink(nil, i.Links)
	}
	if link == nil {
		link = &Link{}
	}
	if len(id) == 0 {
		// if there's no id set, try to create one, either from data or just a uuid
		if len(link.Href) > 0 && (!i.Created.IsZero() || !i.Updated.IsZero()) {
			dateStr := anyTimeFormat("2006-01-02", i.Updated, i.Created)
			host, path := link.Href, "/invalid.html"
			if url, err := url.Parse(link.Href); err == nil {
				host, path = url.Host, url.Path
			}
			id = fmt.Sprintf("tag:%s,%s:%s", host, dateStr, path)
		} else {
			id = "urn:uuid:" + NewUUID().String()
		}
	}
	var name, email string
	if i.Author != nil {
		name, email = i.Author.Name, i.Author.Email
	}

	link_rel := link.Rel
	if link_rel == "" {
		link_rel = "alternate"
	}
	x := &AtomEntry{
		Title:     i.Title,
		XMLBase:   i.XMLBase,
		Language:  i.Language,
		Links:     []AtomLink{{Href: link.Href, Rel: link_rel, Type: link.Type}},
		Id:        id,
		Updated:   anyTimeFormat(time.RFC3339, i.Updated, i.Created),
		Published: anyTimeFormat(time.RFC3339, i.Created),
	}

	x.Extensions = i.Extensions
	x.Category = i.Category
	x.Categories = atomCategoriesFromNames(i.Category, i.Categories)
	if len(x.Categories) > 0 {
		x.Category = x.Categories[0].Term
	}
	if (i.Link != nil && i.Link.Href != "") || len(i.Links) > 0 {
		x.Links = atomLinks(i.Link, i.Links, "")
	}

	// if there's a description, assume it's html
	if len(i.Description) > 0 {
		x.Summary = &AtomSummary{Content: i.Description, Type: "html"}
	}

	// if there's a content, assume it's html
	if len(i.Content) > 0 {
		x.Content = &AtomContent{Content: i.Content, Type: "html"}
	}

	if i.Enclosure != nil && !hasEnclosure(x.Links, i.Enclosure.Url) {
		x.Links = append(x.Links, AtomLink{Href: i.Enclosure.Url, Rel: "enclosure", Type: i.Enclosure.Type, Length: i.Enclosure.Length})
	}

	if len(name) > 0 || len(email) > 0 {
		x.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: name, Email: email}}
	}
	return x
}

// create a new AtomFeed with a generic Feed struct's data
func (a *Atom) AtomFeed() *AtomFeed {
	updated := anyTimeFormat(time.RFC3339, a.Updated, a.Created)
	link := a.Link
	if link == nil {
		link = &Link{}
	}
	id := a.Id
	if id == "" {
		id = link.Href
	}
	if id == "" {
		if home := homeLink(a.Link, a.Links); home != nil {
			id = home.Href
		} else if self := selfLink(a.Link, a.Links, "application/atom+xml"); self != nil {
			id = self.Href
		}
	}
	feed := &AtomFeed{
		Xmlns:    ns,
		Title:    a.Title,
		XMLBase:  a.XMLBase,
		Language: a.Language,
		Link:     &AtomLink{Href: link.Href, Rel: link.Rel},
		Subtitle: a.Description,
		Id:       id,
		Updated:  updated,
		Rights:   a.Copyright,
	}
	feed.Extensions = a.Extensions
	feed.Category = a.Category
	feed.Categories = atomCategoriesFromNames(a.Category, a.Categories)
	if len(feed.Categories) > 0 {
		feed.Category = feed.Categories[0].Term
	}
	if (a.Link != nil && a.Link.Href != "") || len(a.Links) > 0 {
		feed.Link = nil
		links := atomLinks(a.Link, a.Links, "application/atom+xml")
		if len(links) > 0 {
			feed.Link = &links[0]
			feed.Links = links[1:]
		}
	}
	if a.Author != nil {
		feed.Author = &AtomAuthor{AtomPerson: AtomPerson{Name: a.Author.Name, Email: a.Author.Email}}
	}
	for _, e := range a.Items {
		feed.Entries = append(feed.Entries, newAtomEntry(e))
	}
	return feed
}

// FeedXml returns an XML-Ready object for an Atom object
func (a *Atom) FeedXml() interface{} {
	return a.AtomFeed()
}

// FeedXml returns an XML-ready object for an AtomFeed object
func (a *AtomFeed) FeedXml() interface{} {
	return a
}

// MarshalXML writes Link followed by Links without changing the source feed.
func (a AtomFeed) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := validateExtensions(a.Extensions, ns); err != nil {
		return err
	}
	type atomFeed AtomFeed
	copy := atomFeed(a)
	copy.Categories = atomCategories(a.Category, a.Categories)
	copy.Links = append([]AtomLink(nil), a.Links...)
	if a.Link != nil {
		copy.Links = append([]AtomLink{*a.Link}, copy.Links...)
	}
	start.Name = a.XMLName
	if start.Name.Local == "" {
		start.Name.Local = "feed"
	}
	if a.Xmlns != "" {
		start.Name.Space = ""
	}
	return e.EncodeElement(struct {
		atomFeed
		XMLChildren []Extension `xml:",any"`
	}{atomFeed: copy, XMLChildren: a.Extensions}, start)
}

// UnmarshalXML keeps the first link in Link and subsequent links in Links.
func (a *AtomFeed) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type atomFeed AtomFeed
	decoded := atomFeed(*a)
	decoded.Links = nil
	decoded.Categories = nil
	extensions, err := decodeExtensions(d, start, &decoded, ns, false)
	if err != nil {
		return err
	}
	if extensions != nil {
		decoded.Extensions = extensions
	}
	if len(decoded.Links) == 0 {
		decoded.Links = a.Links
	} else {
		first := decoded.Links[0]
		decoded.Link = &first
		if len(decoded.Links) > 1 {
			decoded.Links = decoded.Links[1:]
		} else {
			decoded.Links = nil
		}
	}
	if len(decoded.Categories) > 0 {
		decoded.Category = decoded.Categories[0].Term
	} else {
		decoded.Categories = a.Categories
	}
	*a = AtomFeed(decoded)
	return nil
}

// MarshalXML emits categories using Atom term attributes, including Category.
func (a AtomEntry) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	if err := validateExtensions(a.Extensions, ns); err != nil {
		return err
	}
	type plain AtomEntry
	value := plain(a)
	value.Categories = atomCategories(a.Category, a.Categories)
	start.Name = a.XMLName
	if start.Name.Local == "" {
		start.Name.Local = "entry"
	}
	if a.Xmlns != "" {
		start.Name.Space = ""
	}
	return e.EncodeElement(struct {
		plain
		XMLChildren []Extension `xml:",any"`
	}{plain: value, XMLChildren: a.Extensions}, start)
}

// UnmarshalXML retains category metadata and mirrors the first term in Category.
func (a *AtomEntry) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	type plain AtomEntry
	value := plain(*a)
	value.Categories = nil
	extensions, err := decodeExtensions(d, start, &value, ns, false)
	if err != nil {
		return err
	}
	if extensions != nil {
		value.Extensions = extensions
	}
	if len(value.Categories) > 0 {
		value.Category = value.Categories[0].Term
	} else {
		value.Categories = a.Categories
	}
	*a = AtomEntry(value)
	return nil
}
