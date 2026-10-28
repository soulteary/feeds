## gorilla/feeds
![testing](https://github.com/gorilla/feeds/actions/workflows/test.yml/badge.svg)
[![codecov](https://codecov.io/github/gorilla/feeds/branch/main/graph/badge.svg)](https://codecov.io/github/gorilla/feeds)
[![godoc](https://godoc.org/github.com/gorilla/feeds?status.svg)](https://godoc.org/github.com/gorilla/feeds)
[![sourcegraph](https://sourcegraph.com/github.com/gorilla/feeds/-/badge.svg)](https://sourcegraph.com/github.com/gorilla/feeds?badge)

![Gorilla Logo](https://github.com/gorilla/.github/assets/53367916/d92caabf-98e0-473e-bfbf-ab554ba435e5)

feeds is a web feed generator library for generating RSS, Atom and JSON feeds from Go
applications.

### Goals

 * Provide a simple interface to create both Atom & RSS 2.0 feeds
 * Full support for [Atom][atom], [RSS 2.0][rss], and [JSON Feed Version 1][jsonfeed] spec elements
 * Ability to modify particulars for each spec

[atom]: https://tools.ietf.org/html/rfc4287
[rss]: http://www.rssboard.org/rss-specification
[jsonfeed]: https://jsonfeed.org/version/1.1

### Usage

JSON Feed output uses the standard `banner_image` and `size_in_bytes` keys.
Decoding also accepts the legacy `banner_` and `size` keys; when both names are
present, the standard key takes precedence, including empty, zero, or null values.
As with other scalar fields decoded by `encoding/json`, null leaves an existing
banner image or attachment size unchanged. Attachment durations are encoded in
seconds for both values and pointers. When decoding into a reused attachment,
an explicit zero duration clears the previous value; missing, null, and negative
durations preserve it. Empty feeds always serialize the required `items: []`
array without changing the source feed.

RSS conversion uses `Author.Email` for both channel `managingEditor` and item
`author`, adding `Author.Name` as `email (name)` when present. A missing or
syntactically invalid single email address causes that optional RSS field to be
omitted; it does not prevent feed generation. Display-name mailbox syntax in
`Email` is accepted and normalized, with an explicit `Name` taking precedence.
Item authors do not inherit the channel author. Name-only authors remain
available in Atom and JSON, and callers using `RssFeed` / `RssItem` directly
retain control over their raw contact strings.

Atom conversion uses an explicit `Feed.Id` as the feed identifier, falling back
to `Feed.Link.Href` only when `Id` is empty. Entry `published` is populated from
`Item.Created` and omitted when the creation time is unknown. Entry `updated`
continues to prefer `Item.Updated`, falling back to `Item.Created`. Item IDs,
feed links, and RSS/JSON conversion are unaffected by these Atom mappings.

### Home pages, subscription URLs, and additional links

Keep using `Feed.Link` / `Item.Link` for the primary page link. `Feed.Links` /
`Item.Links` add links without replacing that field. For example:

```go
feed := &feeds.Feed{
    Link: &feeds.Link{Href: "https://example.com/"},
    Links: []*feeds.Link{
        {Href: "https://example.com/atom.xml", Rel: "self", Type: "application/atom+xml"},
        {Href: "https://example.com/rss.xml", Rel: "self", Type: "application/rss+xml"},
        {Href: "https://example.com/feed.json", Rel: "self", Type: "application/feed+json"},
        {Href: "https://example.com/hub", Rel: "hub"},
    },
}
```

- A primary link with an empty `Rel` or `Rel: "alternate"` remains the home page
  (or item permalink). If absent, the first additional alternate/unspecified
  relation with no type or an HTML/XHTML type supplies the page URL.
- A subscription URL must be explicit: `Rel: "self"`. Each format prefers the
  self link matching its media type, then an untyped self link. Self links for
  other formats are not reused. An untyped self URL opts into use by all formats;
  use format-specific types when the endpoints differ. Generated Atom/RSS self
  links without a type receive the current format's media type.
- Atom emits the primary link followed by additional links, including their
  relation, type, and length. RSS keeps its home page in the channel `link` and
  emits self/hub links in the Atom namespace. JSON maps the page and subscription
  URLs to `home_page_url` and `feed_url` separately. A primary `Link` explicitly
  marked self/hub/enclosure is no longer treated as a home page in RSS/JSON.
- Nil links, empty URLs, and duplicate additional links are skipped. Duplicate
  comparison treats an empty relation as alternate and otherwise compares all
  link fields. Different media types or relations are retained. Only one self
  link is selected per feed format; the first equally specific candidate wins.
- The existing low-level `AtomFeed.Link` remains the first link. Its new `Links`
  field contains additional links. XML decoding stores the first link in `Link`
  and the remainder in `Links`; encoding writes both. Explicitly decoded links
  replace the previous link collection; missing links preserve it. `RssFeed`
  exposes namespaced links as `AtomLinks` separately from its page `Link`.
- Existing page-only feeds using a plain `Link{Href: ...}` keep their previous
  serialized output. Configured link type/length attributes are now preserved. No self link
  is guessed from a home page; callers must supply the actual subscription URL.
  Feed IDs retain the existing explicit-ID / primary-link fallback; when neither
  supplies an ID, an additional home page or Atom self URL can supply it.

No field is removed or changes type. Adding exported fields can still require
updates to positional (unkeyed) `Feed`, `Item`, `AtomFeed`, or `RssFeed` struct
literals; keyed struct literals remain compatible. No third-party dependency is
needed for link handling.

### Categories

`Feed` and `Item` expose `Category string` and `Categories []string`. The singular
category is placed first, followed by nonempty, distinct values from the plural
field in input order. Comparison is exact and case-sensitive; nonempty terms
(including Unicode and whitespace) are not rewritten.

```go
feed.Category = "Technology"
feed.Categories = []string{"Go", "RSS"}
feed.Items = []*feeds.Item{{
    Id: "urn:post:1",
    Category: "Go",
    Categories: []string{"Tutorial", "Go"},
}}
```

RSS emits repeated `category` elements on channels/items. Atom emits repeated
`category` elements with a `term` attribute on feeds/entries. JSON Feed maps
**item** categories to `tags`; the specification has no feed-level category
field, so feed categories are not emitted there or inherited by items.

Existing `RssFeed.Category`, `RssItem.Category`, `AtomFeed.Category`, and
`AtomEntry.Category` fields retain their string types. Their new `Categories`
fields hold the complete collection (`[]string` for RSS, `[]AtomCategory` for
Atom). XML decoding retains all categories and mirrors the first in `Category`.
On output, `Category` is merged first without duplicating the collection's
matching category. Missing category elements preserve existing values on reused
objects; explicit elements replace the old collection.

Use `AtomCategory{Term: "Go", Scheme: "https://example.com/topics", Label: "Go"}`
when scheme or label metadata is needed. Categories with the same term but
different metadata are retained. If legacy `Category` matches a rich category,
its metadata is preserved rather than adding a duplicate bare term. Legacy Atom
text-only category input is accepted, but output always uses the standard `term`
attribute, including when only the existing `Category` field is set. RSS's
existing single-category output is unchanged. No-category output is unchanged
for all formats, and formatting does not mutate source categories.

The new fields are additive; positional struct literals for the affected public
structs may need updates. Keyed literals using the existing fields remain valid.

### Languages

Set `Feed.Language` to the feed's primary language and `Item.Language` for an
explicit item override, using language tags such as `en-US`, `zh-Hant-TW`, or
`sr-Latn-RS`.

- RSS maps `Feed.Language` to the existing channel `language` element. RSS 2.0
  has no core item-level language element, so item overrides are not emitted
  there; no nonstandard item `language` element is invented.
- Atom maps feed and entry languages to `xml:lang` in the reserved XML namespace.
  When an entry language is empty it is omitted, so XML language inheritance
  supplies the feed context. The Go entry field remains empty, not a copied
  feed value. A standalone entry has only its own explicitly set language.
- JSON Feed maps the feed and item fields to their respective `language` keys;
  an omitted item value leaves the feed's primary language as the default.

The existing `RssFeed.Language` and `JSONFeed.Language` remain strings. New
`AtomFeed.Language`, `AtomEntry.Language`, and `JSONItem.Language` fields support
low-level encoding/decoding as well. Empty strings are omitted; the string API
models unspecified/inherited languages, not an explicit empty `xml:lang` reset.
Values are preserved exactly, without forced validation or normalization. Callers
remain responsible for supplying valid language tags. Source objects are not
mutated, and feeds without language values keep their previous output. No new
dependency is required. As with other additive fields, positional struct literals
for affected public types may need updates; keyed literals remain compatible.

### Custom XML extensions

`Feed.Extensions` and `Item.Extensions` append foreign XML child elements to both
RSS and Atom output. The low-level `RssFeed`, `RssItem`, `AtomFeed`, and `AtomEntry`
models expose the same field for format-specific extensions. XML extensions are
not mapped into JSON Feed.

```go
feed.Extensions = []feeds.Extension{{
    XMLName: xml.Name{Space: "https://example.com/feed-extension", Local: "rating"},
    Attrs: []xml.Attr{{Name: xml.Name{Local: "scale"}, Value: "5"}},
    Content: []feeds.ExtensionContent{{Text: "4 & rising"}},
}}
```

Import `encoding/xml` for `xml.Name` and `xml.Attr`. `Space` is the namespace URI;
`Local` is an unprefixed local name. Top-level extensions require a foreign,
nonempty namespace; native fields cannot be replaced through extensions. Use
`RssFeed.AtomLinks` or the generic link model for RSS Atom links already modeled
by the library. Attributes follow the same expanded-name convention; namespace
declarations and duplicate attribute names are rejected. The encoder supplies
namespace declarations and escapes ordinary text/attribute values.

Each ordered `ExtensionContent` item contains text, a child `*Extension`, or a
comment. Nested children may use other namespaces or the empty namespace; an
explicit default-namespace reset preserves unqualified children. Whitespace and
mixed text/child order are retained even when the surrounding feed is indented.
Only encoder-generated XML is used internally; content text is never raw XML.

Decoding captures foreign direct children before they can be confused with
native fields sharing local names. Native descendants continue through their
existing streaming decoders. Explicit foreign collections replace old ones on a
reused object; absent collections preserve existing values. The consume fixtures
now retain their Dublin Core creators instead of discarding them; the Atom
fixture also declares its previously undeclared `dc` prefix.

Roundtrips preserve expanded element/attribute names and ordered content, not
byte spelling, prefix selection, namespace-declaration placement, or CDATA
boundaries. Namespace declarations are not retained, and QName-valued strings
(such as `xsi:type="prefix:Name"`) are not resolved or rewritten; payloads relying
on those lexical prefix bindings need a schema-aware implementation outside this
API. Processing instructions/directives inside an extension return an error.
The payload tree must be acyclic. Feed/entry wrapper attributes and extensions
on the outer RSS document are outside this child-element API.

No existing field is removed or changes type, no dependency is added, and
extension-free output stays unchanged. Additive fields may require updates to
positional struct literals; keyed literals remain compatible.

### Complete example

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/gorilla/feeds"
)

func main() {
    now := time.Now()
    feed := &feeds.Feed{
        Title:       "jmoiron.net blog",
        Link:        &feeds.Link{Href: "http://jmoiron.net/blog"},
        Description: "discussion about tech, footie, photos",
        Author:      &feeds.Author{Name: "Jason Moiron", Email: "jmoiron@jmoiron.net"},
        Created:     now,
    }

    feed.Items = []*feeds.Item{
        &feeds.Item{
            Title:       "Limiting Concurrency in Go",
            Link:        &feeds.Link{Href: "http://jmoiron.net/blog/limiting-concurrency-in-go/"},
            Description: "A discussion on controlled parallelism in golang",
            Author:      &feeds.Author{Name: "Jason Moiron", Email: "jmoiron@jmoiron.net"},
            Created:     now,
        },
        &feeds.Item{
            Title:       "Logic-less Template Redux",
            Link:        &feeds.Link{Href: "http://jmoiron.net/blog/logicless-template-redux/"},
            Description: "More thoughts on logicless templates",
            Created:     now,
        },
        &feeds.Item{
            Title:       "Idiomatic Code Reuse in Go",
            Link:        &feeds.Link{Href: "http://jmoiron.net/blog/idiomatic-code-reuse-in-go/"},
            Description: "How to use interfaces <em>effectively</em>",
            Created:     now,
        },
    }

    atom, err := feed.ToAtom()
    if err != nil {
        log.Fatal(err)
    }

    rss, err := feed.ToRss()
    if err != nil {
        log.Fatal(err)
    }

    json, err := feed.ToJSON()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(atom, "\n", rss, "\n", json)
}
```

Outputs:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<feed xmlns="http://www.w3.org/2005/Atom">
  <title>jmoiron.net blog</title>
  <link href="http://jmoiron.net/blog"></link>
  <id>http://jmoiron.net/blog</id>
  <updated>2013-01-16T03:26:01-05:00</updated>
  <summary>discussion about tech, footie, photos</summary>
  <entry>
    <title>Limiting Concurrency in Go</title>
    <link href="http://jmoiron.net/blog/limiting-concurrency-in-go/"></link>
    <updated>2013-01-16T03:26:01-05:00</updated>
    <id>tag:jmoiron.net,2013-01-16:/blog/limiting-concurrency-in-go/</id>
    <summary type="html">A discussion on controlled parallelism in golang</summary>
    <author>
      <name>Jason Moiron</name>
      <email>jmoiron@jmoiron.net</email>
    </author>
  </entry>
  <entry>
    <title>Logic-less Template Redux</title>
    <link href="http://jmoiron.net/blog/logicless-template-redux/"></link>
    <updated>2013-01-16T03:26:01-05:00</updated>
    <id>tag:jmoiron.net,2013-01-16:/blog/logicless-template-redux/</id>
    <summary type="html">More thoughts on logicless templates</summary>
    <author></author>
  </entry>
  <entry>
    <title>Idiomatic Code Reuse in Go</title>
    <link href="http://jmoiron.net/blog/idiomatic-code-reuse-in-go/"></link>
    <updated>2013-01-16T03:26:01-05:00</updated>
    <id>tag:jmoiron.net,2013-01-16:/blog/idiomatic-code-reuse-in-go/</id>
    <summary type="html">How to use interfaces &lt;em&gt;effectively&lt;/em&gt;</summary>
    <author></author>
  </entry>
</feed>

<?xml version="1.0" encoding="UTF-8"?>
<rss version="2.0">
  <channel>
    <title>jmoiron.net blog</title>
    <link>http://jmoiron.net/blog</link>
    <description>discussion about tech, footie, photos</description>
    <managingEditor>jmoiron@jmoiron.net (Jason Moiron)</managingEditor>
    <pubDate>2013-01-16T03:22:24-05:00</pubDate>
    <item>
      <title>Limiting Concurrency in Go</title>
      <link>http://jmoiron.net/blog/limiting-concurrency-in-go/</link>
      <description>A discussion on controlled parallelism in golang</description>
      <pubDate>2013-01-16T03:22:24-05:00</pubDate>
    </item>
    <item>
      <title>Logic-less Template Redux</title>
      <link>http://jmoiron.net/blog/logicless-template-redux/</link>
      <description>More thoughts on logicless templates</description>
      <pubDate>2013-01-16T03:22:24-05:00</pubDate>
    </item>
    <item>
      <title>Idiomatic Code Reuse in Go</title>
      <link>http://jmoiron.net/blog/idiomatic-code-reuse-in-go/</link>
      <description>How to use interfaces &lt;em&gt;effectively&lt;/em&gt;</description>
      <pubDate>2013-01-16T03:22:24-05:00</pubDate>
    </item>
  </channel>
</rss>

{
  "version": "https://jsonfeed.org/version/1.1",
  "title": "jmoiron.net blog",
  "home_page_url": "http://jmoiron.net/blog",
  "description": "discussion about tech, footie, photos",
  "author": {
    "name": "Jason Moiron"
  },
  "authors": [
    {
      "name": "Jason Moiron"
    }
  ],
  "items": [
    {
      "id": "",
      "url": "http://jmoiron.net/blog/limiting-concurrency-in-go/",
      "title": "Limiting Concurrency in Go",
      "summary": "A discussion on controlled parallelism in golang",
      "date_published": "2013-01-16T03:22:24.530817846-05:00",
      "author": {
        "name": "Jason Moiron"
      },
      "authors": [
        {
          "name": "Jason Moiron"
        }
      ]
    },
    {
      "id": "",
      "url": "http://jmoiron.net/blog/logicless-template-redux/",
      "title": "Logic-less Template Redux",
      "summary": "More thoughts on logicless templates",
      "date_published": "2013-01-16T03:22:24.530817846-05:00"
    },
    {
      "id": "",
      "url": "http://jmoiron.net/blog/idiomatic-code-reuse-in-go/",
      "title": "Idiomatic Code Reuse in Go",
      "summary": "How to use interfaces \u003cem\u003eeffectively\u003c/em\u003e",
      "date_published": "2013-01-16T03:22:24.530817846-05:00"
    }
  ]
}
```
