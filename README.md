# soulteary/feeds

Independently maintained fork of [gorilla/feeds](https://github.com/gorilla/feeds).
Starting with v1.3.0, this fork publishes the Go module `github.com/soulteary/feeds`.
Go 1.20 remains the minimum supported version.

```sh
go get github.com/soulteary/feeds@v1.3.0
```

To adopt this fork, change imports from `github.com/gorilla/feeds` to
`github.com/soulteary/feeds`. Existing Go API names remain available; see the
conversion and wire-format notes below for intentional behavior corrections.

[![Go Test Coverage](./.github/coverage.svg)](./.github/go-test-report.md)
[![Go Report Card](./.github/goreportcard.svg)](./.github/goreportcard-report.md)
[![Compatibility](https://github.com/soulteary/feeds/actions/workflows/test.yml/badge.svg)](https://github.com/soulteary/feeds/actions/workflows/test.yml)
[![Security](https://github.com/soulteary/feeds/actions/workflows/security.yml/badge.svg)](https://github.com/soulteary/feeds/actions/workflows/security.yml)

feeds is a web feed generator library for generating RSS, Atom and JSON feeds from Go
applications.

### Tests, quality reports and performance

Following [soulteary/cascadia](https://github.com/soulteary/cascadia), the Go quality
workflow uses [go-test-report-action](https://github.com/soulteary/go-test-report-action)
to run race-enabled tests and enforce 100% total and per-package statement
coverage, and [goreportcard-action](https://github.com/soulteary/goreportcard-action)
to generate the local quality badge/report. Reports and raw test output are
uploaded as artifacts. PR checks have read-only permissions; successful main
runs publish stable reports only if main still matches the tested revision.
Initial checked-in reports were generated locally; subsequent main runs refresh
them in CI. A 100% statement result does not prove all branches or inputs correct.

The reporting/security toolchain is pinned in `.github/go-version`; it is separate
from the module's Go 1.20 minimum. Compatibility tests cover Go 1.20 on Linux and
the reporting version on Linux, macOS and Windows. Security checks use GoSec and
govulncheck. A manual benchmark workflow records timings and allocations without
using noisy shared-runner latency as a pass/fail threshold.

See [testing and CI](docs/testing.md) and [performance measurements](docs/performance.md)
for fixtures, raw results, compatibility checks, limitations and reproduction.
The XML extension optimization avoids per-name decoders for ordinary ASCII names
and per-leaf encoders for simple text, preserving the fallback and escaping rules.
Existing public APIs, dependencies and wire formats are unchanged.

### Goals

 * Provide a simple interface to create both Atom & RSS 2.0 feeds
 * Support common [Atom][atom], [RSS 2.0][rss], and [JSON Feed 1.1][jsonfeed] metadata with low-level format controls
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
low-level encoding/decoding as well. Empty strings are normally omitted. Low-level `AtomFeed.LanguageSet` and
`AtomEntry.LanguageSet` distinguish explicitly present languages: set the flag
with an empty `Language` to emit `xml:lang=""` and clear inherited language.
XML decoding retains this presence flag; it does not copy inherited values.
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

### iTunes / Apple Podcasts RSS metadata

Set optional `Feed.ITunes` / `Item.ITunes` pointers, or their low-level
`RssFeed.ITunes` / `RssItem.ITunes` equivalents. These fields affect RSS only and
convert to the existing XML extension model; Atom and JSON remain unchanged.

```go
clean := false
feed.Language = "en-US"
feed.ITunes = &feeds.ITunesFeed{
    Author: "Example Network",
    Image: &feeds.ITunesImage{Href: "https://example.com/show.jpg"},
    Explicit: &clean,
    Type: feeds.ITunesFeedTypeSerial,
    Categories: []feeds.ITunesCategory{{
        Text: "Society & Culture",
        Categories: []feeds.ITunesCategory{{Text: "Documentary"}},
    }},
}
feed.Items[0].ITunes = &feeds.ITunesItem{
    Duration: "90",
    Episode: 1,
    Season: 1,
    EpisodeType: feeds.ITunesEpisodeTypeFull,
    Explicit: &clean,
}
```

Show metadata supports title, author, image, nested categories, explicit rating,
show type, owner, new-feed-url, block, complete, and applepodcastsverify. Episode
metadata supports title, author, image, explicit rating, duration, episode/season,
episode type, and block. Legacy subtitle, summary, keywords, order, and
isClosedCaptioned are available for readers that still consume them.

`Explicit` is a `*bool`: nil omits the tag, while false emits `false`. Block and
complete emit `Yes` only when true. `IsClosedCaptioned` uses optional Yes/No.
Episode/season/order zero values are omitted. `Duration` is a string: use integer
seconds (including explicit `"0"`) or a supported clock format such as
`"00:01:30"`, not Go duration strings like `"1m30s"`. Constants use lowercase
`episodic`/`serial` and `full`/`trailer`/`bonus`.

The RSS wrapper includes `xmlns:itunes` only when typed or raw iTunes elements
are present, including item-only metadata. Generated elements use the conventional
`itunes` prefix with the exact `ITunesNamespace` URI. The extension model's new
optional `Prefix` hint adds scoped declarations; it does not change expanded-name
semantics and is not retained on decoding. All values still use normal XML
escaping. Custom attributes retain independent namespaces even when their natural
prefix would clash with the element hint.

`ITunesFeed.Extensions()` / `ITunesItem.Extensions()` also expose the generated
extension elements. Typed fields replace raw extensions with matching expanded
names only when the typed field emits an element; unknown extensions and omitted
typed fields retain raw values. Parsing retains iTunes data in `Extensions`
rather than populating typed structs. On a reused RSS object, newly decoded
iTunes extensions clear any stale typed pointer so it cannot override the input.
Subsequent RSS encoding preserves the decoded metadata and root namespace.

Empty pointers/metadata do not change the default output or add dependencies.
Existing fields retain their names/types; additive fields can affect positional
struct literals. No metadata is inferred from existing author/image/category
fields or inherited from the show into an episode. Callers remain responsible
for category spelling, supported values, positive episode/season numbers, stable
GUIDs, valid enclosures, artwork/hosting requirements, and the complete podcast
feed. This is serialization support, not an Apple submission validator. See
[Apple's RSS guide](https://help.apple.com/itc/podcasts_connect/en.lproj/itcb54353390.html)
and [technical requirements](https://podcasters.apple.com/support/823-podcast-requirements).

### XML base URIs

Set `Feed.XMLBase` and `Item.XMLBase` to declare optional XML base URI references:

```go
feed.XMLBase = "https://example.com/show/"
feed.Items[0].XMLBase = "episodes/"
```

RSS places the feed value on `channel` and the item value on `item`; Atom places
these on `feed` and `entry`. Attributes use `xml:base` in the reserved
`http://www.w3.org/XML/1998/namespace` namespace, not an unqualified `base`
attribute or a custom namespace. Empty values are omitted. Descendants with no
explicit base use the inherited XML context; the Go fields remain unset rather
than receiving copied defaults. Empty `xml:base` is not a document-base reset.

The low-level `AtomFeed`, `AtomEntry`, `RssFeed`, `RssItem`, and `RssFeedXml`
models also expose `XMLBase`. The RSS wrapper field declares a document-level
base, independently of channel/item values. Preserve that wrapper and use
`ToXML(&rssDocument)` / `WriteXML(&rssDocument, writer)` when retaining root
context after decoding; encoding only `rssDocument.Channel` intentionally
rewraps just the channel and cannot retain a discarded root context. Modeled
content/iTunes namespace declarations are retained by wrapper decoding.

Absolute and relative values, Unicode, and URI-reference spelling are preserved
with normal XML escaping; links, IDs, enclosure URLs, and nested content are not
resolved, normalized, or rewritten. Reading fields does not compute an effective
base URI. Existing extension attributes can express `xml:base` on nested custom
elements. URI validation and resolution remain caller/consumer responsibilities;
see [XML Base](https://www.w3.org/TR/xmlbase/).

JSON Feed has no `xml:base` counterpart, so the new fields do not affect JSON
output. Unset bases keep existing XML output unchanged, with no new dependencies
or existing-field type changes. As with other additive fields, positional struct
literals for affected types may need updating.

### Feed images

The existing `Feed.Image` maps its `Url` to Atom `logo` and JSON Feed `icon`.
Nil images or empty URLs omit those optional fields. RSS retains its existing
image URL, title, link, width, and height mapping. Metadata other than the URL is
not mapped to Atom/JSON because their feed image fields contain only a reference.

The generic image does not also populate Atom `icon` or JSON `favicon`; these
small-image roles may require a different resource. Set low-level `AtomFeed.Icon`
and `JSONFeed.Favicon` independently when needed. Existing low-level logo/icon
values remain caller-controlled. Feed images are not inherited by items, and the
existing JSON item image mapping from an image enclosure is unchanged.

Image URL references are passed through exactly with XML/JSON escaping. No URL
resolution, file access, downloading, resizing, dimension-based selection, or
image validation is performed. Callers provide suitable assets for each format;
relative XML references can use the existing `XMLBase` context. The `Image` type
and all existing fields remain unchanged. Feeds without images retain their
previous output, and no dependency is added.

### XML roundtrip and empty-item behavior

Atom entry links are decoded into `AtomEntry.Links`, including alternate and
media enclosure links. Explicit incoming link elements replace an existing entry
link collection; absent links preserve the current collection.

RSS `content:encoded` is recognized by its namespace URI rather than a particular
prefix and decoded into `RssItem.Content`. It is no longer also retained as a raw
extension. If a caller supplies both typed Content and a matching raw extension,
typed Content wins, so the body is emitted once without altering source slices.
Existing RSS content output keeps its `content:encoded`/CDATA spelling.

`RssItem.Source` remains a string. On legacy output it supplies both the source
text and its `url` attribute. To distinguish a source feed's URL from its name,
set `RssItem.SourceInfo = &RssSource{URL: "https://example.com/feed", Name: "Name"}`.
The structured value takes precedence; decoding retains the URL/name there and
mirrors the source text in `Source`. Missing source elements preserve existing
values, while explicit elements replace them.

Generic Atom/RSS/JSON conversion skips nil entries in `Feed.Items`, preserves
the order of the remaining entries, and leaves the original slice untouched.
A feed containing only nil entries has the same output as an empty feed.

These fixes retain existing public field types and function signatures. The new
low-level presence/source fields can require updates to positional struct literals.
No third-party dependency is introduced.

### Complete example

```go
package main

import (
    "fmt"
    "log"
    "time"
    "github.com/soulteary/feeds"
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

### Converting decoded RSS and Atom to the shared model

Use `xml.Unmarshal` with `AtomFeed` or `RssFeedXml`, then call `ToFeed()`:

```go
var document feeds.RssFeedXml
if err := xml.Unmarshal(data, &document); err != nil {
    return err
}
feed, err := document.ToFeed()
if err != nil {
    return err
}
// The shared feed can now be edited or serialized with ToAtom/ToRss/ToJSON.
_ = feed
```

`(*AtomFeed).ToFeed`, `(*RssFeed).ToFeed` and `(*RssFeedXml).ToFeed`
return `(*Feed, error)`. `(*AtomEntry).ToItem` and `(*RssItem).ToItem`
return `(*Item, error)`. These are additive methods; existing fields and APIs
are unchanged. Nil receivers/missing RSS channels return errors, and nil entries
are skipped. Results own their mutable metadata, links and extension trees;
converting or editing a result does not mutate the input.

Shared titles, IDs, dates, authors, links, categories, images, language,
XML bases, content and foreign extensions are mapped where the source format
provides them. Atom page links become the primary Link; self, hub and enclosure
relations stay in Links. The first enclosure also populates Enclosure. Feed
logo is preferred over icon. Missing Atom entry authors inherit a copied feed
author. Nonempty Atom entry IDs set Item.IsPermaLink to "false": an Atom ID,
including an HTTP identifier, does not establish an article permalink. RSS output
therefore emits an explicit isPermaLink="false" on the converted GUID. Callers
that know an ID is a permalink can explicitly set the shared flag to "true".
An empty Atom ID keeps the shared flag empty and does not invent an RSS GUID.

RSS managingEditor/author mailboxes are parsed using net/mail; legacy
non-mailbox strings remain Name. RSS SourceInfo.URL wins over legacy Source,
including an explicitly empty URL. RSS pubDate maps to Created, lastBuildDate
to Updated; missing dates remain zero. Atom updated and published remain distinct.

Invalid dates return errors identifying their field and entry index. Atom accepts
RFC3339 timestamps including fractions. RSS accepts the RFC1123/RFC822 layouts
with numeric offsets or recognized UT/UTC/GMT and North American zone names;
unknown named zones are rejected rather than silently treated as UTC. Atom text
and text/plain content/summary are HTML-escaped for the shared HTML fields;
html and text/html remain unchanged. Other content types (including XHTML and
binary types) return errors because the current low-level/shared models cannot
represent them faithfully. An explicit empty entry language reset under a
nonempty feed language also returns an error rather than becoming inheritance.

Complete RSS conversion combines root and channel xml:base, leaving resource
references untouched. Two relative declarations requiring the original document
URL cannot safely be flattened and return an error; callers can first supply an
absolute document base. Channel-only ToFeed retains its own base without assuming
an outer context. RSS podcast metadata is retained as copied XML extensions,
using the existing typed-over-raw merge rules.

This is conversion of the fields modeled by this library, not a lossless XML
editor. Format-only metadata without a shared destination (such as Atom category
scheme/label, author URI, contributor, separate icon when logo is present, entry
rights/source, and RSS source display name, generator, comments, TTL and textInput)
is not retained in Feed. Category terms remain, with stable deduplication. The
existing low-level models also do not capture every XML construct or attribute;
keep those models/original XML when exact preservation is required. Existing
serializers retain their documented timestamp precision and per-format limits.

RSS content:encoded now declares its own content-module namespace. This preserves
its meaning even when input documents bind the content prefix to another URI,
or when content is marshaled without an RSS root. The extra local declaration
changes XML spelling but not the normal element's expanded name or content.
