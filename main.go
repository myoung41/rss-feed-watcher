package main

import (
	"bufio"
	"crypto/fnv"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	var stateFile string
	var listFile string
	var timeout time.Duration
	var showFirst bool

	flag.StringVar(&stateFile, "state", "", "path to state file (default: derived from the feed URL under the user config dir). Not allowed with -list.")
	flag.StringVar(&listFile, "list", "", "path to a file of feed URLs, one per line, to check as a batch instead of a single feed on the command line")
	flag.DurationVar(&timeout, "timeout", 15*time.Second, "HTTP request timeout")
	flag.BoolVar(&showFirst, "first-run-show", false, "print all items on the first run instead of just recording them as a baseline")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <feed-url>\n       %s [flags] -list <file>\n", os.Args[0], os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	var feedURLs []string
	if listFile != "" {
		if flag.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "feedwatch: a feed URL argument can't be combined with -list")
			os.Exit(2)
		}
		if stateFile != "" {
			fmt.Fprintln(os.Stderr, "feedwatch: -state can't be combined with -list; each feed in the list gets its own derived state file")
			os.Exit(2)
		}
		urls, err := readURLList(listFile)
		if err != nil {
			fmt.Fprintln(os.Stderr, "feedwatch:", err)
			os.Exit(1)
		}
		if len(urls) == 0 {
			fmt.Fprintln(os.Stderr, "feedwatch: no feed URLs found in", listFile)
			os.Exit(1)
		}
		feedURLs = urls
	} else {
		if flag.NArg() != 1 {
			flag.Usage()
			os.Exit(2)
		}
		feedURLs = []string{flag.Arg(0)}
	}

	client := &http.Client{Timeout: timeout}

	failed := false
	for _, feedURL := range feedURLs {
		sf := stateFile
		if sf == "" {
			path, err := defaultStatePath(feedURL)
			if err != nil {
				fmt.Fprintln(os.Stderr, "feedwatch:", err)
				failed = true
				continue
			}
			sf = path
		}
		if err := checkFeed(client, feedURL, sf, showFirst); err != nil {
			fmt.Fprintf(os.Stderr, "feedwatch: %s: %v\n", feedURL, err)
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

// checkFeed fetches a single feed, prints any items not already recorded
// in its state file, and updates the state file. Errors are returned
// rather than printed so a batch run (-list) can report which feed URL
// they came from and keep going with the rest.
func checkFeed(client *http.Client, feedURL, stateFile string, showFirst bool) error {
	body, err := fetch(client, feedURL)
	if err != nil {
		return err
	}

	items, err := ParseFeed(body)
	if err != nil {
		return err
	}

	state, err := loadState(stateFile)
	if err != nil {
		return err
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
		return err
	}

	if firstRun && !showFirst {
		fmt.Fprintf(os.Stderr, "feedwatch: recorded %d item(s) as baseline in %s, run again later to see new ones\n", len(items), stateFile)
	}
	return nil
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

// readURLList reads one feed URL per line. Blank lines and lines starting
// with '#' (after trimming) are skipped so the list file can carry
// comments and spacing without extra tooling.
func readURLList(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var urls []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		urls = append(urls, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return urls, nil
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
