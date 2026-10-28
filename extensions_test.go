package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

func sampleExtension() Extension {
	return Extension{XMLName: xml.Name{Space: "urn:example:extension", Local: "title"}, Attrs: []xml.Attr{{Name: xml.Name{Space: "urn:example:attribute", Local: "id"}, Value: `a"&<>`}, {Name: xml.Name{Space: "http://www.w3.org/XML/1998/namespace", Local: "lang"}, Value: "zh-Hant"}}, Content: []ExtensionContent{
		{Text: " before & < "},
		{Element: &Extension{XMLName: xml.Name{Space: "urn:example:child", Local: "strong"}, Content: []ExtensionContent{{Text: "nested"}}}},
		{Text: " middle "},
		{Element: &Extension{XMLName: xml.Name{Local: "plain"}, Content: []ExtensionContent{{Text: " unqualified "}}}},
		{Comment: "keep comment"}, {Text: " after "},
	}}
}

func TestExtensionsGenericMappings(t *testing.T) {
	extension := sampleExtension()
	f := &Feed{Id: "urn:feed", Title: "standard title", Extensions: []Extension{extension}, Items: []*Item{{Id: "urn:item", Title: "standard item", Extensions: []Extension{extension}}}}
	before, _ := json.Marshal(extension)
	for _, format := range []string{"Atom", "RSS"} {
		t.Run(format, func(t *testing.T) {
			var output string
			var err error
			var b bytes.Buffer
			if format == "Atom" {
				output, err = f.ToAtom()
				if err == nil {
					err = f.WriteAtom(&b)
				}
			} else {
				output, err = f.ToRss()
				if err == nil {
					err = f.WriteRss(&b)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			if output != b.String() {
				t.Fatal("writer output disagrees")
			}
			var feedExtensions, itemExtensions []Extension
			var title, itemTitle string
			if format == "Atom" {
				var a AtomFeed
				if err := xml.Unmarshal([]byte(output), &a); err != nil {
					t.Fatalf("%s: %v", output, err)
				}
				feedExtensions, itemExtensions = a.Extensions, a.Entries[0].Extensions
				title, itemTitle = a.Title, a.Entries[0].Title
			} else {
				var r RssFeedXml
				if err := xml.Unmarshal([]byte(output), &r); err != nil {
					t.Fatalf("%s: %v", output, err)
				}
				feedExtensions, itemExtensions = r.Channel.Extensions, r.Channel.Items[0].Extensions
				title, itemTitle = r.Channel.Title, r.Channel.Items[0].Title
			}
			if title != "standard title" || itemTitle != "standard item" {
				t.Fatalf("foreign title replaced standard fields: %s", output)
			}
			if !reflect.DeepEqual(feedExtensions, []Extension{extension}) || !reflect.DeepEqual(itemExtensions, []Extension{extension}) {
				t.Fatalf("extension names/content/attributes changed: %#v %#v\n%s", feedExtensions, itemExtensions, output)
			}
		})
	}
	// XML-only data never leaks into JSON Feed and source payloads are unchanged.
	j, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(j, "extension") || strings.Contains(j, "nested") {
		t.Fatal(j)
	}
	after, _ := json.Marshal(f.Extensions[0])
	if !bytes.Equal(before, after) {
		t.Fatal("source extension mutated")
	}
}

func TestExtensionPrefixScopesAndMixedContent(t *testing.T) {
	input := `<x:root xmlns:x="urn:one" xmlns:a="urn:attrs" a:id="1"> before <x:child xmlns:x="urn:two">child</x:child> after <plain xmlns="">value</plain><!--comment--></x:root>`
	var value Extension
	if err := xml.Unmarshal([]byte(input), &value); err != nil {
		t.Fatal(err)
	}
	if value.XMLName.Space != "urn:one" || value.Content[1].Element.XMLName.Space != "urn:two" || value.Content[3].Element.XMLName.Space != "" {
		t.Fatalf("scope resolution failed: %+v", value)
	}
	for _, form := range []interface{}{value, &value} {
		var b bytes.Buffer
		e := xml.NewEncoder(&b)
		e.Indent("", "    ")
		if err := e.Encode(form); err != nil {
			t.Fatal(err)
		}
		var again Extension
		if err := xml.Unmarshal(b.Bytes(), &again); err != nil {
			t.Fatalf("invalid output %s: %v", b.String(), err)
		}
		if !reflect.DeepEqual(value, again) {
			t.Fatalf("mixed whitespace or namespace changed: %s", b.String())
		}
	}
	// Namespace bindings declared on a feed ancestor are resolved before capture.
	for _, input := range []string{
		`<feed xmlns="http://www.w3.org/2005/Atom" xmlns:x="urn:one"><title>native</title><x:title x:id="one"><x:child/></x:title></feed>`,
		`<feed xmlns:x="urn:one"><title>native</title><x:title x:id="one"><x:child/></x:title></feed>`,
	} {
		var a AtomFeed
		if err := xml.Unmarshal([]byte(input), &a); err != nil {
			t.Fatal(err)
		}
		if a.Title != "native" || len(a.Extensions) != 1 || a.Extensions[0].Attrs[0].Name.Space != "urn:one" {
			t.Fatalf("foreign collision or ancestor binding lost: %+v", a)
		}
	}
}

func TestExtensionValidation(t *testing.T) {
	for _, extension := range []Extension{
		{XMLName: xml.Name{Local: "unqualified"}},
		{XMLName: xml.Name{Space: ns, Local: "title"}},
		{XMLName: xml.Name{Space: "urn:x", Local: "x:bad"}},
		{XMLName: xml.Name{Space: "urn:x", Local: "1bad"}},
		{XMLName: xml.Name{Space: "urn:x", Local: "ok"}, Attrs: []xml.Attr{{Name: xml.Name{Local: "xmlns"}, Value: "urn:bad"}}},
		{XMLName: xml.Name{Space: "urn:x", Local: "ok"}, Attrs: []xml.Attr{{Name: xml.Name{Local: "a"}}, {Name: xml.Name{Local: "a"}}}},
		{XMLName: xml.Name{Space: "urn:x", Local: "ok"}, Content: []ExtensionContent{{Text: "text", Element: &Extension{XMLName: xml.Name{Space: "urn:x", Local: "child"}}}}},
		{XMLName: xml.Name{Space: "urn:x", Local: "ok"}, Content: []ExtensionContent{{Comment: "bad--comment"}}},
	} {
		f := &Feed{Extensions: []Extension{extension}}
		if _, err := f.ToAtom(); err == nil {
			t.Fatalf("expected invalid extension error: %+v", extension)
		}
	}
	var value Extension
	if err := xml.Unmarshal([]byte(`<x xmlns="urn:x"><child></x>`), &value); err == nil {
		t.Fatal("malformed extension accepted")
	}
}

func TestLowLevelExtensionsAndReuse(t *testing.T) {
	for _, value := range []interface{}{
		&AtomFeed{Extensions: []Extension{sampleExtension()}},
		&AtomEntry{Extensions: []Extension{sampleExtension()}},
		&RssFeed{Extensions: []Extension{sampleExtension()}},
		&RssItem{Extensions: []Extension{sampleExtension()}},
	} {
		for _, form := range []interface{}{value, reflect.ValueOf(value).Elem().Interface()} {
			output, err := xml.Marshal(form)
			if err != nil {
				t.Fatal(err)
			}
			again := reflect.New(reflect.TypeOf(value).Elem()).Interface()
			if err := xml.Unmarshal(output, again); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(reflect.ValueOf(value).Elem().FieldByName("Extensions").Interface(), reflect.ValueOf(again).Elem().FieldByName("Extensions").Interface()) {
				t.Fatalf("low-level roundtrip failed: %s", output)
			}
		}
	}
	var feed AtomFeed
	if err := xml.Unmarshal([]byte(`<feed xmlns:x="urn:x"><x:one/></feed>`), &feed); err != nil {
		t.Fatal(err)
	}
	if err := xml.Unmarshal([]byte(`<feed/>`), &feed); err != nil {
		t.Fatal(err)
	}
	if len(feed.Extensions) != 1 {
		t.Fatal("absent extensions cleared old values")
	}
	if err := xml.Unmarshal([]byte(`<feed xmlns:x="urn:x"><x:two/></feed>`), &feed); err != nil {
		t.Fatal(err)
	}
	if len(feed.Extensions) != 1 || feed.Extensions[0].XMLName.Local != "two" {
		t.Fatal("explicit extension collection did not replace old values")
	}
}
