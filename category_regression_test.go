package feeds

import (
	"encoding/xml"
	"testing"
)

// This regression uses only the existing API and can run against the baseline.
func TestLegacyAtomCategoryUsesTermAttribute(t *testing.T) {
	for _, value := range []interface{}{AtomFeed{Category: "Go"}, AtomEntry{Category: "Go"}} {
		output, err := xml.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Categories []struct {
				Term string `xml:"term,attr"`
			} `xml:"category"`
		}
		if err := xml.Unmarshal(output, &decoded); err != nil {
			t.Fatal(err)
		}
		if len(decoded.Categories) != 1 || decoded.Categories[0].Term != "Go" {
			t.Fatalf("Atom Category must use term attribute: %s", output)
		}
	}
}
