package feeds

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// Extension is a namespace-aware XML element. Names use namespace URIs in
// Space and unprefixed local names in Local; prefixes are chosen by encoding/xml.
type Extension struct {
	XMLName xml.Name
	Attrs   []xml.Attr
	Content []ExtensionContent
}

// ExtensionContent preserves the order of text, child elements, and comments.
// Set only one field per content item. Text is ordinary text, not raw XML.
type ExtensionContent struct {
	Text    string
	Element *Extension
	Comment string
}

func validExtensionName(name xml.Name) bool {
	if name.Local == "" || strings.ContainsAny(name.Local, ": \t\r\n<>/&=\"'") {
		return false
	}
	_, err := xml.NewDecoder(strings.NewReader("<" + name.Local + "/>")).Token()
	return err == nil
}

// MarshalXML emits expanded names and escapes text and attribute values.
func (x Extension) MarshalXML(e *xml.Encoder, _ xml.StartElement) error {
	if !validExtensionName(x.XMLName) || x.XMLName.Space == "http://www.w3.org/2000/xmlns/" {
		return fmt.Errorf("feeds: invalid extension element name %q", x.XMLName.Local)
	}
	start := xml.StartElement{Name: x.XMLName}
	seen := make(map[xml.Name]bool)
	for _, attr := range x.Attrs {
		if !validExtensionName(attr.Name) || attr.Name.Local == "xmlns" || attr.Name.Space == "xmlns" || attr.Name.Space == "http://www.w3.org/2000/xmlns/" {
			return fmt.Errorf("feeds: extension attributes must use expanded names, not namespace declarations: %v", attr.Name)
		}
		if seen[attr.Name] {
			return fmt.Errorf("feeds: duplicate extension attribute %v", attr.Name)
		}
		seen[attr.Name] = true
		start.Attr = append(start.Attr, attr)
	}
	// encoding/xml does not automatically reset an inherited default namespace
	// for unqualified elements. An explicit reset keeps nested payload names exact.
	if x.XMLName.Space == "" {
		start.Attr = append(start.Attr, xml.Attr{Name: xml.Name{Local: "xmlns"}, Value: ""})
	}
	var inner bytes.Buffer
	compact := xml.NewEncoder(&inner)
	for _, part := range x.Content {
		n := 0
		if part.Text != "" {
			n++
		}
		if part.Element != nil {
			n++
		}
		if part.Comment != "" {
			n++
		}
		if n > 1 {
			return fmt.Errorf("feeds: extension content must contain only one of text, element, or comment")
		}
		var err error
		switch {
		case part.Element != nil:
			err = compact.Encode(*part.Element)
		case part.Comment != "":
			if strings.Contains(part.Comment, "--") || strings.HasSuffix(part.Comment, "-") {
				return fmt.Errorf("feeds: invalid XML extension comment")
			}
			err = compact.EncodeToken(xml.Comment(part.Comment))
		default:
			err = compact.EncodeToken(xml.CharData(part.Text))
		}
		if err != nil {
			return err
		}
	}
	if err := compact.Flush(); err != nil {
		return err
	}
	// Only encoder-generated, escaped XML enters innerxml. Keeping the payload
	// compact prevents the outer feed encoder's indentation altering mixed text.
	return e.EncodeElement(struct {
		Inner string `xml:",innerxml"`
	}{Inner: inner.String()}, start)
}

// UnmarshalXML preserves resolved names, attributes, whitespace, and content order.
// Namespace declarations are not copied: expanded names carry their meaning.
func (x *Extension) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	value := Extension{XMLName: start.Name}
	for _, attr := range start.Attr {
		if attr.Name.Space == "xmlns" || (attr.Name.Space == "" && attr.Name.Local == "xmlns") {
			continue
		}
		value.Attrs = append(value.Attrs, attr)
	}
	for {
		token, err := d.Token()
		if err != nil {
			return err
		}
		switch token := token.(type) {
		case xml.StartElement:
			child := new(Extension)
			if err := d.DecodeElement(child, &token); err != nil {
				return err
			}
			value.Content = append(value.Content, ExtensionContent{Element: child})
		case xml.CharData:
			// Coalesce adjacent text tokens, whose boundaries depend on decoder buffering.
			if len(value.Content) > 0 && value.Content[len(value.Content)-1].Element == nil && value.Content[len(value.Content)-1].Comment == "" {
				value.Content[len(value.Content)-1].Text += string(token)
			} else {
				value.Content = append(value.Content, ExtensionContent{Text: string(token)})
			}
		case xml.Comment:
			value.Content = append(value.Content, ExtensionContent{Comment: string(token)})
		case xml.EndElement:
			*x = value
			return nil
		default:
			return fmt.Errorf("feeds: unsupported XML token %T in extension", token)
		}
	}
}

func validateExtensions(extensions []Extension, native string) error {
	for _, extension := range extensions {
		if native == "" && extension.XMLName.Space == ns && extension.XMLName.Local == "link" {
			return fmt.Errorf("feeds: use RssFeed.AtomLinks for Atom link elements")
		}
		if extension.XMLName.Space == "" || extension.XMLName.Space == native {
			return fmt.Errorf("feeds: top-level XML extension %q requires a foreign namespace URI", extension.XMLName.Local)
		}
	}
	return nil
}

// extensionTokenReader filters only foreign direct children. Nested native
// elements keep streaming through their own existing decoders.
type extensionTokenReader struct {
	decoder    *xml.Decoder
	start      *xml.StartElement
	depth      int
	native     string
	rss        bool
	extensions []Extension
	done       bool
}

func (r *extensionTokenReader) Token() (xml.Token, error) {
	if r.start != nil {
		start := *r.start
		r.start = nil
		r.depth = 1
		return start, nil
	}
	if r.done {
		return nil, io.EOF
	}
	for {
		token, err := r.decoder.Token()
		if err != nil {
			return nil, err
		}
		switch t := token.(type) {
		case xml.StartElement:
			foreign := t.Name.Space != "" && t.Name.Space != r.native
			if r.rss && t.Name.Space == ns && t.Name.Local == "link" {
				foreign = false
			}
			if r.depth == 1 && foreign {
				var extension Extension
				if err := r.decoder.DecodeElement(&extension, &t); err != nil {
					return nil, err
				}
				r.extensions = append(r.extensions, extension)
				continue
			}
			r.depth++
		case xml.EndElement:
			r.depth--
			if r.depth == 0 {
				r.done = true
			}
		}
		return token, nil
	}
}

// Decode foreign children before the legacy decoder can confuse a foreign
// element named title/link/category with an existing unqualified field.
func decodeExtensions(d *xml.Decoder, start xml.StartElement, value interface{}, native string, rss bool) ([]Extension, error) {
	reader := &extensionTokenReader{decoder: d, start: &start, native: native, rss: rss}
	decoder := xml.NewTokenDecoder(reader)
	if err := decoder.Decode(value); err != nil {
		return nil, err
	}
	return reader.extensions, nil
}
