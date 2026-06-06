package postimg

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ccollins476ad/bdfrscrape/download"
	"github.com/ccollins476ad/bdfrscrape/util"
	"github.com/ccollins476ad/bdfrscrape/web"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/html"
)

var linkRegexp = regexp.MustCompile(`background-image:url\('(https://i.postimg.cc/[^']+)'\)`)

type ImageLink struct {
	ShortName string // name on disk
	FullName  string // url
}

func (il *ImageLink) IsPopulated() bool {
	return il.ShortName != "" && il.FullName != ""
}

// Downloader retrieves postimg albums from the web. It implements the
// media.Downloader interface.
type Downloader struct {
	s *download.Store
}

func NewDownloader(s *download.Store) *Downloader {
	return &Downloader{
		s: s,
	}
}

// Download retrieves postimg albums from the given url. See
// media.Downloader#Download for API details.
func (dl *Downloader) Download(ctx context.Context, u string) (string, error) {
	logger := dl.s.Logger().WithFields(log.Fields{
		util.LogFieldDownloader: "postimg",
		util.LogFieldTopURL:     u,
	})

	logger.Tracef("Download(): u=%s", u)

	if strings.HasPrefix(u, "https://postimg.cc/gallery/") {
		return dl.downloadAlbum(ctx, logger, u)
	}

	if strings.HasPrefix(u, "https://postimg.cc/") {
		return dl.downloadSingleImagePage(ctx, logger, u)
	}

	return "", nil
}

// parseAlbum extracts the urls of all images from a postimg album.
func parseAlbum(logger *log.Entry, doc *html.Node) ([]ImageLink, error) {
	var links []ImageLink

	urls := extractImageURLs(logger, doc)
	for _, u := range urls {
		l := ImageLink{
			ShortName: u,
			FullName:  u,
		}
		links = append(links, l)
	}
	return links, nil
}

func extractImageURLs(logger *log.Entry, node *html.Node) []string {
	var inner func(n *html.Node, callDepth int) []string

	inner = func(n *html.Node, callDepth int) []string {
		if callDepth >= 10 {
			logger.Errorf("aborting album crawl due to excessive recursion: depth=%d", callDepth)
			return nil
		}

		var urls []string

		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				// This is just the attribute that postimg seems to use for album images.
				if attr.Key == "data-pswp-src" {
					urls = append(urls, attr.Val)
				}
			}
		}

		// Recursively traverse sibling and child HTML nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			childURLs := inner(c, callDepth+1)
			urls = append(urls, childURLs...)
		}

		return urls
	}

	return inner(node, 0)
}

// downloadImage downloads an individual postimg image from the given url.
func (dl *Downloader) downloadImage(ctx context.Context, il ImageLink) (string, error) {
	filename, err := download.URLToFilename(il.ShortName)
	if err != nil {
		return "", err
	}
	return dl.s.DownloadAs(ctx, il.FullName, nil, filename)
}

// downloadAlbum downloads a postimg album from the given url. It downloads
// each constituent image, then builds an html gallery. It returns the path of
// the gallery.
func (dl *Downloader) downloadAlbum(ctx context.Context, logger *log.Entry, albumURL string) (string, error) {
	desc, err := dl.s.EvaluateURL(albumURL, "")
	if err != nil {
		return "", err
	}

	if desc.IsLocal {
		// Already downloaded.
		return desc.Filename, nil
	}

	body, err := download.GetBody(ctx, dl.s.HTTPClient(), albumURL, nil)
	if err != nil {
		return "", err
	}
	defer body.Close()

	doc, err := html.Parse(download.NewContextReader(ctx, body))
	if err != nil {
		return "", err
	}

	links, err := parseAlbum(logger, doc)
	if err != nil {
		return "", err
	}

	var filenames []string
	for _, l := range links {
		filename, err := dl.downloadImage(ctx, l)
		if err != nil {
			return "", err
		}
		filenames = append(filenames, filename)
	}

	gallery := web.BuildGallery(filenames)

	err = dl.s.SaveFile(desc.Filename, []byte(gallery))
	if err != nil {
		return "", err
	}

	return desc.Filename, nil
}

func (dl *Downloader) downloadSingleImagePage(ctx context.Context, logger *log.Entry, u string) (string, error) {
	desc, err := dl.s.EvaluateURL(u, "")
	if err != nil {
		return "", err
	}

	if desc.IsLocal {
		// Already downloaded.
		return desc.Filename, nil
	}

	body, err := download.GetBody(ctx, dl.s.HTTPClient(), u, nil)
	if err != nil {
		return "", err
	}
	defer body.Close()

	doc, err := html.Parse(download.NewContextReader(ctx, body))
	if err != nil {
		return "", err
	}

	imageURLs := web.EmbeddedImageURLs(doc)
	if len(imageURLs) == 0 {
		return "", fmt.Errorf("single image postimg page does not contain an image url")
	}

	return dl.s.DownloadAs(ctx, imageURLs[0], nil, desc.Filename)
}
