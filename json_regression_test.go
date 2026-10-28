package feeds

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestJSONFieldNamesCompatibility(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		banner      string
		size        int32
	}{
		{"standard", `{"banner_image":"new","attachments":[{"size_in_bytes":42}]}`, "new", 42},
		{"legacy", `{"banner_":"old","attachments":[{"size":21}]}`, "old", 21},
		{"standard wins", `{"banner_":"old","banner_image":"new","attachments":[{"size":21,"size_in_bytes":42}]}`, "new", 42},
		{"reverse order", `{"banner_image":"new","banner_":"old","attachments":[{"size_in_bytes":42,"size":21}]}`, "new", 42},
		{"explicit zero", `{"banner_":"old","banner_image":"","attachments":[{"size":21,"size_in_bytes":0}]}`, "", 0},
		{"explicit null", `{"banner_":"old","banner_image":null,"attachments":[{"size":21,"size_in_bytes":null}]}`, "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var item JSONItem
			if err := json.Unmarshal([]byte(tc.input), &item); err != nil {
				t.Fatal(err)
			}
			if item.BannerImage != tc.banner || item.Attachments[0].Size != tc.size {
				t.Fatalf("decoded: %+v", item)
			}
			b, err := json.Marshal(item)
			if err != nil {
				t.Fatal(err)
			}
			var output map[string]json.RawMessage
			if err := json.Unmarshal(b, &output); err != nil {
				t.Fatal(err)
			}
			if _, ok := output["banner_"]; ok {
				t.Fatalf("legacy key in %s", b)
			}
			var attachments []map[string]json.RawMessage
			if err := json.Unmarshal(output["attachments"], &attachments); err != nil {
				t.Fatal(err)
			}
			if _, ok := attachments[0]["size"]; ok {
				t.Fatalf("legacy size in %s", b)
			}
			var roundtrip JSONItem
			if err := json.Unmarshal(b, &roundtrip); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(item, roundtrip) {
				t.Fatalf("roundtrip changed data: %s", b)
			}
		})
	}
	// Missing/null fields preserve values when decoding into reused objects.
	item := JSONItem{BannerImage: "keep", Title: "old"}
	att := JSONAttachment{Size: 17}
	for _, input := range []string{`{}`, `{"banner_image":null,"banner_":"ignored"}`} {
		if err := json.Unmarshal([]byte(input), &item); err != nil {
			t.Fatal(err)
		}
	}
	for _, input := range []string{`{}`, `{"size_in_bytes":null,"size":99}`} {
		if err := json.Unmarshal([]byte(input), &att); err != nil {
			t.Fatal(err)
		}
	}
	if item.BannerImage != "keep" || att.Size != 17 {
		t.Fatal("missing/null fields cleared existing values")
	}
	for _, input := range []string{`{"banner_image":3}`, `{"banner_":3}`, `{"attachments":[{"size_in_bytes":"bad"}]}`, `{"attachments":[{"size":2147483648}]}`} {
		if err := json.Unmarshal([]byte(input), &JSONItem{}); err == nil {
			t.Fatalf("expected invalid field error: %s", input)
		}
	}
}

func TestJSONAttachmentDurationMarshalForms(t *testing.T) {
	for _, duration := range []time.Duration{0, time.Second, 1500 * time.Millisecond} {
		a := JSONAttachment{Url: "https://example.com/a", Duration: duration, Size: 42}
		forms := map[string]any{"value": a, "pointer": &a, "map": map[string]JSONAttachment{"a": a}, "slice": []JSONAttachment{a}, "item": JSONItem{Attachments: []JSONAttachment{a}}}
		for name, value := range forms {
			t.Run(name+"/"+duration.String(), func(t *testing.T) {
				b, err := json.Marshal(value)
				if err != nil {
					t.Fatal(err)
				}
				var decoded any
				if err := json.Unmarshal(b, &decoded); err != nil {
					t.Fatal(err)
				}
				var check func(any)
				check = func(v any) {
					switch v := v.(type) {
					case map[string]any:
						if _, ok := v["url"]; ok {
							if duration == 0 {
								if _, ok := v["duration_in_seconds"]; ok {
									t.Fatalf("zero should remain omitted: %s", b)
								}
							} else if v["duration_in_seconds"] != duration.Seconds() {
								t.Fatalf("wrong duration: %s", b)
							}
							if v["size_in_bytes"] != float64(42) {
								t.Fatalf("wrong size: %s", b)
							}
						}
						for _, child := range v {
							check(child)
						}
					case []any:
						for _, child := range v {
							check(child)
						}
					}
				}
				check(decoded)
			})
		}
	}
	var p *JSONAttachment
	b, err := json.Marshal(p)
	if err != nil || string(b) != "null" {
		t.Fatalf("nil attachment: %s %v", b, err)
	}
}

func TestJSONAttachmentDurationUnmarshal(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		want        time.Duration
		bad         bool
	}{
		{"zero", `{"duration_in_seconds":0}`, 0, false},
		{"fraction", `{"duration_in_seconds":1.5}`, 1500 * time.Millisecond, false},
		{"missing", `{}`, 5 * time.Second, false},
		{"null", `{"duration_in_seconds":null}`, 5 * time.Second, false},
		{"negative preserves legacy behavior", `{"duration_in_seconds":-1}`, 5 * time.Second, false},
		{"invalid", `{"duration_in_seconds":"bad"}`, 5 * time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := JSONAttachment{Duration: 5 * time.Second, Size: 17}
			err := json.Unmarshal([]byte(tc.input), &a)
			if (err != nil) != tc.bad {
				t.Fatalf("error: %v", err)
			}
			if a.Duration != tc.want || a.Size != 17 {
				t.Fatalf("decoded: %+v", a)
			}
		})
	}
}

func TestJSONEmptyItemsAllAPIs(t *testing.T) {
	for _, items := range [][]*JSONItem{nil, {}} {
		original := JSONFeed{Version: jsonFeedVersion, Title: "test", Items: items}
		for _, v := range []any{original, &original, map[string]JSONFeed{"feed": original}} {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			if _, ok := v.(map[string]JSONFeed); ok {
				var m map[string]json.RawMessage
				if err := json.Unmarshal(b, &m); err != nil {
					t.Fatal(err)
				}
				b = m["feed"]
			}
			assertEmptyItems(t, b)
		}
		b, err := original.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		assertEmptyItems(t, []byte(b))
		if (original.Items == nil) != (items == nil) {
			t.Fatal("marshaling mutated original")
		}
	}
	for _, items := range [][]*Item{nil, {}} {
		f := Feed{Title: "test", Items: items}
		s, err := f.ToJSON()
		if err != nil {
			t.Fatal(err)
		}
		assertEmptyItems(t, []byte(s))
		var b bytes.Buffer
		if err := f.WriteJSON(&b); err != nil {
			t.Fatal(err)
		}
		assertEmptyItems(t, b.Bytes())
		if (f.Items == nil) != (items == nil) {
			t.Fatal("marshaling mutated generic feed")
		}
	}
	populated := JSONFeed{Items: []*JSONItem{{Id: "one", ContentText: "body"}}}
	b, err := json.Marshal(populated)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip JSONFeed
	if err := json.Unmarshal(b, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(populated, roundtrip) {
		t.Fatalf("populated feed changed: %s", b)
	}
	var p *JSONFeed
	b, err = json.Marshal(p)
	if err != nil || string(b) != "null" {
		t.Fatalf("nil feed: %s %v", b, err)
	}
}
func assertEmptyItems(t *testing.T, b []byte) {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if string(m["items"]) != "[]" {
		t.Fatalf("expected required items:[] in %s", b)
	}
}
