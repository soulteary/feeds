package feeds

import "mime"

// collectLinks returns distinct, nonempty links in primary-then-additional order.
func collectLinks(primary *Link, additional []*Link) []Link {
	var result []Link
	seen := make(map[Link]bool)
	add := func(link *Link) {
		if link == nil || link.Href == "" {
			return
		}
		key := *link
		if key.Rel == "" {
			key.Rel = "alternate"
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, *link)
		}
	}
	add(primary)
	for _, link := range additional {
		add(link)
	}
	return result
}

func homeLink(primary *Link, additional []*Link) *Link {
	// Preserve the legacy primary link's meaning unless its relation explicitly
	// identifies another resource, such as self, hub, or enclosure.
	if primary != nil && primary.Href != "" && (primary.Rel == "" || primary.Rel == "alternate") {
		copy := *primary
		return &copy
	}
	for _, link := range collectLinks(nil, additional) {
		media, _, _ := mime.ParseMediaType(link.Type)
		if (link.Rel == "" || link.Rel == "alternate") && (link.Type == "" || media == "text/html" || media == "application/xhtml+xml") {
			copy := link
			return &copy
		}
	}
	return nil
}

// Prefer a format-specific self URL over an untyped self URL. Never infer a
// subscription address from the home page, and never reuse another format's URL.
func selfLink(primary *Link, additional []*Link, mediaType string) *Link {
	var fallback *Link
	for _, link := range collectLinks(primary, additional) {
		if link.Rel != "self" {
			continue
		}
		media, _, _ := mime.ParseMediaType(link.Type)
		if media == mediaType {
			copy := link
			return &copy
		}
		if link.Type == "" && fallback == nil {
			copy := link
			fallback = &copy
		}
	}
	return fallback
}

func atomLinks(primary *Link, additional []*Link, mediaType string) []AtomLink {
	var result []AtomLink
	var selected *Link
	if mediaType != "" {
		selected = selfLink(primary, additional, mediaType)
	}
	for _, link := range collectLinks(primary, additional) {
		if mediaType != "" && link.Rel == "self" && (selected == nil || link != *selected) {
			continue
		}
		rel := link.Rel
		// Preserve the legacy feed Link's empty rel; Atom's default is alternate.
		if rel == "" && (mediaType == "" || primary == nil || link != *primary) {
			rel = "alternate"
		}
		media := link.Type
		if rel == "self" && media == "" && mediaType != "" {
			media = mediaType
		}
		result = append(result, AtomLink{Href: link.Href, Rel: rel, Type: media, Length: link.Length})
	}
	return result
}

func hasEnclosure(links []AtomLink, href string) bool {
	for _, link := range links {
		if link.Rel == "enclosure" && link.Href == href {
			return true
		}
	}
	return false
}
