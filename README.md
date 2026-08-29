# rss-feed-watcher

A command-line tool that fetches an RSS or Atom feed and prints only the
items it hasn't shown you before.

I wanted something I could stick in a cron job or a shell script to
notify me when a blog or changelog posts something new, without pulling
in a feed reader, a database, or a mail client. `feedwatch` does one
thing: it remembers which item IDs it has already printed, and only
prints the ones it hasn't.

## Build

```
go build -o feedwatch .
```

Requires Go 1.22+ and nothing else. No third-party dependencies.

## Usage

```
feedwatch https://example.com/feed.xml
```

The first time you run it against a given feed, it won't print
anything - it records every current item as a baseline and tells you
so on stderr. Run it again after the feed has new items and you'll see
those:

```
$ feedwatch https://blog.example.com/rss.xml
New Release: v2.3.0
https://blog.example.com/posts/v2.3.0

Migrating our build pipeline
https://blog.example.com/posts/build-pipeline
```

Each line pair is a title followed by a link, with a blank line
between items, so it's easy to pipe into something else:

```
feedwatch https://blog.example.com/rss.xml | mail -s "blog updates" me@example.com
```

Run it from cron every 15 minutes and you get a lightweight "what's
new" notifier with no server component.

### Flags

```
-state string
      path to state file (default: derived from the feed URL under the
      user config dir, e.g. ~/.config/feedwatch/<hash>.json on Linux)
-timeout duration
      HTTP request timeout (default 15s)
-first-run-show
      print all items on the first run instead of just recording them
      as a baseline
```

### State

State is a small JSON file mapping item IDs to the Unix timestamp they
were first seen. Delete it to reset a feed back to "never checked".
Each feed URL gets its own state file by default, so you can watch as
many feeds as you like without them stepping on each other.

## What it doesn't do

No OPML import, no HTML rendering, no feed discovery from a site URL,
no persistent daemon. If you want a full reader, this isn't it - it's
meant to be one link in a pipeline.
