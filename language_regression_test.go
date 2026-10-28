package feeds

import (
	"encoding/json"
	"encoding/xml"
	"testing"
)

// These probes use pre-existing APIs and expose the old language data loss.
func TestLanguageInputSurvivesLowLevelRoundtrip(t *testing.T) {
	for _, tc := range []struct {
		name, input string
		value       interface{}
	}{
		{"Atom feed", `<feed xml:lang="en-US"/>`, &AtomFeed{}},
		{"Atom entry", `<entry xml:lang="ja"/>`, &AtomEntry{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := xml.Unmarshal([]byte(tc.input), tc.value); err != nil {
				t.Fatal(err)
			}
			output, err := xml.Marshal(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			var original, decoded struct {
				Language string `xml:"http://www.w3.org/XML/1998/namespace lang,attr"`
			}
			if err := xml.Unmarshal([]byte(tc.input), &original); err != nil {
				t.Fatal(err)
			}
			if err := xml.Unmarshal(output, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Language != original.Language {
				t.Fatalf("language lost: input %s, output %s", tc.input, output)
			}
		})
	}
	t.Run("JSON item", func(t *testing.T) {
		var item JSONItem
		if err := json.Unmarshal([]byte(`{"id":"one","language":"zh-Hant"}`), &item); err != nil {
			t.Fatal(err)
		}
		output, err := json.Marshal(item)
		if err != nil {
			t.Fatal(err)
		}
		var decoded map[string]interface{}
		if err := json.Unmarshal(output, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded["language"] != "zh-Hant" {
			t.Fatalf("JSON item language lost: %s", output)
		}
	})
}
