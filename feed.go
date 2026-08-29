package main

import (
	"encoding/xml"
	"fmt"
)

// Item is the subset of an RSS <item> or Atom <entry> we care about.
type Item struct {
	Title string
	Link  string
	GUID  string
}

// ID returns a stable identifier for deduping across runs. Feeds are
// inconsistent about which fields they bother to fill in, so we fall
// back in order of how likely a field is to actually be stable.
func (it Item) ID() string {
	if it.GUID != "" {
		return it.GUID
	}
	if it.Link != "" {
		return it.Link
	}
	return it.Title
}

type rssFeed struct {
	Channel struct {
		Items []struct {
			Title string `xml:"title"`
			Link  string `xml:"link"`
			GUID  string `xml:"guid"`
		} `xml:"item"`
	} `xml:"channel"`
}

type atomFeed struct {
	Entries []struct {
		Title string `xml:"title"`
		ID    string `xml:"id"`
		Links []struct {
			Href string `xml:"href,attr"`
			Rel  string `xml:"rel,attr"`
		} `xml:"link"`
	} `xml:"entry"`
}

// ParseFeed handles RSS 2.0 and Atom, which is the split you actually
// run into in the wild. Both are just XML with different tag names, so
// we try RSS first and fall back to Atom rather than sniffing headers.
func ParseFeed(data []byte) ([]Item, error) {
	var rss rssFeed
	if err := xml.Unmarshal(data, &rss); err == nil && len(rss.Channel.Items) > 0 {
		items := make([]Item, 0, len(rss.Channel.Items))
		for _, it := range rss.Channel.Items {
			items = append(items, Item{Title: it.Title, Link: it.Link, GUID: it.GUID})
		}
		return items, nil
	}

	var atom atomFeed
	if err := xml.Unmarshal(data, &atom); err == nil && len(atom.Entries) > 0 {
		items := make([]Item, 0, len(atom.Entries))
		for _, e := range atom.Entries {
			link := ""
			for _, l := range e.Links {
				if l.Rel == "alternate" || l.Rel == "" {
					link = l.Href
					break
				}
			}
			items = append(items, Item{Title: e.Title, Link: link, GUID: e.ID})
		}
		return items, nil
	}

	return nil, fmt.Errorf("no recognizable RSS or Atom items found")
}
