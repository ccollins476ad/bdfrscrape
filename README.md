# bdfrscrape

```
bdfrscrape [flags]... <source> <dest_dir>
```

bdfrscrape is an unofficial companion tool to [Bulk Downloader for Reddit](https://github.com/Serene-Arc/bulk-downloader-for-reddit) (abbreviated as "bdfr" in this document).

## Installation

Download the go compiler, then run the below command:

```
go install github.com/ccollins476ad/bdfrscrape@latest
```

## Suggested Use

bdfrscrape is not very useful by itself, since its output is not easily consumable by humans. I recommend the [reddscare](https://github.com/ccollins476ad/reddscare) tool, which packages bdfrscrape and other tools in a more user-friendly form.

## Some Details

### bdfr

bdfr (the other tool) saves reddit posts and comments to disk. If a post is a non-text media post, bdfr saves the media itself to disk. 

### bdfrscrape

bdfrscrape processes bdfr output. It scrapes bdfr posts and comments for links to media, saves the media to disk, and rewrites the message bodies so that they link to the local copies of the media. It does not overwrite the bdfr output; it saves its output to a different directory (`dest_dir`). bdfrscrape furthers bdfr's goal of preserving reddit content by copying linked media and saving them so that the local user retains control.

### Example

The below example shows how to use bdfrscrape to process bdfr output.

1. First, run bdfr to clone a subreddit:
```
python -m bdfr clone /home/ccollins/tmp/bdfr-test --subreddit AskHistorians --sort new --stop-on-exist --config /home/ccollins/.config/mybdfrconfig.cfg --no-dupes
```
The subreddit has been cloned to `/home/ccollins/tmp/bdfr-test/AskHistorians`

2. Run bdfrscrape to process the output from step 1:
```
bdfrscrape -j 8 -v /home/ccollins/tmp/bdfr-test/AskHistorians /home/ccollins/tmp/scrape-test/AskHistorians
```
The processed content has been saved to `/home/ccollins/tmp/scrape-test/AskHistorians`
