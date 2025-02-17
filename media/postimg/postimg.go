package postimg

import (
	"context"
	"regexp"
	"strings"

	"github.com/ccollins476ad/bdfrscrape/download"
	"github.com/ccollins476ad/bdfrscrape/web"
	"golang.org/x/net/html"
)

var linkRegexp = regexp.MustCompile(`background-image:url\('(https://i.postimg.cc/[^']+)'\)`)

type ImageLink struct {
	ShortName string
	FullName  string
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
	if strings.HasPrefix(u, "https://postimg.cc/gallery/") {
		return dl.downloadAlbum(ctx, u)
	}
	return "", nil
}

// parseAlbum extracts the urls of all images from a postimg album.
func parseAlbum(doc *html.Node) ([]ImageLink, error) {
	var links []ImageLink

	web.ForEachLink(doc, func(n *html.Node) error {
		var link ImageLink

		for _, a := range n.Attr {
			switch a.Key {
			case "href":
				link.ShortName = a.Val

			case "style":
				matches := linkRegexp.FindStringSubmatch(a.Val)
				if len(matches) > 0 {
					link.FullName = matches[1]
				}
			}
		}

		if link.IsPopulated() {
			links = append(links, link)
		}

		return nil
	})

	return links, nil
}

// downloadImage downloads an individual postimg image from the given url.
func (dl *Downloader) downloadImage(ctx context.Context, il ImageLink) (string, error) {
	filename, err := download.URLToFilename(il.ShortName)
	if err != nil {
		return "", err
	}
	return dl.s.DownloadAs(ctx, il.FullName, nil, filename)
}

// downloadImage downloads a postimg album from the given url. It downloads
// each constituent image, then builds an html gallery. It returns the path of
// the gallery.
func (dl *Downloader) downloadAlbum(ctx context.Context, albumURL string) (string, error) {
	desc, err := dl.s.EvaluateURL(albumURL)
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

	links, err := parseAlbum(doc)
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
