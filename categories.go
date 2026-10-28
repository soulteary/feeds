package feeds

import "encoding/xml"

// AtomCategory represents an Atom category's term and optional metadata.
type AtomCategory struct {
	XMLName xml.Name `xml:"category"`
	Term    string   `xml:"term,attr"`
	Scheme  string   `xml:"scheme,attr,omitempty"`
	Label   string   `xml:"label,attr,omitempty"`
}

// UnmarshalXML also accepts the legacy text-only category representation.
func (c *AtomCategory) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	var value struct {
		XMLName xml.Name `xml:"category"`
		Term    *string  `xml:"term,attr"`
		Scheme  string   `xml:"scheme,attr"`
		Label   string   `xml:"label,attr"`
		Text    string   `xml:",chardata"`
	}
	if err := d.DecodeElement(&value, &start); err != nil {
		return err
	}
	term := value.Text
	if value.Term != nil {
		term = *value.Term
	}
	*c = AtomCategory{XMLName: value.XMLName, Term: term, Scheme: value.Scheme, Label: value.Label}
	return nil
}

func categoryNames(primary string, categories []string) []string {
	var result []string
	seen := make(map[string]bool)
	add := func(name string) {
		if name != "" && !seen[name] {
			seen[name] = true
			result = append(result, name)
		}
	}
	add(primary)
	for _, name := range categories {
		add(name)
	}
	return result
}

func atomCategoriesFromNames(primary string, categories []string) []AtomCategory {
	var result []AtomCategory
	for _, term := range categoryNames(primary, categories) {
		result = append(result, AtomCategory{Term: term})
	}
	return result
}

func atomCategories(primary string, categories []AtomCategory) []AtomCategory {
	var result []AtomCategory
	type key struct{ term, scheme, label string }
	seen := make(map[key]bool)
	add := func(category AtomCategory) {
		k := key{category.Term, category.Scheme, category.Label}
		if category.Term != "" && !seen[k] {
			seen[k] = true
			result = append(result, category)
		}
	}
	if primary != "" {
		first := AtomCategory{Term: primary}
		// A decoded Category mirrors the first term. Retain its richer metadata
		// rather than emitting a duplicate term without scheme or label.
		for _, category := range categories {
			if category.Term == primary {
				first = category
				break
			}
		}
		add(first)
	}
	for _, category := range categories {
		add(category)
	}
	return result
}
