package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

var (
	reContains = regexp.MustCompile(`:contains\("((?:[^"\\]|\\.)*)"\)`)
	reMatches  = regexp.MustCompile(`:matches\("((?:[^"\\]|\\.)*)"\)`)
	reParentOf = regexp.MustCompile(`:parent-of\(([^)]*)\)`)
)

// extractDisplayFunction scans command tokens for display function tokens
// (attr{...}, text{}, json{}, --number, -n) and returns the filtered token list
// along with the appropriate Displayer. If no display token is found, returns nil.
func extractDisplayFunction(cmds []string) ([]string, Displayer) {
	var displayCmd Displayer
	var filtered []string
	for _, cmd := range cmds {
		switch {
		case strings.HasPrefix(cmd, "attr{") && strings.HasSuffix(cmd, "}"):
			attr := cmd[5 : len(cmd)-1]
			displayCmd = AttrDisplayer{Attr: attr}
		case cmd == "text{}":
			displayCmd = TextDisplayer{}
		case cmd == "json{}":
			displayCmd = JSONDisplayer{}
		case cmd == "--number" || cmd == "-n":
			displayCmd = CountDisplayer{}
		default:
			filtered = append(filtered, cmd)
		}
	}
	return filtered, displayCmd
}

// extractCustomPseudo extracts non-standard pseudo-classes (:contains, :matches, :parent-of)
// from a CSS selector string. Returns the remaining goquery-compatible selector
// and a list of node filter functions to apply afterwards.
func extractCustomPseudo(selector string) (string, []func(*html.Node) bool, error) {
	base := selector
	var filters []func(*html.Node) bool

	// Extract :contains("...")
	if matches := reContains.FindStringSubmatch(base); len(matches) == 2 {
		text := unescapeQuotedString(matches[1])
		filters = append(filters, func(n *html.Node) bool {
			return nodeContainsText(n, text)
		})
		base = reContains.ReplaceAllString(base, "")
	}

	// Extract :matches("...")
	if matches := reMatches.FindStringSubmatch(base); len(matches) == 2 {
		pattern := unescapeQuotedString(matches[1])
		re2, err := regexp.Compile(pattern)
		if err != nil {
			return "", nil, fmt.Errorf("invalid regex in :matches(): %w", err)
		}
		filters = append(filters, func(n *html.Node) bool {
			return nodeMatchesText(n, re2)
		})
		base = reMatches.ReplaceAllString(base, "")
	}

	// Extract :parent-of(...)
	if matches := reParentOf.FindStringSubmatch(base); len(matches) == 2 {
		innerSelector := matches[1]
		filters = append(filters, func(n *html.Node) bool {
			return nodeHasChildMatching(n, innerSelector)
		})
		base = reParentOf.ReplaceAllString(base, "")
	}

	return base, filters, nil
}

// unescapeQuotedString converts an escaped string (from inside quotes) back to its literal form.
func unescapeQuotedString(s string) string {
	s = strings.ReplaceAll(s, `\"`, `"`)
	s = strings.ReplaceAll(s, `\\`, `\`)
	return s
}

// nodeContainsText checks whether the node has a direct text child containing text.
func nodeContainsText(n *html.Node, text string) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.Contains(c.Data, text) {
			return true
		}
	}
	return false
}

// nodeMatchesText checks whether the node has a direct text child matching the regex.
func nodeMatchesText(n *html.Node, re *regexp.Regexp) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && re.MatchString(c.Data) {
			return true
		}
	}
	return false
}

// nodeHasChildMatching checks if the node has a direct child element matching the given CSS selector.
func nodeHasChildMatching(n *html.Node, selector string) bool {
	sel := goquery.NewDocumentFromNode(n)
	return sel.Children().Filter(selector).Length() > 0
}

// applyWithMode applies a CSS selector to a goquery Selection using the specified traversal mode.
//
//	mode ""  = descendant selector (uses Find)
//	mode ">" = direct child selector (uses ChildrenFiltered)
//	mode "+" = adjacent sibling selector (uses NextFiltered)
func applyWithMode(sel *goquery.Selection, selector string, mode string) (*goquery.Selection, error) {
	baseSelector, filters, err := extractCustomPseudo(selector)
	if err != nil {
		return nil, err
	}

	if baseSelector == "" {
		baseSelector = "*"
	}

	var newSel *goquery.Selection
	switch mode {
	case "":
		newSel = sel.Find(baseSelector)
	case ">":
		newSel = sel.ChildrenFiltered(baseSelector)
	case "+":
		newSel = sel.NextFiltered(baseSelector)
	}

	// Apply custom pseudo-class filters
	for _, filter := range filters {
		newSel = newSel.FilterFunction(func(i int, s *goquery.Selection) bool {
			if len(s.Nodes) == 0 {
				return false
			}
			return filter(s.Nodes[0])
		})
	}

	return newSel, nil
}

// applyHead returns the first n elements of the selection.
func applyHead(sel *goquery.Selection, n int) *goquery.Selection {
	if n >= sel.Length() {
		return sel
	}
	return sel.Slice(0, n)
}

// applyTail returns the last n elements of the selection.
func applyTail(sel *goquery.Selection, n int) *goquery.Selection {
	if n >= sel.Length() {
		return sel
	}
	return sel.Slice(sel.Length()-n, sel.Length())
}

// parseHeadTail extracts the integer argument from a head(n) or tail(n) token.
func parseHeadTail(cmd string) (int, error) {
	open := strings.IndexRune(cmd, '(')
	closeParen := strings.IndexRune(cmd, ')')
	if open < 0 || closeParen < 0 || closeParen <= open+1 {
		return 0, fmt.Errorf("invalid head/tail selector: %s", cmd)
	}
	n, err := strconv.Atoi(cmd[open+1 : closeParen])
	if err != nil {
		return 0, err
	}
	if n < 0 {
		return 0, fmt.Errorf("argument to head/tail must be non-negative")
	}
	return n, nil
}

// combineSelections merges multiple goquery Selections into a single slice of *html.Node.
func combineSelections(selections []*goquery.Selection) []*html.Node {
	var allNodes []*html.Node
	for _, sel := range selections {
		if sel == nil {
			continue
		}
		sel.Each(func(i int, s *goquery.Selection) {
			allNodes = append(allNodes, s.Nodes...)
		})
	}
	return allNodes
}
