package feeds

import (
	"fmt"
	"html"
	"net/mail"
	"net/url"
	"strings"
	"time"
)

// conversionDate accepts the Atom timestamp format or the four RSS date layouts.
// Missing dates remain zero; invalid dates return an error rather than disappear.
func conversionDate(value, field string, atom bool) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822}
	if atom {
		layouts = []string{time.RFC3339Nano}
	}
	for _, layout := range layouts {
		location := time.UTC
		parseValue := value
		if !atom && (layout == time.RFC1123 || layout == time.RFC822) {
			parts := strings.Fields(value)
			zone := ""
			if len(parts) > 0 {
				zone = parts[len(parts)-1]
			}
			offsets := map[string]int{"UT": 0, "UTC": 0, "GMT": 0, "EST": -5, "EDT": -4, "CST": -6, "CDT": -5, "MST": -7, "MDT": -6, "PST": -8, "PDT": -7}
			hours, ok := offsets[zone]
			if !ok {
				continue
			}
			location = time.FixedZone(zone, hours*3600)
			if zone == "UT" {
				parseValue = strings.TrimSuffix(value, "UT") + "UTC"
			}
		}
		if date, err := time.ParseInLocation(layout, parseValue, location); err == nil {
			return date, nil
		}
	}
	return time.Time{}, fmt.Errorf("feeds: invalid %s date %q", field, value)
}

// Shared Content and Description carry HTML, as used by existing serializers.
func conversionAtomText(value, kind string) (string, error) {
	switch kind {
	case "", "text", "text/plain":
		return html.EscapeString(value), nil
	case "html", "text/html":
		return value, nil
	default:
		return "", fmt.Errorf("feeds: unsupported Atom text/content type %q", kind)
	}
}

func conversionAuthor(value string) *Author {
	if value == "" {
		return nil
	}
	if address, err := mail.ParseAddress(value); err == nil {
		return &Author{Name: address.Name, Email: address.Address}
	}
	return &Author{Name: value}
}

func conversionAtomAuthor(value *AtomAuthor) *Author {
	if value == nil {
		return nil
	}
	return &Author{Name: value.Name, Email: value.Email}
}

// cloneExtensions copies the extension tree so conversion results can be edited.
func cloneExtensions(values []Extension) []Extension {
	if values == nil {
		return nil
	}
	result := make([]Extension, len(values))
	for i, value := range values {
		result[i] = value
		result[i].Attrs = append(value.Attrs[:0:0], value.Attrs...)
		result[i].Content = append(value.Content[:0:0], value.Content...)
		for j, part := range value.Content {
			if part.Element != nil {
				child := cloneExtensions([]Extension{*part.Element})
				result[i].Content[j].Element = &child[0]
			}
		}
	}
	return result
}

func conversionAtomLinks(first *AtomLink, links []AtomLink) (*Link, []*Link) {
	var all []*Link
	if first != nil {
		all = append(all, &Link{Href: first.Href, Rel: first.Rel, Type: first.Type, Length: first.Length})
	}
	for _, link := range links {
		all = append(all, &Link{Href: link.Href, Rel: link.Rel, Type: link.Type, Length: link.Length})
	}
	// Keep all relations; only a page link may become the primary home page.
	primary := homeLink(nil, all)
	var rest []*Link
	removed := false
	for _, link := range all {
		if primary != nil && !removed && *link == *primary {
			removed = true
			continue
		}
		rest = append(rest, link)
	}
	return primary, rest
}

func conversionAtomCategories(primary string, values []AtomCategory) []string {
	names := make([]string, 0, len(values))
	for _, value := range values {
		names = append(names, value.Term)
	}
	return categoryNames(primary, names)
}

// ToItem converts a modeled Atom entry to the shared item model. Dates are
// validated. Format-specific metadata without a shared field is not retained;
// see README for the conversion boundaries. The receiver is not modified.
func (a *AtomEntry) ToItem() (*Item, error) {
	if a == nil {
		return nil, fmt.Errorf("feeds: nil Atom entry")
	}
	updated, err := conversionDate(a.Updated, "Atom updated", true)
	if err != nil {
		return nil, err
	}
	created, err := conversionDate(a.Published, "Atom published", true)
	if err != nil {
		return nil, err
	}
	link, links := conversionAtomLinks(nil, a.Links)
	item := &Item{Title: a.Title, Id: a.Id, XMLBase: a.XMLBase, Language: a.Language, Updated: updated, Created: created, Author: conversionAtomAuthor(a.Author), Link: link, Links: links, Category: a.Category, Categories: conversionAtomCategories(a.Category, a.Categories), Extensions: cloneExtensions(a.Extensions)}
	if a.Summary != nil {
		item.Description, err = conversionAtomText(a.Summary.Content, a.Summary.Type)
		if err != nil {
			return nil, err
		}
	}
	if a.Content != nil {
		item.Content, err = conversionAtomText(a.Content.Content, a.Content.Type)
		if err != nil {
			return nil, err
		}
	}
	for _, link := range links {
		if link.Rel == "enclosure" {
			item.Enclosure = &Enclosure{Url: link.Href, Length: link.Length, Type: link.Type}
			break
		}
	}
	return item, nil
}

