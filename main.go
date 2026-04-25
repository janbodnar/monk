package main

// monk is a command-line tool for parsing HTML and applying CSS selectors.

import (
	"fmt"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var version = "0.1.0"

func main() {
	// process flags and arguments
	cmds, err := ParseArgs()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}
	defer config.Input.Close()

	// Extract display function tokens (attr{}, text{}, json{}, --number)
	cmds, displayFromCmd := extractDisplayFunction(cmds)
	if displayFromCmd != nil {
		// Override flag-based displayer with command-based displayer
		config.Displayer = displayFromCmd
	}

	// Parse the input using goquery, handling charset detection
	doc, err := ParseDocument(config.Input, config.Charset, config.ContentType)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(2)
	}

	// Split commands by comma into groups (each group is OR'd together)
	commandGroups := splitByComma(cmds)

	// Apply each group of selectors and collect results
	var allNodes []*goquery.Selection
	for _, group := range commandGroups {
		nodes, err := applyCommandGroup(doc, group)
		if err != nil {
			fmt.Fprintf(os.Stderr, "selector parsing error: %v\n", err)
			os.Exit(2)
		}
		allNodes = append(allNodes, nodes)
	}

	// Convert goquery selections to html.Node slice and display
	finalNodes := combineSelections(allNodes)
	config.Displayer.Display(finalNodes)
}

// splitByComma splits a command token list into groups separated by commas.
func splitByComma(cmds []string) [][]string {
	var groups [][]string
	var current []string
	for _, cmd := range cmds {
		if cmd == "," {
			if len(current) > 0 {
				groups = append(groups, current)
			}
			current = nil
		} else {
			current = append(current, cmd)
		}
	}
	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

// applyCommandGroup applies a sequence of CSS selectors to a goquery document.
// The first selector starts from the document root; subsequent selectors
// chain off the previous results. Supports descendant (""), child (">"),
// and adjacent sibling ("+") traversal modes.
func applyCommandGroup(doc *goquery.Document, cmds []string) (*goquery.Selection, error) {
	if len(cmds) == 0 {
		return doc.Selection, nil
	}

	// Start with document root
	sel := doc.Selection

	for i := 0; i < len(cmds); {
		cmd := cmds[i]

		// Handle traversal mode operators
		mode := ""
		switch cmd {
		case ">":
			mode = ">"
			i++
			if i >= len(cmds) {
				return nil, fmt.Errorf("expected selector after '>'")
			}
			cmd = cmds[i]
		case "+":
			mode = "+"
			i++
			if i >= len(cmds) {
				return nil, fmt.Errorf("expected selector after '+'")
			}
			cmd = cmds[i]
		}

		// Handle head/tail selectors
		if strings.HasPrefix(cmd, "head(") || strings.HasPrefix(cmd, "tail(") {
			n, err := parseHeadTail(cmd)
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(cmd, "head(") {
				sel = applyHead(sel, n)
			} else {
				sel = applyTail(sel, n)
			}
			i++
			continue
		}

		// Apply selector with current traversal mode
		var err error
		sel, err = applyWithMode(sel, cmd, mode)
		if err != nil {
			return nil, err
		}
		i++
	}

	return sel, nil
}
