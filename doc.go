/*
Package feeds generates RSS 2.0, Atom, and JSON Feed documents.

# Installing

	go get github.com/soulteary/feeds

Feed and Item provide the shared data model. RssFeed, AtomFeed, and JSONFeed
expose format-specific fields for callers that need more control.

# Examples

Create a Feed and some Items in that feed using the generic interfaces:

	import (
		"time"
		. "github.com/soulteary/feeds"
	)

	now := time.Now()

	feed := &Feed{
		Title:       "jmoiron.net blog",
		Link:        &Link{Href: "http://jmoiron.net/blog"},
		Description: "discussion about tech, footie, photos",
		Author:      &Author{Name: "Jason Moiron", Email: "jmoiron@jmoiron.net"},
		Created:     now,
		Copyright:   "This work is copyright © Benjamin Button",
	}

	feed.Items = []*Item{
		&Item{
			Title:       "Limiting Concurrency in Go",
			Link:        &Link{Href: "http://jmoiron.net/blog/limiting-concurrency-in-go/"},
			Description: "A discussion on controlled parallelism in golang",
			Author:      &Author{Name: "Jason Moiron", Email: "jmoiron@jmoiron.net"},
			Created:     now,
		},
		&Item{
			Title:       "Logic-less Template Redux",
			Link:        &Link{Href: "http://jmoiron.net/blog/logicless-template-redux/"},
			Description: "More thoughts on logicless templates",
			Created:     now,
		},
		&Item{
			Title:       "Idiomatic Code Reuse in Go",
			Link:        &Link{Href: "http://jmoiron.net/blog/idiomatic-code-reuse-in-go/"},
			Description: "How to use interfaces <em>effectively</em>",
			Created:     now,
		},
	}

From here, you can output Atom, RSS, or JSON Feed versions of this feed easily

	atom, err := feed.ToAtom()
	rss, err := feed.ToRss()
	json, err := feed.ToJSON()

You can also get access to the underlying objects that feeds uses to export its XML

	atomFeed := (&Atom{Feed: feed}).AtomFeed()
	rssFeed := (&Rss{Feed: feed}).RssFeed()
	jsonFeed := (&JSON{Feed: feed}).JSONFeed()

From here, you can modify or add each syndication's specific fields before outputting

	atomFeed.Subtitle = "plays the blues"
	atom, err := ToXML(atomFeed)
	rssFeed.Generator = "soulteary/feeds v1.3.0 (github.com/soulteary/feeds)"
	rss, err := ToXML(rssFeed)
	jsonFeed.NextUrl = "https://www.example.com/feed.json?page=2"
	json, err := jsonFeed.ToJSON()
*/
package feeds
