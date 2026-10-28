package feeds

// rss support
// validation done according to spec here:
//    http://cyber.law.harvard.edu/rss/rss.html

import (
	"encoding/xml"
	"fmt"
	"net/mail"
	"strings"
	"time"
)

// private wrapper around the RssFeed which gives us the <rss>..</rss> xml
type RssFeedXml struct {
	XMLName          xml.Name `xml:"rss"`
	Version          string   `xml:"version,attr"`
	ContentNamespace string   `xml:"xmlns:content,attr"`
	Channel          *RssFeed
}

type RssContent struct {
	XMLName xml.Name `xml:"content:encoded"`
	Content string   `xml:",cdata"`
}

type RssImage struct {
	XMLName xml.Name `xml:"image"`
	Url     string   `xml:"url"`
	Title   string   `xml:"title"`
	Link    string   `xml:"link"`
	Width   int      `xml:"width,omitempty"`
	Height  int      `xml:"height,omitempty"`
}

type RssTextInput struct {
	XMLName     xml.Name `xml:"textInput"`
	Title       string   `xml:"title"`
	Description string   `xml:"description"`
	Name        string   `xml:"name"`
	Link        string   `xml:"link"`
}

type RssFeed struct {
	XMLName        xml.Name      `xml:"channel"`
	Title          string        `xml:"title"` // required
	AtomLinks      []RssAtomLink `xml:"http://www.w3.org/2005/Atom link,omitempty"`
	Link           string        `xml:"link"`        // required
	Description    string        `xml:"description"` // required
	Language       string        `xml:"language,omitempty"`
	Copyright      string        `xml:"copyright,omitempty"`
	ManagingEditor string        `xml:"managingEditor,omitempty"` // Author used
	WebMaster      string        `xml:"webMaster,omitempty"`
	PubDate        string        `xml:"pubDate,omitempty"`       // created or updated
	LastBuildDate  string        `xml:"lastBuildDate,omitempty"` // updated used
	Category       string        `xml:"category,omitempty"`
	Generator      string        `xml:"generator,omitempty"`
	Docs           string        `xml:"docs,omitempty"`
	Cloud          string        `xml:"cloud,omitempty"`
	Ttl            int           `xml:"ttl,omitempty"`
	Rating         string        `xml:"rating,omitempty"`
	SkipHours      string        `xml:"skipHours,omitempty"`
	SkipDays       string        `xml:"skipDays,omitempty"`
	Image          *RssImage
	TextInput      *RssTextInput
	Items          []*RssItem `xml:"item"`
}

type RssItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`       // required
	Link        string   `xml:"link"`        // required
	Description string   `xml:"description"` // required
	Content     *RssContent
	Author      string `xml:"author,omitempty"`
	Category    string `xml:"category,omitempty"`
	Comments    string `xml:"comments,omitempty"`
	Enclosure   *RssEnclosure
	Guid        *RssGuid // Id used
	PubDate     string   `xml:"pubDate,omitempty"` // created or updated
	Source      string   `xml:"source,omitempty"`
}

type RssEnclosure struct {
	//RSS 2.0 <enclosure url="http://example.com/file.mp3" length="123456789" type="audio/mpeg" />
	XMLName xml.Name `xml:"enclosure"`
	Url     string   `xml:"url,attr"`
	Length  string   `xml:"length,attr"`
	Type    string   `xml:"type,attr"`
}

type RssGuid struct {
	//RSS 2.0 <guid isPermaLink="true">http://inessential.com/2002/09/01.php#a2</guid>
	XMLName     xml.Name `xml:"guid"`
	Id          string   `xml:",chardata"`
	IsPermaLink string   `xml:"isPermaLink,attr,omitempty"` // "true", "false", or an empty string
}

type Rss struct {
	*Feed
}

// create a new RssItem with a generic Item struct's data
func newRssItem(i *Item) *RssItem {
	item := &RssItem{
		Title:       i.Title,
		Description: i.Description,
		PubDate:     anyTimeFormat(time.RFC1123Z, i.Created, i.Updated),
	}
	if i.Id != "" {
		item.Guid = &RssGuid{Id: i.Id, IsPermaLink: i.IsPermaLink}
	}
	if home := homeLink(i.Link, i.Links); home != nil {
		item.Link = home.Href
	}
	if len(i.Content) > 0 {
		item.Content = &RssContent{Content: i.Content}
	}
	if i.Source != nil {
		item.Source = i.Source.Href
	}

	// Define a closure
	if i.Enclosure != nil && i.Enclosure.Type != "" && i.Enclosure.Length != "" {
		item.Enclosure = &RssEnclosure{Url: i.Enclosure.Url, Type: i.Enclosure.Type, Length: i.Enclosure.Length}
	}

	item.Author = rssAuthor(i.Author)
	return item
}

// create a new RssFeed with a generic Feed struct's data
func (r *Rss) RssFeed() *RssFeed {
	pub := anyTimeFormat(time.RFC1123Z, r.Created, r.Updated)
	build := anyTimeFormat(time.RFC1123Z, r.Updated)
	author := rssAuthor(r.Author)

	var image *RssImage
	if r.Image != nil {
		image = &RssImage{Url: r.Image.Url, Title: r.Image.Title, Link: r.Image.Link, Width: r.Image.Width, Height: r.Image.Height}
	}

	var href string
	if home := homeLink(r.Link, r.Links); home != nil {
		href = home.Href
	}
	channel := &RssFeed{
		Title:          r.Title,
		Link:           href,
		Description:    r.Description,
		ManagingEditor: author,
		PubDate:        pub,
		LastBuildDate:  build,
		Copyright:      r.Copyright,
		Image:          image,
	}
	for _, link := range atomLinks(r.Link, r.Links, "application/rss+xml") {
		if link.Rel == "self" || link.Rel == "hub" {
			channel.AtomLinks = append(channel.AtomLinks, RssAtomLink{Href: link.Href, Rel: link.Rel, Type: link.Type, Length: link.Length})
		}
	}
	for _, i := range r.Items {
		channel.Items = append(channel.Items, newRssItem(i))
	}
	return channel
}

// rssAuthor formats the optional RSS contact fields. A name alone is not an
// email address, so omit the field when no single valid mailbox is available.
func rssAuthor(author *Author) string {
	if author == nil || strings.ContainsAny(author.Email, "\r\n") {
		return ""
	}
	address, err := mail.ParseAddress(strings.TrimSpace(author.Email))
	if err != nil {
		return ""
	}
	// Address.String preserves quoting for local parts that need it. Remove its
	// enclosing angle brackets to retain the existing email (name) RSS format.
	mailbox := (&mail.Address{Address: address.Address}).String()
	mailbox = strings.TrimSuffix(strings.TrimPrefix(mailbox, "<"), ">")
	name := strings.TrimSpace(author.Name)
	if name == "" {
		name = address.Name
	}
	if name == "" {
		return mailbox
	}
	// Keep the display name inside one RFC-style comment, even when it contains
	// parentheses or backslashes, and prevent line breaks in the contact value.
	name = strings.NewReplacer("\\", "\\\\", "(", "\\(", ")", "\\)", "\r", " ", "\n", " ").Replace(name)
	return fmt.Sprintf("%s (%s)", mailbox, name)
}

// FeedXml returns an XML-Ready object for an Rss object
func (r *Rss) FeedXml() interface{} {
	// only generate version 2.0 feeds for now
	return r.RssFeed().FeedXml()

}

// FeedXml returns an XML-ready object for an RssFeed object
func (r *RssFeed) FeedXml() interface{} {
	return &RssFeedXml{
		Version:          "2.0",
		Channel:          r,
		ContentNamespace: "http://purl.org/rss/1.0/modules/content/",
	}
}

// RssAtomLink is an Atom-namespaced link in an RSS channel.
type RssAtomLink struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/Atom link"`
	Href    string   `xml:"href,attr"`
	Rel     string   `xml:"rel,attr,omitempty"`
	Type    string   `xml:"type,attr,omitempty"`
	Length  string   `xml:"length,attr,omitempty"`
}
