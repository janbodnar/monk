package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	colorable "github.com/mattn/go-colorable"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

func init() {
	color.Output = colorable.NewColorableStdout()
}

// isVoidElement checks if a node is a void element (self-closing tags).
func isVoidElement(n *html.Node) bool {
	switch n.DataAtom {
	case atom.Area, atom.Base, atom.Br, atom.Col, atom.Command, atom.Embed,
		atom.Hr, atom.Img, atom.Input, atom.Keygen, atom.Link,
		atom.Meta, atom.Param, atom.Source, atom.Track, atom.Wbr:
		return true
	}
	return false
}

var (
	tagColor     = color.New(color.FgCyan)
	tokenColor   = color.New(color.FgCyan)
	attrKeyColor = color.New(color.FgGreen)
	quoteColor   = color.New(color.FgBlue)
	commentColor = color.New(color.FgYellow)
)

// Displayer interface for outputting selected HTML nodes.
type Displayer interface {
	Display(nodes []*html.Node)
}

// TreeDisplayer outputs HTML nodes in a tree format.
type TreeDisplayer struct{}

// Display prints the HTML nodes in tree format.
func (t TreeDisplayer) Display(nodes []*html.Node) {
	for _, node := range nodes {
		t.printTreeNode(node, 0)
	}
}

func (t TreeDisplayer) printPreformatted(n *html.Node) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		if config.EscapeHTML {
			if n.Parent == nil || n.Parent.DataAtom != atom.Script {
				s = html.EscapeString(s)
			}
		}
		fmt.Print(s)
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			t.printPreformatted(c)
		}
	case html.ElementNode:
		fmt.Printf("<%s", n.Data)
		for _, a := range n.Attr {
			val := a.Val
			if config.EscapeHTML {
				val = html.EscapeString(val)
			}
			fmt.Printf(` %s="%s"`, a.Key, val)
		}
		fmt.Print(">")
		if !isVoidElement(n) {
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				t.printPreformatted(c)
			}
			fmt.Printf("</%s>", n.Data)
		}
	case html.CommentNode:
		data := n.Data
		if config.EscapeHTML {
			data = html.EscapeString(data)
		}
		fmt.Printf("<!--%s-->", data)
		if !config.RawOutput {
			fmt.Println()
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			t.printPreformatted(c)
		}
	case html.DoctypeNode, html.DocumentNode:
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			t.printPreformatted(c)
		}
	}
}

func (t TreeDisplayer) printTreeNode(n *html.Node, level int) {
	switch n.Type {
	case html.TextNode:
		s := n.Data
		if config.EscapeHTML {
			if n.Parent == nil || n.Parent.DataAtom != atom.Script {
				s = html.EscapeString(s)
			}
		}
		if config.RawOutput {
			fmt.Print(s)
		} else {
			s = strings.TrimSpace(s)
			if s != "" {
				t.printIndent(level)
				fmt.Println(s)
			}
		}
	case html.ElementNode:
		if !config.RawOutput {
			t.printIndent(level)
		}
		if n.DataAtom == atom.Pre && !config.PrintColor && config.Preformatted {
			t.printPreformatted(n)
			if !config.RawOutput {
				fmt.Println()
			}
			return
		}
		if config.PrintColor {
			tokenColor.Print("<")
			tagColor.Printf("%s", n.Data)
		} else {
			fmt.Printf("<%s", n.Data)
		}
		for _, a := range n.Attr {
			val := a.Val
			if config.EscapeHTML {
				val = html.EscapeString(val)
			}
			if config.PrintColor {
				fmt.Print(" ")
				attrKeyColor.Printf("%s", a.Key)
				tokenColor.Print("=")
				quoteColor.Printf(`"%s"`, val)
			} else {
				fmt.Printf(` %s="%s"`, a.Key, val)
			}
		}
		if config.PrintColor {
			tokenColor.Print(">")
		} else {
			fmt.Print(">")
		}
		if !config.RawOutput {
			fmt.Println()
		}
		if !isVoidElement(n) {
			t.printChildren(n, level+1)
			if !config.RawOutput {
				t.printIndent(level)
			}
			if config.PrintColor {
				tokenColor.Print("</")
				tagColor.Printf("%s", n.Data)
				tokenColor.Print(">")
			} else {
				fmt.Printf("</%s>", n.Data)
			}
			if !config.RawOutput {
				fmt.Println()
			}
		}
	case html.CommentNode:
		if !config.RawOutput {
			t.printIndent(level)
		}
		data := n.Data
		if config.EscapeHTML {
			data = html.EscapeString(data)
		}
		if config.PrintColor {
			commentColor.Printf("<!--%s-->", data)
		} else {
			fmt.Printf("<!--%s-->", data)
		}
		if !config.RawOutput {
			fmt.Println()
		}
		t.printChildren(n, level)
	case html.DoctypeNode, html.DocumentNode:
		t.printChildren(n, level)
	}
}

