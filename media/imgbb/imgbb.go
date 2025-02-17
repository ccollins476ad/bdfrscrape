package imgbb

import (
	"context"
	"fmt"
	"strings"

	"github.com/ccollins476ad/bdfrscrape/download"
	"github.com/ccollins476ad/bdfrscrape/web"
	"golang.org/x/net/html"
)

// Downloader retrieves imgbb images from the web. It implements the
// media.Downloader interface.
type Downloader struct {
	s *download.Store
}

func NewDownloader(s *download.Store) *Downloader {
	return &Downloader{
		s: s,
	}
}

// Download retrieves imgbb media from the given url. It can download albums
// and individual images. See media.Downloader#Download for API details.
func (dl *Downloader) Download(ctx context.Context, u string) (string, error) {
	if strings.HasPrefix(u, "https://ibb.co/album/") {
		return dl.downloadAlbum(ctx, u)
	}
	if strings.HasPrefix(u, "https://ibb.co/") {
		return dl.downloadImage(ctx, u)
	}
	return "", nil
}

// embeddedImageURLs reads the imgbb album at the specified url and returns the
// urls of all its images.
func embeddedImageURLs(doc *html.Node) []string {
	var urls []string

	rawURLs := web.EmbeddedImageURLs(doc)
	for _, ru := range rawURLs {
		if strings.HasPrefix(ru, "https://") {
			urls = append(urls, ru)
		}
	}

	return urls
}

// parseAlbum extracts the urls of all images from a imgbb album.
func parseAlbum(doc *html.Node) ([]string, error) {
	urls := embeddedImageURLs(doc)
	if len(urls) == 0 {
		return nil, fmt.Errorf("imgbb album contains 0 embedded image urls")
	}

	return urls, nil
}

// downloadImage downloads an imgbb album from the given url. It downloads each
// constituent image, then builds an html gallery. It returns the path of the
// gallery.
func (dl *Downloader) downloadAlbum(ctx context.Context, u string) (string, error) {
	desc, err := dl.s.EvaluateURL(u)
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

	links, err := parseAlbum(doc)
	if err != nil {
		return "", err
	}

	var filenames []string
	for _, link := range links {
		filename, err := dl.s.Download(ctx, link, nil)
		if err != nil {
			return "", fmt.Errorf("failed to save image belonging to imgbb album: image_url=%s err=%w", link, err)
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

// downloadImage downloads an individual imgbb image from the given url.
func (dl *Downloader) downloadImage(ctx context.Context, u string) (string, error) {
	desc, err := dl.s.EvaluateURL(u)
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

	imgURLs := web.EmbeddedImageURLs(doc)
	var targetURL string
	for _, iu := range imgURLs {
		if strings.HasPrefix(iu, "https://") {
			if targetURL != "" {
				return "", fmt.Errorf("imgbb page contains multiple image links: first=%s second=%s", targetURL, iu)
			}
			targetURL = iu
		}
	}
	if targetURL == "" {
		return "", fmt.Errorf("imgbb page lacks image link")
	}

	return dl.s.DownloadAs(ctx, targetURL, nil, desc.Filename)
}
