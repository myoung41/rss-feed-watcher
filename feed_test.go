package main

import (
	"os"
	"testing"
)

func readTestdata(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatalf("reading testdata/%s: %v", name, err)
	}
	return data
}

func TestParseFeedRSS(t *testing.T) {
	items, err := ParseFeed(readTestdata(t, "sample.rss.xml"))
	if err != nil {
		t.Fatalf("ParseFeed: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("got %d items, want 3", len(items))
	}

	first := items[0]
	if first.Title != "First post" || first.Link != "https://example.com/posts/first" {
		t.Errorf("first item = %+v, unexpected fields", first)
	}
	if first.ID() != "urn:uuid:11111111-1111-1111-1111-111111111111" {
		t.Errorf("first.ID() = %q, want guid", first.ID())
	}

	// isPermaLink="false" doesn't change our GUID handling: it's still
	// the field we prefer as an identifier.
	second := items[1]
	if second.ID() != "tag:example.com,2026:second" {
		t.Errorf("second.ID() = %q, want the guid text", second.ID())
	}

	// No guid: falls back to link.
	third := items[2]
	if third.ID() != "https://example.com/posts/no-guid" {
		t.Errorf("third.ID() = %q, want link fallback", third.ID())
	}
}

func TestParseFeedAtom(t *testing.T) {
	items, err := ParseFeed(readTestdata(t, "sample.atom.xml"))
	if err != nil {
		t.Fatalf("ParseFeed: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}

	// Entry one has both a "self" and an "alternate" link; we want the
	// alternate one, not whichever comes first in the document.
	one := items[0]
	if one.Link != "https://example.com/entry-one" {
		t.Errorf("one.Link = %q, want the rel=alternate href", one.Link)
	}
	if one.ID() != "urn:uuid:1225c695-cfb8-4ebb-aaaa-80da344efa6a" {
		t.Errorf("one.ID() = %q, want atom id", one.ID())
	}

	// Entry two's only link has no rel attribute, which we also accept.
	two := items[1]
	if two.Link != "https://example.com/entry-two" {
		t.Errorf("two.Link = %q, want the unrelabeled href", two.Link)
	}
}

func TestParseFeedNoItems(t *testing.T) {
	if _, err := ParseFeed(readTestdata(t, "empty.xml")); err == nil {
		t.Fatal("ParseFeed on a feed with no items: expected an error, got nil")
	}
}

func TestParseFeedGarbage(t *testing.T) {
	if _, err := ParseFeed([]byte("not xml at all")); err == nil {
		t.Fatal("ParseFeed on non-XML input: expected an error, got nil")
	}
}

func TestItemIDFallbackOrder(t *testing.T) {
	cases := []struct {
		name string
		item Item
		want string
	}{
		{"guid wins", Item{Title: "t", Link: "l", GUID: "g"}, "g"},
		{"link when no guid", Item{Title: "t", Link: "l"}, "l"},
		{"title as last resort", Item{Title: "t"}, "t"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.item.ID(); got != c.want {
				t.Errorf("ID() = %q, want %q", got, c.want)
			}
		})
	}
}
