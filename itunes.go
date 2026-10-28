package feeds

import (
	"encoding/xml"
	"strconv"
)

// ITunesNamespace is the Apple Podcasts RSS namespace (including the www).
const ITunesNamespace = "http://www.itunes.com/dtds/podcast-1.0.dtd"

type ITunesFeedType string

const (
	ITunesFeedTypeEpisodic ITunesFeedType = "episodic"
	ITunesFeedTypeSerial   ITunesFeedType = "serial"
)

type ITunesEpisodeType string

const (
	ITunesEpisodeTypeFull    ITunesEpisodeType = "full"
	ITunesEpisodeTypeTrailer ITunesEpisodeType = "trailer"
	ITunesEpisodeTypeBonus   ITunesEpisodeType = "bonus"
)

// ITunesFeed holds optional RSS-only show metadata. Explicit nil is unspecified.
// Legacy fields are retained for readers that still use them.
type ITunesFeed struct {
	Title, Author, Subtitle, Summary, Keywords string
	Image                                      *ITunesImage
	Categories                                 []ITunesCategory
	Explicit                                   *bool
	Type                                       ITunesFeedType
	Owner                                      *ITunesOwner
	NewFeedURL                                 string
	Block, Complete                            bool
	ApplePodcastsVerify                        string
}

type ITunesImage struct{ Href string }
type ITunesOwner struct{ Name, Email string }

// ITunesCategory uses Apple's category names and optional nested subcategories.
type ITunesCategory struct {
	Text       string
	Categories []ITunesCategory
}

// ITunesItem holds optional RSS-only episode metadata. Duration accepts Apple
// duration strings, preferably integer seconds (e.g. "90"), or "HH:MM:SS".
// Episode/Season/Order zero values are omitted. Values are caller-validated.
type ITunesItem struct {
	Title, Author, Subtitle, Summary, Keywords string
	Image                                      *ITunesImage
	Explicit                                   *bool
	Duration                                   string
	Episode, Season                            int
	EpisodeType                                ITunesEpisodeType
	Block                                      bool
	IsClosedCaptioned                          *bool
	Order                                      int
}

func itunesText(name, value string) Extension {
	return Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: name}, Prefix: "itunes", Content: []ExtensionContent{{Text: value}}}
}
func itunesImage(image *ITunesImage) []Extension {
	if image == nil || image.Href == "" {
		return nil
	}
	return []Extension{{XMLName: xml.Name{Space: ITunesNamespace, Local: "image"}, Prefix: "itunes", Attrs: []xml.Attr{{Name: xml.Name{Local: "href"}, Value: image.Href}}}}
}
func itunesCategories(categories []ITunesCategory) []Extension {
	var result []Extension
	for _, category := range categories {
		if category.Text == "" {
			continue
		}
		element := Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: "category"}, Prefix: "itunes", Attrs: []xml.Attr{{Name: xml.Name{Local: "text"}, Value: category.Text}}}
		for _, child := range itunesCategories(category.Categories) {
			copy := child
			element.Content = append(element.Content, ExtensionContent{Element: &copy})
		}
		result = append(result, element)
	}
	return result
}
func itunesBool(name string, value *bool) []Extension {
	if value == nil {
		return nil
	}
	return []Extension{itunesText(name, strconv.FormatBool(*value))}
}
func appendITunesText(extensions []Extension, name, value string) []Extension {
	if value != "" {
		return append(extensions, itunesText(name, value))
	}
	return extensions
}

// Extensions converts optional show metadata to the existing XML extension model.
// Nil receivers and zero-valued metadata produce no elements.
func (f *ITunesFeed) Extensions() []Extension {
	if f == nil {
		return nil
	}
	var x []Extension
	for _, field := range []struct{ name, value string }{{"title", f.Title}, {"author", f.Author}, {"subtitle", f.Subtitle}, {"summary", f.Summary}, {"keywords", f.Keywords}, {"type", string(f.Type)}, {"new-feed-url", f.NewFeedURL}, {"applepodcastsverify", f.ApplePodcastsVerify}} {
		x = appendITunesText(x, field.name, field.value)
	}
	x = append(x, itunesImage(f.Image)...)
	x = append(x, itunesCategories(f.Categories)...)
	x = append(x, itunesBool("explicit", f.Explicit)...)
	if f.Owner != nil {
		owner := Extension{XMLName: xml.Name{Space: ITunesNamespace, Local: "owner"}, Prefix: "itunes"}
		for _, field := range []struct{ name, value string }{{"name", f.Owner.Name}, {"email", f.Owner.Email}} {
			if field.value != "" {
				child := itunesText(field.name, field.value)
				owner.Content = append(owner.Content, ExtensionContent{Element: &child})
			}
		}
		if len(owner.Content) > 0 {
			x = append(x, owner)
		}
	}
	if f.Block {
		x = append(x, itunesText("block", "Yes"))
	}
	if f.Complete {
		x = append(x, itunesText("complete", "Yes"))
	}
	return x
}

// Extensions converts optional episode metadata to the existing XML extension model.
func (i *ITunesItem) Extensions() []Extension {
	if i == nil {
		return nil
	}
	var x []Extension
	for _, field := range []struct{ name, value string }{{"title", i.Title}, {"author", i.Author}, {"subtitle", i.Subtitle}, {"summary", i.Summary}, {"keywords", i.Keywords}, {"duration", i.Duration}, {"episodeType", string(i.EpisodeType)}} {
		x = appendITunesText(x, field.name, field.value)
	}
	x = append(x, itunesImage(i.Image)...)
	x = append(x, itunesBool("explicit", i.Explicit)...)
	for _, field := range []struct {
		name  string
		value int
	}{{"episode", i.Episode}, {"season", i.Season}, {"order", i.Order}} {
		if field.value != 0 {
			x = append(x, itunesText(field.name, strconv.Itoa(field.value)))
		}
	}
	if i.Block {
		x = append(x, itunesText("block", "Yes"))
	}
	if i.IsClosedCaptioned != nil {
		value := "No"
		if *i.IsClosedCaptioned {
			value = "Yes"
		}
		x = append(x, itunesText("isClosedCaptioned", value))
	}
	return x
}

func mergeITunesExtensions(existing, generated []Extension) []Extension {
	if len(generated) == 0 {
		return existing
	}
	overridden := make(map[xml.Name]bool)
	for _, element := range generated {
		overridden[element.XMLName] = true
	}
	result := make([]Extension, 0, len(existing)+len(generated))
	for _, element := range existing {
		if !overridden[element.XMLName] {
			result = append(result, element)
		}
	}
	return append(result, generated...)
}
func hasITunesExtensions(extensions []Extension) bool {
	for _, element := range extensions {
		if element.XMLName.Space == ITunesNamespace {
			return true
		}
		for _, part := range element.Content {
			if part.Element != nil && hasITunesExtensions([]Extension{*part.Element}) {
				return true
			}
		}
	}
	return false
}
