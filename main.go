package main

import (
	"crypto/fnv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	var stateFile string
	var timeout time.Duration
	var showFirst bool

	flag.StringVar(&stateFile, "state", "", "path to state file (default: derived from the feed URL under the user config dir)")
	flag.DurationVar(&timeout, "timeout", 15*time.Second, "HTTP request timeout")
	flag.BoolVar(&showFirst, "first-run-show", false, "print all items on the first run instead of just recording them as a baseline")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <feed-url>\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(2)
	}
	feedURL := flag.Arg(0)

	if stateFile == "" {
		path, err := defaultStatePath(feedURL)
		if err != nil {
			fmt.Fprintln(os.Stderr, "feedwatch:", err)
			os.Exit(1)
		}
		stateFile = path
	}

	client := &http.Client{Timeout: timeout}
	body, err := fetch(client, feedURL)
	if err != nil {
		fmt.Fprintln(os.Stderr, "feedwatch:", err)
		os.Exit(1)
	}

	items, err := ParseFeed(body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "feedwatch:", err)
		os.Exit(1)
	}

	state, err := loadState(stateFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "feedwatch:", err)
		os.Exit(1)
	}

	// An empty state file means we've never checked this feed before.
	// Dumping every existing item on that run is noisy and usually not
	// what you want from a "what's new" tool, so we record a baseline
	// instead unless the caller asks otherwise.
	firstRun := len(state.Seen) == 0
	now := time.Now().Unix()

	for _, it := range items {
		id := it.ID()
		if _, ok := state.Seen[id]; ok {
			continue
		}
		state.Seen[id] = now
		if firstRun && !showFirst {
			continue
		}
		fmt.Printf("%s\n%s\n\n", it.Title, it.Link)
	}

	if err := saveState(stateFile, state); err != nil {
		fmt.Fprintln(os.Stderr, "feedwatch:", err)
		os.Exit(1)
	}

	if firstRun && !showFirst {
		fmt.Fprintf(os.Stderr, "feedwatch: recorded %d item(s) as baseline in %s, run again later to see new ones\n", len(items), stateFile)
	}
}

func fetch(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "feedwatch/1.0 (+https://github.com/myoung41/rss-feed-watcher)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetching feed: unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func defaultStatePath(feedURL string) (string, error) {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	h := fnv.New64a()
	h.Write([]byte(feedURL))
	name := fmt.Sprintf("%x.json", h.Sum64())
	return filepath.Join(cfgDir, "feedwatch", name), nil
}
