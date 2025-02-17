package web

import (
	"fmt"
	"strings"
)

// BuildGallery constructs an html web page displaying images with the given
// filenames.
func BuildGallery(filenames []string) string {
	sb := strings.Builder{}

	sb.WriteString(`<!DOCTYPE html>
<html>
<body>
`)

	for _, f := range filenames {
		sb.WriteString(fmt.Sprintf("<img src=\"%s\" alt=\"%s\" style=\"background-size:100% 100%\">\n", f, f))
	}

	sb.WriteString(`</body>
</html> 
`)

	return sb.String()
}
