# monk

`monk` is a command-line tool for parsing HTML and applying CSS selectors. It reads HTML from a file, a URL, or standard input and prints the matching elements in a readable tree format.

## Installation

```bash
go install github.com/janbodnar/monk@latest
```

## Usage

```
monk [flags] [selectors]
```

Selectors are standard CSS selectors. Multiple selectors separated by spaces are chained (each one narrows down the previous result). Selectors separated by commas are evaluated independently and their results are merged.

## Flags

| Flag | Description |
|------|-------------|
| `-f <file>` | Read HTML from a file |
| `-u <url>` | Fetch HTML from a URL |
| `-c` | Print output with colour highlighting |
| `-i <n\|str>` | Indent width in spaces, or a custom indent string |
| `-n` | Print the number of matched elements |
| `-l <n>` | Restrict output depth to `n` levels |
| `-p` | Do not escape HTML entities in output |
| `-r` | Raw output (no newlines or indentation) |
| `--pre` | Preserve formatting inside `<pre>` elements |
| `--charset <name>` | Force a specific character encoding |
| `--json` | Output matched nodes as JSON |
| `--text` | Output plain text content of matched nodes |
| `--attr <name>` | Output the value of an attribute from matched nodes |
| `-v` / `--version` | Display the version number |

## Examples

| Command | Description |
|---------|-------------|
| `monk -f page.html h1` | Print all `<h1>` elements from a local file |
| `monk -u https://example.com p` | Fetch a page and print all `<p>` elements |
| `cat page.html \| monk div p` | Pipe HTML and select `<p>` elements inside `<div>` elements |
| `monk -f page.html ul > li` | Select direct `<li>` children of `<ul>` elements |
| `monk -f page.html h2 + p` | Select `<p>` elements immediately following an `<h2>` |
| `monk -f page.html h1, h2` | Select all `<h1>` and `<h2>` elements |
| `monk -f page.html 'p:contains("hello")'` | Select `<p>` elements whose text contains "hello" |
| `monk -f page.html 'p:matches("^Hello")'` | Select `<p>` elements whose text matches a regular expression |
| `monk -f page.html 'ul:parent-of(li)'` | Select `<ul>` elements that have a direct `<li>` child |
| `monk -f page.html div head(3)` | Select the first 3 `<div>` elements |
| `monk -f page.html div tail(2)` | Select the last 2 `<div>` elements |
| `monk -f page.html a attr{href}` | Print the `href` attribute of every `<a>` element |
| `monk -f page.html --attr href a` | Same as above using a flag |
| `monk -f page.html p --text` | Print the plain text content of `<p>` elements |
| `monk -f page.html div --json` | Output matched `<div>` elements as JSON |
| `monk -f page.html div -n` | Print the number of matched `<div>` elements |
| `monk -f page.html -l 2 body` | Print the `<body>` tree limited to 2 levels deep |
| `monk -f page.html -c div` | Print matching elements with colour highlighting |

## Selector Operators

| Operator | Example | Description |
|----------|---------|-------------|
| ` ` (space) | `div p` | Descendant: select `<p>` anywhere inside `<div>` |
| `>` | `ul > li` | Child: select direct `<li>` children of `<ul>` |
| `+` | `h2 + p` | Adjacent sibling: select `<p>` immediately after `<h2>` |
| `,` | `h1, h2` | Union: select all `<h1>` and all `<h2>` elements |

## Display Functions (inline)

These tokens can be appended to a selector chain in place of a flag:

| Token | Description |
|-------|-------------|
| `text{}` | Output plain text content |
| `json{}` | Output nodes as JSON |
| `attr{name}` | Output the value of an attribute |
| `-n` / `--number` | Output the count of matched nodes |
