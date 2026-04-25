package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	charsetPkg "golang.org/x/net/html/charset"
	"golang.org/x/text/transform"
)

// Flag variables
var (
	flagColor   = flag.Bool("c", false, "print result with color")
	flagFile    = flag.String("f", "", "file to read from")
	flagURL     = flag.String("u", "", "URL to fetch HTML from")
	flagIndent  = flag.String("i", "", "number of spaces to use for indent or character")
	flagNumber  = flag.Bool("n", false, "print number of elements selected")
	flagLimit   = flag.Int("l", -1, "restrict number of levels printed")
	flagPlain   = flag.Bool("p", false, "don't escape html")
	flagRaw     = flag.Bool("r", false, "raw output")
	flagPre     = flag.Bool("pre", false, "preserve preformatted text")
	flagCharset = flag.String("charset", "", "specify the charset for monk to use")
	flagJSON    = flag.Bool("json", false, "output in JSON format")
	flagText    = flag.Bool("text", false, "output as plain text")
	flagAttr    = flag.String("attr", "", "output the value of the specified attribute")
	flagV       = flag.Bool("v", false, "display version")
	flagVersion = flag.Bool("version", false, "display version")
)

func init() {
	flag.Usage = func() {
		ShowHelp(os.Stdout)
		os.Exit(0)
	}
}

// ParseDocument parses HTML from the given reader, returning a goquery Document.
// contentType is the HTTP Content-Type header value (may be empty).
func ParseDocument(r io.Reader, charset string, contentType string) (*goquery.Document, error) {
	var err error
	if charset == "" {
		// attempt to guess the charset of the HTML document.
		// charsetPkg inspects meta tags/headers and returns a reader
		// that decodes the input to UTF-8, which is what goquery expects.
		r, err = charsetPkg.NewReader(r, contentType)
		if err != nil {
			return nil, err
		}
	} else {
		// let the user specify the charset
		e, name := charsetPkg.Lookup(charset)
		if name == "" {
			return nil, fmt.Errorf("'%s' is not a valid charset", charset)
		}
		r = transform.NewReader(r, e.NewDecoder())
	}
	return goquery.NewDocumentFromReader(r)
}

// ShowHelp displays the usage information.
func ShowHelp(w io.Writer) {
	helpString := `Usage
    monk [flags] [selectors]
Version
    %s
Flags
    -c --color         print result with color
    -f --file          file to read from
    -u --url           URL to fetch HTML from
    -i --indent        number of spaces to use for indent or character
    -n --number        print number of elements selected
    -l --limit         restrict number of levels printed
    -p --plain         don't escape html
    -r --raw           raw output
    --pre              preserve preformatted text
    --charset          specify the charset for monk to use
    --json             output in JSON format
    --text             output as plain text
    --attr <name>      output the value of the specified attribute
    -v --version       display version
`
	fmt.Fprintf(w, helpString, version)
}

// ParseArgs parses command-line flags and arguments, setting global variables and returning selectors.
func ParseArgs() ([]string, error) {
	flag.Parse()

	// Handle version
	if *flagV || *flagVersion {
		fmt.Println(version)
		os.Exit(0)
	}

	// If no arguments, show help
	if len(flag.Args()) == 0 {
		ShowHelp(os.Stdout)
		os.Exit(0)
	}

	// Set config variables from flags
	if *flagFile != "" && *flagURL != "" {
		return []string{}, fmt.Errorf("-f and -u are mutually exclusive")
	}
	if *flagFile != "" {
		var err error
		config.Input, err = os.Open(*flagFile)
		if err != nil {
			return []string{}, fmt.Errorf("error opening file: %w", err)
		}
	}
	if *flagURL != "" {
		resp, err := http.Get(*flagURL) // #nosec G107 -- URL is user-supplied input, intended
		if err != nil {
			return []string{}, fmt.Errorf("error fetching URL: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			return []string{}, fmt.Errorf("HTTP %d fetching %s", resp.StatusCode, *flagURL)
		}
		config.Input = resp.Body
		config.ContentType = resp.Header.Get("Content-Type")
	}
	config.Charset = *flagCharset
	config.MaxPrintLevel = *flagLimit
	config.Preformatted = *flagPre
	config.PrintColor = *flagColor
	config.EscapeHTML = !*flagPlain
	config.RawOutput = *flagRaw

	if *flagIndent != "" {
		if indentLevel, err := strconv.Atoi(*flagIndent); err == nil {
			config.IndentString = strings.Repeat(" ", indentLevel)
		} else {
			config.IndentString = *flagIndent
		}
	}

	// Set displayer
	displayerCount := 0
	if *flagText {
		config.Displayer = TextDisplayer{}
		displayerCount++
	}
	if *flagJSON {
		config.Displayer = JSONDisplayer{}
		displayerCount++
	}
	if *flagAttr != "" {
		config.Displayer = AttrDisplayer{Attr: *flagAttr}
		displayerCount++
	}
	if *flagNumber {
		config.Displayer = CountDisplayer{}
		displayerCount++
	}
	if displayerCount > 1 {
		return []string{}, fmt.Errorf("only one display option can be specified")
	}

	// Get selectors from positional args
	selectors := strings.Join(flag.Args(), " ")
	return ParseCommands(selectors)
}

// ParseCommands splits a command string into individual commands, handling quotes and commas.
func ParseCommands(cmdString string) ([]string, error) {
	var cmds []string
	last, next, max := 0, 0, len(cmdString)
	for {
		// if we're at the end of the string, return
		if next == max {
			if next > last {
				cmds = append(cmds, cmdString[last:next])
			}
			return cmds, nil
		}
		// evaluate a rune
		c := cmdString[next]
		switch c {
		case ' ':
			if next > last {
				cmds = append(cmds, cmdString[last:next])
			}
			last = next + 1
		case ',':
			if next > last {
				cmds = append(cmds, cmdString[last:next])
			}
			cmds = append(cmds, ",")
			last = next + 1
		case '\'', '"':
			// for quotes, consume runes until the quote has ended
			quoteChar := c
			for {
				next++
				if next == max {
					return []string{}, fmt.Errorf("unmatched open quote (%c)", quoteChar)
				}
				if cmdString[next] == '\\' {
					next++
					if next == max {
						return []string{}, fmt.Errorf("unmatched open quote (%c)", quoteChar)
					}
				} else if cmdString[next] == quoteChar {
					break
				}
			}
		}
		next++
	}
}
