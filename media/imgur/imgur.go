package imgur

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/ccollins476ad/bdfrscrape/download"
	"github.com/ccollins476ad/bdfrscrape/web"
	"github.com/koffeinsource/go-imgur"
	log "github.com/sirupsen/logrus"
)

const (
	clientID = "ab1802d70cb1deb"
)

var getHeader = http.Header{
	"Authorization": []string{"Client-ID " + clientID},
	"referer":       []string{"https://imgur.com/"},
	"origin":        []string{"https://imgur.com"},
	"content-type":  []string{"application/json"},
	"user-agent":    []string{"curl/7.84.0"},
}

type albumInfoDataWrapper struct {
	AI      *imgur.AlbumInfo `json:"data"`
	Success bool             `json:"success"`
	Status  int              `json:"status"`
}

// Downloader retrieves imgur images and albums from the web. It implements the
// media.Downloader interface.
type Downloader struct {
	s *download.Store
}

func NewDownloader(s *download.Store) *Downloader {
	return &Downloader{
		s: s,
	}
}

// Download retrieves imgur media from the given url. It can download albums
// and individual images. See media.Downloader#Download for API details.
func (dl *Downloader) Download(ctx context.Context, u string) (string, error) {
	// Album.
	if strings.HasPrefix(u, "https://imgur.com/a/") {
		return dl.downloadAlbum(ctx, u)
	}

	// Individual image.
	if strings.HasPrefix(u, "https://i.imgur.com/") {
		return dl.downloadImage(ctx, u)
	}

	// Alternate image url format:
	//     https://imgur.com/<image_id>
	imageID := strings.TrimPrefix(u, "https://imgur.com/")
	if len(imageID) == 7 {
		return dl.downloadImage(ctx, "https://i.imgur.com/"+imageID+".jpeg")
	}

	return "", nil
}

// albumLinks reads the imgur album at the specified url and returns the urls
// of all its images.
func albumLinks(ctx context.Context, hc *http.Client, u string) ([]string, error) {
	log.Debugf("scanning imgur album: %s", u)

	trimmed := strings.TrimPrefix(u, "https://imgur.com/a/")
	if len(trimmed) < 7 {
		return nil, fmt.Errorf("imgur album hash length too short: have=%d want=7 hash=%s", len(trimmed), trimmed)
	}
	if len(trimmed) > 7 {
		hash := trimmed[len(trimmed)-7:]
		log.Debugf("removing imgur album prefix: %s --> %s", trimmed, hash)
		trimmed = hash
	}

	u = "https://api.imgur.com/3/album/" + trimmed

	b, err := download.Get(ctx, hc, u, getHeader)
	if err != nil {
		return nil, err
	}

	aidw := &albumInfoDataWrapper{}
	err = json.Unmarshal(b, aidw)
	if err != nil {
		return nil, fmt.Errorf("failed to decode album info: %w", err)
	}

	if !aidw.Success {
		return nil, fmt.Errorf("album info response has success=false")
	}

	album := aidw.AI

	var links []string
	for _, img := range album.Images {
		log.Debugf("detected imgur album image link: %s", img.Link)
		links = append(links, img.Link)
	}

	return links, nil
}

// downloadImage downloads an individual imgur image from the given url.
func (dl *Downloader) downloadImage(ctx context.Context, u string) (string, error) {
	return dl.s.Download(ctx, u, getHeader)
}

// downloadImage downloads an imgur album from the given url. It downloads each
// constituent image, then builds an html gallery. It returns the path of the
// gallery.
func (dl *Downloader) downloadAlbum(ctx context.Context, albumURL string) (string, error) {
	desc, err := dl.s.EvaluateURL(albumURL)
	if err != nil {
		return "", err
	}

	if desc.IsLocal {
		// Already downloaded.
		return desc.Filename, nil
	}

	urls, err := albumLinks(ctx, dl.s.HTTPClient(), albumURL)
	if err != nil {
		return "", err
	}

	var filenames []string
	for _, u := range urls {
		filename, err := dl.downloadImage(ctx, u)
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
