package main

import (
	"io"
	"os"
)

// Config holds the configuration for the monk tool.
type Config struct {
	Input         io.ReadCloser
	Charset       string
	ContentType   string
	MaxPrintLevel int
	Preformatted  bool
	PrintColor    bool
	EscapeHTML    bool
	IndentString  string
	Displayer     Displayer
	RawOutput     bool
}

var config = Config{
	Input:         os.Stdin,
	MaxPrintLevel: -1,
	EscapeHTML:    true,
	IndentString:  " ",
	Displayer:     TreeDisplayer{},
}
