package media

import "context"

// Downloader retrieves media from the web and saves it to disk. Most
// downloader implementations only know how to access a particular web site
// (e.g., imgurl).
type Downloader interface {
	// Download retrieves the media file at url=u and saves it to disk. It is a
	// no-op if the destination file already exists (i.e., has already been
	// downloaded). It returns the path of the saved file, relative to the
	// downloader's base directory.
	Download(ctx context.Context, u string) (string, error)
}
