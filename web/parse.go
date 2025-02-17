package web

import (
	"golang.org/x/net/html"
)

// extractLinkFromNode returns the href anchor text associated with the given
// html node. It returns the empty string if the node is not a link.
func extractLinkFromNode(n *html.Node) string {
	if n.Type != html.ElementNode || n.Data != "a" {
		return ""
	}

	for _, a := range n.Attr {
		if a.Key == "href" {
			return a.Val
		}
	}

	return ""
}

// ForEachNode applies a function to the given node and each of its
// descendants.
func ForEachNode(node *html.Node, fn func(n *html.Node) error) error {
	var iter func(n *html.Node) error
	iter = func(n *html.Node) error {
		err := fn(n)
		if err != nil {
			return err
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			err := iter(c)
			if err != nil {
				return err
			}
		}

		return nil
	}

	return iter(node)
}

// ForEachNode applies a function to each `a href` element in the given html
// node and its descendants.
func ForEachLink(node *html.Node, fn func(n *html.Node) error) error {
	return ForEachNode(node, func(n *html.Node) error {
		if extractLinkFromNode(n) != "" {
			return fn(n)
		}
		return nil
	})
}

// NodesWithDataVal returns a slice of all descendant nodes whose "data" field
// has the given value.
func NodesWithDataVal(node *html.Node, dataName string) []*html.Node {
	var nodes []*html.Node

	ForEachNode(node, func(n *html.Node) error {
		if n.Type == html.ElementNode && n.Data == dataName {
			nodes = append(nodes, n)
		}
		return nil
	})

	return nodes
}

// EmbeddedImageURLs returns a slice of all image URLs embedded in the given
// html document.
func EmbeddedImageURLs(doc *html.Node) []string {
	nodes := NodesWithDataVal(doc, "img")

	var urls []string
	for _, n := range nodes {
		for _, a := range n.Attr {
			if a.Key == "src" {
				urls = append(urls, a.Val)
				break
			}
		}
	}

	return urls
}
