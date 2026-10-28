package feeds

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"reflect"
	"strings"
	"testing"
)

// Uses the existing Feed.Image API, so these tests can also run on the baseline.
func TestFeedImageMappings(t *testing.T) {
	for _, tc := range []struct {
		name  string
		image *Image
	}{
		{"nil", nil},
		{"empty", &Image{}},
		{"URL only", &Image{Url: "https://example.com/logo.png"}},
		{"complete RSS image", &Image{Url: "https://example.com/logo.png", Title: "Logo", Link: "https://example.com/", Width: 144, Height: 72}},
		{"Unicode and escaping", &Image{Url: `https://example.com/图片.png?a=1&b="two"`, Title: "图片 & more", Link: "https://example.com/"}},
		{"relative image", &Image{Url: "images/logo.png"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			want := ""
			var before Image
			if tc.image != nil {
				want = tc.image.Url
				before = *tc.image
			}
			f := &Feed{Id: "urn:feed", XMLBase: "https://example.com/", Image: tc.image, Items: []*Item{{Id: "urn:item", Enclosure: &Enclosure{Url: "https://example.com/item.jpg", Type: "image/jpeg", Length: "42"}}}}
			a := (&Atom{f}).AtomFeed()
			if a.Logo != want || a.Icon != "" {
				t.Fatalf("logo=%q icon=%q; want logo=%q only", a.Logo, a.Icon, want)
			}
			atom := atomXMLForTest(t, f)
			var decodedAtom AtomFeed
			if err := xml.Unmarshal([]byte(atom), &decodedAtom); err != nil {
				t.Fatal(err)
			}
			if decodedAtom.Logo != want || decodedAtom.Icon != "" {
				t.Fatalf("Atom image roundtrip: %s", atom)
			}
			if want == "" && (strings.Contains(atom, "<logo>") || strings.Contains(atom, "<icon>")) {
				t.Fatalf("empty Atom images should be omitted: %s", atom)
			}
			output, err := f.ToJSON()
			if err != nil {
				t.Fatal(err)
			}
			var writer bytes.Buffer
			if err := f.WriteJSON(&writer); err != nil {
				t.Fatal(err)
			}
			var decodedJSON, written JSONFeed
			if err := json.Unmarshal([]byte(output), &decodedJSON); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(writer.Bytes(), &written); err != nil {
				t.Fatal(err)
			}
			if decodedJSON.Icon != want || decodedJSON.Favicon != "" || !reflect.DeepEqual(decodedJSON, written) {
				t.Fatalf("JSON images: %s", output)
			}
			if decodedJSON.Items[0].Image != "https://example.com/item.jpg" {
				t.Fatal("feed image replaced item enclosure image")
			}
			if want == "" && (strings.Contains(output, `"icon"`) || strings.Contains(output, `"favicon"`)) {
				t.Fatalf("empty JSON images should be omitted: %s", output)
			}
			rss := (&Rss{f}).RssFeed()
			if tc.image == nil {
				if rss.Image != nil {
					t.Fatal("nil image became RSS image")
				}
			} else {
				expected := &RssImage{Url: before.Url, Title: before.Title, Link: before.Link, Width: before.Width, Height: before.Height}
				if !reflect.DeepEqual(rss.Image, expected) {
					t.Fatalf("RSS image mapping changed: %+v", rss.Image)
				}
			}
			rssOutput, err := f.ToRss()
			if err != nil {
				t.Fatal(err)
			}
			writer.Reset()
			if err := f.WriteRss(&writer); err != nil {
				t.Fatal(err)
			}
			if rssOutput != writer.String() {
				t.Fatal("RSS writer disagrees")
			}
			if tc.image != nil && !reflect.DeepEqual(before, *tc.image) {
				t.Fatal("source image mutated")
			}
		})
	}
}

func TestLowLevelImageFieldsRemainIndependent(t *testing.T) {
	atom := &AtomFeed{Logo: "https://example.com/wide.png", Icon: "https://example.com/small.png"}
	output, err := ToXML(atom)
	if err != nil {
		t.Fatal(err)
	}
	var decoded AtomFeed
	if err := xml.Unmarshal([]byte(output), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Logo != atom.Logo || decoded.Icon != atom.Icon {
		t.Fatal("low-level Atom images changed")
	}
	j := &JSONFeed{Icon: "https://example.com/square.png", Favicon: "https://example.com/favicon.ico"}
	b, err := j.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip JSONFeed
	if err := json.Unmarshal([]byte(b), &roundtrip); err != nil {
		t.Fatal(err)
	}
	if roundtrip.Icon != j.Icon || roundtrip.Favicon != j.Favicon {
		t.Fatal("low-level JSON images changed")
	}
}