func (t TreeDisplayer) printChildren(n *html.Node, level int) {
	if config.MaxPrintLevel > -1 && level >= config.MaxPrintLevel {
		t.printIndent(level)
		fmt.Println("...")
		return
	}
	child := n.FirstChild
	for child != nil {
		t.printTreeNode(child, level)
		child = child.NextSibling
	}
}

func (t TreeDisplayer) printIndent(level int) {
	for ; level > 0; level-- {
		fmt.Print(config.IndentString)
	}
}

// TextDisplayer outputs the text content of HTML nodes.
type TextDisplayer struct{}

func (t TextDisplayer) Display(nodes []*html.Node) {
	for _, node := range nodes {
		if node.Type == html.TextNode {
			data := node.Data
			if config.EscapeHTML {
				if node.Parent == nil || node.Parent.DataAtom != atom.Script {
					data = html.EscapeString(data)
				}
			}
			fmt.Println(data)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			t.Display([]*html.Node{child})
		}
	}
}

// AttrDisplayer outputs the value of a specified attribute from HTML nodes.
type AttrDisplayer struct {
	Attr string
}

func (a AttrDisplayer) Display(nodes []*html.Node) {
	for _, node := range nodes {
		for _, attr := range node.Attr {
			if attr.Key == a.Attr {
				val := attr.Val
				if config.EscapeHTML {
					val = html.EscapeString(val)
				}
				fmt.Print(val)
				if !config.RawOutput {
					fmt.Println()
				}
			}
		}
	}
}

// JSONDisplayer outputs HTML nodes as JSON.
type JSONDisplayer struct{}

func jsonify(node *html.Node) map[string]any {
	vals := map[string]any{}
	if len(node.Attr) > 0 {
		for _, attr := range node.Attr {
			if config.EscapeHTML {
				vals[attr.Key] = html.EscapeString(attr.Val)
			} else {
				vals[attr.Key] = attr.Val
			}
		}
	}
	vals["tag"] = node.DataAtom.String()
	children := []any{}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case html.ElementNode:
			children = append(children, jsonify(child))
		case html.TextNode:
			text := strings.TrimSpace(child.Data)
			if text != "" {
				if config.EscapeHTML {
					if node.DataAtom != atom.Script {
						text = html.EscapeString(text)
					}
				}
				currText, ok := vals["text"]
				if ok {
					text = fmt.Sprintf("%s %s", currText, text)
				}
				vals["text"] = text
			}
		case html.CommentNode:
			comment := strings.TrimSpace(child.Data)
			if config.EscapeHTML {
				comment = html.EscapeString(comment)
			}
			currComment, ok := vals["comment"]
			if ok {
				comment = fmt.Sprintf("%s %s", currComment, comment)
			}
			vals["comment"] = comment
		}
	}
	if len(children) > 0 {
		vals["children"] = children
	}
	return vals
}

func (j JSONDisplayer) Display(nodes []*html.Node) {
	jsonNodes := []map[string]any{}
	for _, node := range nodes {
		jsonNodes = append(jsonNodes, jsonify(node))
	}
	data, err := json.MarshalIndent(&jsonNodes, "", config.IndentString)
	if err != nil {
		panic("Could not jsonify nodes")
	}
	os.Stdout.Write(data)
	if !config.RawOutput {
		fmt.Println()
	}
}

// CountDisplayer outputs the count of selected nodes.
type CountDisplayer struct{}

func (d CountDisplayer) Display(nodes []*html.Node) {
	fmt.Println(len(nodes))
}
