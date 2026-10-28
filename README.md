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
