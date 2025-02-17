package download

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/ccollins476ad/bdfrscrape/fileutil"
	"github.com/flytam/filenamify"
	log "github.com/sirupsen/logrus"
)

var AlreadyAttempted = errors.New("download already attempted")

// Store downloads media linked to by bdfr messages.
type Store struct {
	destDir string // constant

	hc *http.Client

	seenMtx sync.Mutex          // Protects the "seen" field.
	seen    map[string]struct{} // Media URLs we have already seen.
}

// Desc decribes a media file.
type Desc struct {
	Filename string // Relative to destination directory
	IsLocal  bool   // True if file already downloaded
}

func NewStore(destDir string) *Store {
	return &Store{
		destDir: destDir,
		hc:      &http.Client{},
		seen:    map[string]struct{}{},
	}
}

// EvaluateURL returns a descriptor for the media file that the given url
// points to. It does not download anything. The `IsLocal` field in the
// descriptor is true if the file has already been downloaded.
func (s *Store) EvaluateURL(u string) (*Desc, error) {
	filename, err := URLToFilename(u)
	if err != nil {
		log.WithError(err).Errorf("failed to convert url to filename: url=%s", u)
		return nil, err
	}

	destPath := s.destDir + "/" + filename
	if fileutil.FileExists(destPath) {
		log.Debugf("skipping %s: file already exists: %s", u, destPath)
		return &Desc{
			Filename: filename,
			IsLocal:  true,
		}, nil
	}

	already := s.see(u)
	if already {
		return nil, AlreadyAttempted
	}

	return &Desc{
		Filename: filename,
		IsLocal:  false,
	}, nil
}

func (s *Store) SaveFile(relPath string, b []byte) error {
	destPath := s.destDir + "/" + relPath
	log.Infof("downloading %s", destPath)
	return os.WriteFile(destPath, b, 0644)
}

// DownloadAs ensures the given media file has been downloaded. It downloads
// the file if it is not already on disk. The filename parameter specifies the
// local path of the file, relative to the configured bdfrscrape destination
// directory. It infers the path from the url if filename is "". It returns the
// local path of the media file, relative to the configured destination
// directory.
func (s *Store) DownloadAs(ctx context.Context, u string, header http.Header, filename string) (string, error) {
	desc, err := s.EvaluateURL(u)
	if err != nil {
		return "", err
	}

	if desc.IsLocal {
		// Already downloaded.
		return desc.Filename, nil
	}

	b, err := Get(ctx, s.hc, u, header)
	if err != nil {
		return "", err
	}

	if filename == "" {
		filename = desc.Filename
	}

	err = s.SaveFile(filename, b)
	if err != nil {
		return "", fmt.Errorf("failed to save http response: %v", err)
	}

	return filename, nil
}

// DownloadAs ensures the given media file has been downloaded. It downloads
// the file if it is not already on disk. It returns the local path of the
// media file, relative to the configured destination directory.
func (s *Store) Download(ctx context.Context, u string, header http.Header) (string, error) {
	return s.DownloadAs(ctx, u, header, "")
}

// HTTPClient returns the store's http client.
func (s *Store) HTTPClient() *http.Client {
	return s.hc
}

// see returns true if the media save has already attempted to download the
// specified media url. Otherwise, it marks the url as "seen" and returns
// false.
func (s *Store) see(u string) bool {
	s.seenMtx.Lock()
	defer s.seenMtx.Unlock()

	_, ok := s.seen[u]
	if ok {
		return true
	}

	s.seen[u] = struct{}{}
	return false
}

// URLToFilename returns the local filename that bdfrscrape would use to save
// the given media url.
func URLToFilename(u string) (string, error) {
	body, err := filenamify.Filenamify(u, filenamify.Options{})
	if err != nil {
		return "", err
	}
	return "_bdfrscrape_" + body, nil
}
