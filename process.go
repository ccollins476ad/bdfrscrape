package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ccollins476ad/bdfrscrape/bdfr"
	"github.com/ccollins476ad/bdfrscrape/download"
	"github.com/ccollins476ad/bdfrscrape/media"
	"github.com/ccollins476ad/bdfrscrape/media/imgbb"
	"github.com/ccollins476ad/bdfrscrape/media/imgur"
	"github.com/ccollins476ad/bdfrscrape/media/postimg"
	log "github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"mvdan.cc/xurls/v2"
)

// processFiles calls processFile() for each filename in the given slice. It
// processes the files in parallel, cfg.Jobs goroutines.
func processFiles(ctx context.Context, cfg *Config, filenames []string) error {
	s := download.NewStore(cfg.DestDir)
	g := &errgroup.Group{}

	startGoroutines := func() {
		filenameChan := make(chan string)
		defer close(filenameChan)

		// Create a set of goroutines to process posts in parallel.
		for i := 0; i < cfg.Jobs; i++ {
			g.Go(func() error {
				// Read filenames from the channel and process them
				// sequentially. Proceed until error or channel closed.
				for filename := range filenameChan {
					err := processFile(ctx, cfg, s, filename)
					if err != nil {
						return err
					}
				}
				return nil
			})
		}

		// Process bdfr posts.
		for _, filename := range filenames {
			select {
			case <-ctx.Done():
				// Operation aborted. Return early to execute deferred channel
				// close.
				return

			case filenameChan <- filename:
			}
		}
	}

	startGoroutines()

	return g.Wait()
}

// processFile reads the given saved bdfr post from disk, processes it with
// processPost(), and writes the processed content to disk in the configured
// destination directory.
func processFile(ctx context.Context, cfg *Config, s *download.Store, filename string) error {
	m, err := bdfr.ReadMessage(cfg.Source + "/" + filename)
	if err != nil {
		return err
	}

	log.Debugf("processing post: filename=%s", filename)
	err = processPost(ctx, s, m)
	if err != nil {
		return err
	}

	b, err := json.Marshal(m)
	if err != nil {
		return err
	}

	err = os.WriteFile(cfg.DestDir+"/"+filename, b, 0644)
	if err != nil {
		return err
	}

	return nil
}

// processPost saves external media referenced by the given bdfr post and its
// comments, then updates the message bodies such that they link to the local
// media instead. That is, it makes a given reddit post fully self-contained
// and localized.
func processPost(ctx context.Context, s *download.Store, m bdfr.Message) error {
	selftext := m.GetString("selftext")
	m.SetString("selftext", processBody(ctx, s, selftext))

	comments, err := m.GetSliceOfMessages("comments")
	if err != nil {
		return err
	}

	for _, c := range comments {
		processComment(ctx, s, c)
	}

	return nil
}

// processComment saves external media referenced by the given bdfr comment,
// then updates its message body such that it links to the local media instead.
func processComment(ctx context.Context, s *download.Store, c bdfr.Message) error {
	body := processBody(ctx, s, c.GetString("body"))
	c.SetString("body", body)

	replies, err := c.GetSliceOfMessages("replies")
	if err != nil {
		return err
	}

	for _, r := range replies {
		err := processComment(ctx, s, r)
		if err != nil {
			return err
		}
	}

	return nil
}

// processBody saves external media referenced in the given post or comment
// body, then updates the body such that it links to the local media instead.
// It returns the modified message body.
func processBody(ctx context.Context, s *download.Store, body string) string {
	processLink := func(link string) {
		localPath, err := downloadMedia(ctx, s, link)
		if err != nil {
			log.WithError(err).Errorf("failed to save link: link=%s", link)
			return
		}
		if localPath == "" {
			// Don't know how to save this link to disk. Ignore.
			return
		}

		// mdlink is the the url of the local copy of the media file.
		mdlink := "media/" + localPath

		// Update the link in the message body to point to the local media
		modded := strings.Replace(body, "]("+link+")", "]("+mdlink+")", -1)
		if modded != body {
			// Message contains an embedded markdown link to the media file.
			log.Debugf("replacing markdown link: (%s) --> (%s)", link, mdlink)
			body = modded
		} else {
			// Message contains a raw url.
			rawlink := fmt.Sprintf(`<a href="media/%s">%s</a>`, localPath, link)
			log.Debugf("replacing raw link: %s --> %s", link, rawlink)
			body = strings.Replace(body, link, rawlink, -1)
		}
	}

	rx := xurls.Strict()
	links := rx.FindAllString(body, -1)

	for _, link := range links {
		processLink(link)
	}

	return body
}

// downloadMedia attempts to download the media file specified by the given
// url. On success, it returns the local path of saved file, relative to
// mdfrscrape's media directory. It is a no-op that appears successful if there
// is already a file with the destination path (e.g., a previous invocation of
// the tool already saved the file). It returns the empty string if it does
// not know how to save the given url. It returns an error if it attempts and
// fails to save the specified media file.
func downloadMedia(ctx context.Context, s *download.Store, u string) (string, error) {
	dls := []media.Downloader{
		imgur.NewDownloader(s),
		postimg.NewDownloader(s),
		imgbb.NewDownloader(s),
	}

	dlOnce := func(dl media.Downloader) (string, error) {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		return dl.Download(ctx, u)
	}

	for _, dl := range dls {
		filename, err := dlOnce(dl)
		if filename != "" || err != nil {
			return filename, err
		}
	}
	return "", nil
}