// ToFeed converts a modeled Atom feed to the shared model without modifying
// the receiver. Invalid dates return an error with the field/item context.
func (a *AtomFeed) ToFeed() (*Feed, error) {
	if a == nil {
		return nil, fmt.Errorf("feeds: nil Atom feed")
	}
	updated, err := conversionDate(a.Updated, "Atom updated", true)
	if err != nil {
		return nil, err
	}
	link, links := conversionAtomLinks(a.Link, a.Links)
	feed := &Feed{Title: a.Title, Id: a.Id, XMLBase: a.XMLBase, Language: a.Language, Updated: updated, Description: a.Subtitle, Subtitle: a.Subtitle, Copyright: a.Rights, Author: conversionAtomAuthor(a.Author), Link: link, Links: links, Category: a.Category, Categories: conversionAtomCategories(a.Category, a.Categories), Extensions: cloneExtensions(a.Extensions)}
	image := a.Logo
	if image == "" {
		image = a.Icon
	}
	if image != "" {
		feed.Image = &Image{Url: image}
	}
	for i, entry := range a.Entries {
		if entry == nil {
			continue
		}
		item, err := entry.ToItem()
		if err != nil {
			return nil, fmt.Errorf("feeds: Atom entry %d: %w", i, err)
		}
		if entry.LanguageSet && entry.Language == "" && a.Language != "" {
			return nil, fmt.Errorf("feeds: Atom entry %d: empty language reset cannot be represented by Item.Language", i)
		}
		if item.Author == nil && feed.Author != nil {
			copy := *feed.Author
			item.Author = &copy
		}
		feed.Items = append(feed.Items, item)
	}
	return feed, nil
}

// ToItem converts a modeled RSS item to the shared model. SourceInfo.URL takes
// precedence over the legacy Source string. The receiver is not modified.
func (r *RssItem) ToItem() (*Item, error) {
	if r == nil {
		return nil, fmt.Errorf("feeds: nil RSS item")
	}
	created, err := conversionDate(r.PubDate, "RSS pubDate", false)
	if err != nil {
		return nil, err
	}
	item := &Item{Title: r.Title, XMLBase: r.XMLBase, Description: r.Description, Created: created, Author: conversionAuthor(r.Author), Category: r.Category, Categories: categoryNames(r.Category, r.Categories), Extensions: cloneExtensions(mergeITunesExtensions(r.Extensions, r.ITunes.Extensions()))}
	if r.Link != "" {
		item.Link = &Link{Href: r.Link}
	}
	if r.Content != nil {
		item.Content = r.Content.Content
	}
	if r.Guid != nil {
		item.Id = r.Guid.Id
		item.IsPermaLink = r.Guid.IsPermaLink
	}
	if r.Enclosure != nil {
		item.Enclosure = &Enclosure{Url: r.Enclosure.Url, Length: r.Enclosure.Length, Type: r.Enclosure.Type}
	}
	source := r.Source
	if r.SourceInfo != nil {
		source = r.SourceInfo.URL
	}
	if source != "" {
		item.Source = &Link{Href: source}
	}
	return item, nil
}

// ToFeed converts an RSS channel to the shared model. It retains the channel's
// XMLBase; use RssFeedXml.ToFeed to include the outer document's base context.
func (r *RssFeed) ToFeed() (*Feed, error) {
	if r == nil {
		return nil, fmt.Errorf("feeds: nil RSS channel")
	}
	created, err := conversionDate(r.PubDate, "RSS pubDate", false)
	if err != nil {
		return nil, err
	}
	updated, err := conversionDate(r.LastBuildDate, "RSS lastBuildDate", false)
	if err != nil {
		return nil, err
	}
	feed := &Feed{Title: r.Title, XMLBase: r.XMLBase, Language: r.Language, Description: r.Description, Copyright: r.Copyright, Created: created, Updated: updated, Author: conversionAuthor(r.ManagingEditor), Category: r.Category, Categories: categoryNames(r.Category, r.Categories), Extensions: cloneExtensions(mergeITunesExtensions(r.Extensions, r.ITunes.Extensions()))}
	if r.Link != "" {
		feed.Link = &Link{Href: r.Link}
	}
	for _, link := range r.AtomLinks {
		feed.Links = append(feed.Links, &Link{Href: link.Href, Rel: link.Rel, Type: link.Type, Length: link.Length})
	}
	if r.Image != nil {
		feed.Image = &Image{Url: r.Image.Url, Title: r.Image.Title, Link: r.Image.Link, Width: r.Image.Width, Height: r.Image.Height}
	}
	for i, entry := range r.Items {
		if entry == nil {
			continue
		}
		item, err := entry.ToItem()
		if err != nil {
			return nil, fmt.Errorf("feeds: RSS item %d: %w", i, err)
		}
		feed.Items = append(feed.Items, item)
	}
	return feed, nil
}

// ToFeed converts a complete RSS document. Root/channel xml:base declarations
// are combined into the shared feed base; relative resource references stay intact.
func (r *RssFeedXml) ToFeed() (*Feed, error) {
	if r == nil {
		return nil, fmt.Errorf("feeds: nil RSS document")
	}
	feed, err := r.Channel.ToFeed()
	if err != nil {
		return nil, err
	}
	if r.XMLBase != "" {
		root, err := url.Parse(r.XMLBase)
		if err != nil {
			return nil, fmt.Errorf("feeds: RSS document base: %w", err)
		}
		channel, err := url.Parse(feed.XMLBase)
		if err != nil {
			return nil, fmt.Errorf("feeds: RSS channel base: %w", err)
		}
		if feed.XMLBase != "" && !root.IsAbs() && !channel.IsAbs() && channel.Host == "" && !strings.HasPrefix(channel.Path, "/") {
			return nil, fmt.Errorf("feeds: cannot flatten two relative RSS base declarations without a document URL")
		}
		if feed.XMLBase == "" {
			feed.XMLBase = r.XMLBase
		} else {
			feed.XMLBase = root.ResolveReference(channel).String()
		}
	}
	return feed, nil
}
