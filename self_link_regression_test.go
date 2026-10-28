package feeds

import (
	"encoding/json"
	"strings"
	"testing"
)

// This regression uses only the existing API, and can also run on the old main.
func TestExplicitSelfIsNotAHomePage(t *testing.T) {
	f := &Feed{Link: &Link{Href: "https://example.com/feed", Rel: "self"}}
	output, err := f.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	var urls struct {
		Home string `json:"home_page_url"`
		Feed string `json:"feed_url"`
	}
	if err := json.Unmarshal([]byte(output), &urls); err != nil {
		t.Fatal(err)
	}
	if urls.Home != "" || urls.Feed != "https://example.com/feed" {
		t.Fatalf("self URL must be feed_url, not home_page_url: %s", output)
	}
	rss, err := f.ToRss()
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(rss, "<link>https://example.com/feed</link>") {
		t.Fatalf("self URL used as RSS homepage: %s", rss)
	}
}
