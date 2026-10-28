package feeds

import (
	"encoding/json"
	"strings"
	"time"
)

const jsonFeedVersion = "https://jsonfeed.org/version/1.1"

// JSONAuthor represents the author of the feed or of an individual item
// in the feed
type JSONAuthor struct {
	Name   string `json:"name,omitempty"`
	Url    string `json:"url,omitempty"`
	Avatar string `json:"avatar,omitempty"`
}

// JSONAttachment represents a related resource. Podcasts, for instance, would
// include an attachment that’s an audio or video file.
type JSONAttachment struct {
	Url      string        `json:"url,omitempty"`
	MIMEType string        `json:"mime_type,omitempty"`
	Title    string        `json:"title,omitempty"`
	Size     int32         `json:"size_in_bytes,omitempty"`
	Duration time.Duration `json:"duration_in_seconds,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
// The Duration field is marshaled in seconds, all other fields are marshaled
// based upon the definitions in struct tags.
func (a JSONAttachment) MarshalJSON() ([]byte, error) {
	type EmbeddedJSONAttachment JSONAttachment
	return json.Marshal(&struct {
		Duration float64 `json:"duration_in_seconds,omitempty"`
		*EmbeddedJSONAttachment
	}{
		EmbeddedJSONAttachment: (*EmbeddedJSONAttachment)(&a),
		Duration:               a.Duration.Seconds(),
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// The Duration field is expected to be in seconds, all other field types
// match the struct definition. The legacy size key is accepted on input;
// size_in_bytes takes precedence when both are present.
func (a *JSONAttachment) UnmarshalJSON(data []byte) error {
	type EmbeddedJSONAttachment JSONAttachment
	var raw struct {
		Duration   *float64        `json:"duration_in_seconds,omitempty"`
		Size       json.RawMessage `json:"size_in_bytes"`
		LegacySize json.RawMessage `json:"size"`
		*EmbeddedJSONAttachment
	}
	raw.EmbeddedJSONAttachment = (*EmbeddedJSONAttachment)(a)

	err := json.Unmarshal(data, &raw)
	if err != nil {
		return err
	}

	// Prefer the standard key even when it explicitly contains zero or null.
	size := raw.Size
	if len(size) == 0 {
		size = raw.LegacySize
	}
	if len(size) != 0 {
		if err := json.Unmarshal(size, &a.Size); err != nil {
			return err
		}
	}
	if raw.Duration != nil && *raw.Duration >= 0 {
		nsec := int64(*raw.Duration * float64(time.Second))
		raw.EmbeddedJSONAttachment.Duration = time.Duration(nsec)
	}

	return nil
}

// JSONItem represents a single entry/post for the feed.
type JSONItem struct {
	Id            string           `json:"id"`
	Url           string           `json:"url,omitempty"`
	ExternalUrl   string           `json:"external_url,omitempty"`
	Title         string           `json:"title,omitempty"`
	ContentHTML   string           `json:"content_html,omitempty"`
	ContentText   string           `json:"content_text,omitempty"`
	Summary       string           `json:"summary,omitempty"`
	Image         string           `json:"image,omitempty"`
	BannerImage   string           `json:"banner_image,omitempty"`
	PublishedDate *time.Time       `json:"date_published,omitempty"`
	ModifiedDate  *time.Time       `json:"date_modified,omitempty"`
	Author        *JSONAuthor      `json:"author,omitempty"` // deprecated in JSON Feed v1.1, keeping for backwards compatibility
	Authors       []*JSONAuthor    `json:"authors,omitempty"`
	Tags          []string         `json:"tags,omitempty"`
	Attachments   []JSONAttachment `json:"attachments,omitempty"`
}

// UnmarshalJSON accepts the legacy banner_ key while preferring banner_image.
// Marshaling always uses the standard JSON Feed field name.
func (i *JSONItem) UnmarshalJSON(data []byte) error {
	type item JSONItem
	raw := struct {
		*item
		BannerImage       json.RawMessage `json:"banner_image"`
		LegacyBannerImage json.RawMessage `json:"banner_"`
	}{item: (*item)(i)}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	banner := raw.BannerImage
	if len(banner) == 0 {
		banner = raw.LegacyBannerImage
	}
	if len(banner) != 0 {
		return json.Unmarshal(banner, &i.BannerImage)
	}
	return nil
}

// JSONHub describes an endpoint that can be used to subscribe to real-time
// notifications from the publisher of this feed.
type JSONHub struct {
	Type string `json:"type"`
	Url  string `json:"url"`
}

// JSONFeed represents a syndication feed in the JSON Feed Version 1 format.
// Matching the specification found here: https://jsonfeed.org/version/1.
type JSONFeed struct {
	Version     string        `json:"version"`
	Title       string        `json:"title"`
	Language    string        `json:"language,omitempty"`
	HomePageUrl string        `json:"home_page_url,omitempty"`
	FeedUrl     string        `json:"feed_url,omitempty"`
	Description string        `json:"description,omitempty"`
	UserComment string        `json:"user_comment,omitempty"`
	NextUrl     string        `json:"next_url,omitempty"`
	Icon        string        `json:"icon,omitempty"`
	Favicon     string        `json:"favicon,omitempty"`
	Author      *JSONAuthor   `json:"author,omitempty"` // deprecated in JSON Feed v1.1, keeping for backwards compatibility
	Authors     []*JSONAuthor `json:"authors,omitempty"`
	Expired     *bool         `json:"expired,omitempty"`
	Hubs        []*JSONHub    `json:"hubs,omitempty"`
	Items       []*JSONItem   `json:"items"`
}

// MarshalJSON includes the required items array without changing the source.
func (f JSONFeed) MarshalJSON() ([]byte, error) {
	type feed JSONFeed
	if f.Items == nil {
		f.Items = []*JSONItem{}
	}
	return json.Marshal(feed(f))
}

// JSON is used to convert a generic Feed to a JSONFeed.
type JSON struct {
	*Feed
}

// ToJSON encodes f into a JSON string. Returns an error if marshalling fails.
func (f *JSON) ToJSON() (string, error) {
	return f.JSONFeed().ToJSON()
}

// ToJSON encodes f into a JSON string. Returns an error if marshalling fails.
func (f *JSONFeed) ToJSON() (string, error) {
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// JSONFeed creates a new JSONFeed with a generic Feed struct's data.
func (f *JSON) JSONFeed() *JSONFeed {
	feed := &JSONFeed{
		Version:     jsonFeedVersion,
		Title:       f.Title,
		Description: f.Description,
	}

	if home := homeLink(f.Link, f.Links); home != nil {
		feed.HomePageUrl = home.Href
	}
	if self := selfLink(f.Link, f.Links, "application/feed+json"); self != nil {
		feed.FeedUrl = self.Href
	}
	if f.Author != nil {
		author := &JSONAuthor{
			Name: f.Author.Name,
		}
		feed.Author = author
		feed.Authors = []*JSONAuthor{author}
	}
	for _, e := range f.Items {
		feed.Items = append(feed.Items, newJSONItem(e))
	}
	return feed
}

func newJSONItem(i *Item) *JSONItem {
	item := &JSONItem{
		Id:      i.Id,
		Title:   i.Title,
		Summary: i.Description,

		ContentHTML: i.Content,
		Tags:        categoryNames(i.Category, i.Categories),
	}

	if home := homeLink(i.Link, i.Links); home != nil {
		item.Url = home.Href
	}
	if i.Source != nil {
		item.ExternalUrl = i.Source.Href
	}
	if i.Author != nil {
		author := &JSONAuthor{
			Name: i.Author.Name,
		}
		item.Author = author
		item.Authors = []*JSONAuthor{author}
	}
	if !i.Created.IsZero() {
		item.PublishedDate = &i.Created
	}
	if !i.Updated.IsZero() {
		item.ModifiedDate = &i.Updated
	}
	if i.Enclosure != nil && strings.HasPrefix(i.Enclosure.Type, "image/") {
		item.Image = i.Enclosure.Url
	}

	return item
}
