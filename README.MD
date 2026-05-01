# monk

`monk` is a command-line tool for parsing HTML and applying CSS selectors. It reads HTML from
standard input, a local file, or a URL, applies one or more CSS selectors, and prints the
matching nodes. Output can be formatted as an indented tree (default), plain text, JSON, or a
single attribute value.

## Installation

```
go install github.com/janbodnar/monk@latest
```

Or build from source:

```
git clone https://github.com/janbodnar/monk
cd monk
go build -o monk .
```

## Usage

```
monk [flags] [selectors]
```

HTML is read from standard input unless `-f` or `-u` is given.

```
curl -s https://example.com | monk h1
monk -f page.html "ul li"
monk -u https://example.com "a"
```

## Flags

| Flag | Description |
|------|-------------|
| `-c` | Print result with color |
| `-f <file>` | Read HTML from a file |
| `-u <url>` | Fetch HTML from a URL |
| `-i <n\|char>` | Number of spaces (or character) to use for indentation |
| `-n` | Print the number of matched elements |
| `-l <n>` | Restrict output to `n` levels deep |
| `-p` | Don't escape HTML entities in output |
| `-r` | Raw output (no newlines between tags) |
| `--pre` | Preserve whitespace inside `<pre>` elements |
| `--charset <cs>` | Specify the character set of the input |
| `--json` | Output matched nodes as JSON |
| `--text` | Output the text content of matched nodes |
| `--attr <name>` | Output the value of the named attribute |
| `-v`, `--version` | Display version |

## Selectors

`monk` supports standard CSS selectors via [goquery](https://github.com/PuerkitoBio/goquery),
plus several extensions:

| Selector | Description |
|----------|-------------|
| `tag` | Match elements by tag name |
| `#id` | Match element by id attribute |
| `.class` | Match elements by class name |
| `[attr]` | Match elements that have the attribute |
| `[attr=val]` | Match elements where attribute equals value |
| `a b` | Descendant: match `b` anywhere inside `a` |
| `a > b` | Child: match `b` that is a direct child of `a` |
| `a + b` | Adjacent sibling: match `b` immediately after `a` |
| `sel1, sel2` | Union: match elements from either selector |
| `:nth-child(n)` | Match element that is the nth child |
| `:first-child` | Match the first child element |
| `:last-child` | Match the last child element |
| `:contains("text")` | Match elements with a direct text child containing `text` |
| `:matches("regex")` | Match elements with a direct text child matching the regex |
| `:parent-of(sel)` | Match elements that have a direct child matching `sel` |
| `head(n)` | Keep only the first `n` matched elements |
| `tail(n)` | Keep only the last `n` matched elements |

## Usage Examples

| Command | Description |
|---------|-------------|
| `monk -f page.html h1` | Select all `<h1>` elements from a file |
| `monk -u https://example.com title` | Fetch a URL and select the `<title>` element |
| `curl -s https://example.com \| monk p` | Pipe HTML and select all `<p>` elements |
| `monk -f page.html "ul li"` | Select all `<li>` elements inside `<ul>` |
| `monk -f page.html "div > p"` | Select `<p>` elements that are direct children of `<div>` |
| `monk -f page.html "h2 + p"` | Select `<p>` immediately following an `<h2>` |
| `monk -f page.html --text p` | Print the text content of all `<p>` elements |
| `monk -f page.html --attr href a` | Print the `href` attribute of all `<a>` elements |
| `monk -f page.html --json ul` | Output matched `<ul>` nodes as JSON |
| `monk -f page.html -n li` | Print the count of matched `<li>` elements |
| `monk -f page.html -l 2 body` | Print the `<body>` tree up to 2 levels deep |
| `monk -f page.html "li:contains(\"blue\")"` | Select `<li>` elements containing the text `blue` |
| `monk -f page.html "li:matches(\"^g\")"` | Select `<li>` elements whose text starts with `g` |
| `monk -f page.html "ul li head(3)"` | Select the first 3 `<li>` elements |
| `monk -f page.html "ul li tail(2)"` | Select the last 2 `<li>` elements |
| `monk -f page.html "h1, h2"` | Select all `<h1>` and `<h2>` elements |
| `monk -f page.html "#one p"` | Select `<p>` inside the element with `id="one"` |
| `monk -f page.html ".level-1 p"` | Select `<p>` inside elements with `class="level-1"` |
| `monk -f page.html -c ul li` | Print matched nodes with syntax highlighting |
| `monk -f page.html -r p` | Raw output without extra newlines |

