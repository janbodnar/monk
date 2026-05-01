# monk

`monk` is a command-line tool for parsing HTML and applying CSS selectors.
It reads HTML from standard input, a local file, or a URL, and outputs the
matching nodes in a readable tree format (default), as plain text, as JSON,
or as attribute values.

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

HTML is read from **stdin** unless `-f` or `-u` is specified.

### Flags

| Flag | Description |
|---|---|
| `-c` | Print result with color |
| `-f <file>` | Read HTML from a local file |
| `-u <url>` | Fetch HTML from a URL |
| `-i <n\|char>` | Number of spaces (or character) to use for indentation |
| `-n` | Print the number of matched elements |
| `-l <n>` | Restrict output to `n` levels deep |
| `-p` | Do not escape HTML entities in output |
| `-r` | Raw output (no newlines added between nodes) |
| `--pre` | Preserve whitespace inside `<pre>` blocks |
| `--charset <name>` | Override charset detection |
| `--json` | Output matched nodes as JSON |
| `--text` | Output plain text content of matched nodes |
| `--attr <name>` | Output the value of the named attribute |
| `-v` / `--version` | Display the version number |

### Selectors

Selectors follow standard CSS syntax. Multiple selectors separated by a
space chain off each other (descendant). Use `>` for direct children and
`+` for adjacent siblings. Separate independent selector groups with `,`
to union their results.

**Inline display tokens** can appear anywhere in the selector chain and
override the output mode:

| Token | Effect |
|---|---|
| `text{}` | Output plain text (same as `--text`) |
| `json{}` | Output JSON (same as `--json`) |
| `attr{name}` | Output attribute value (same as `--attr name`) |
| `--number` / `-n` | Output count (same as `-n`) |

**Slice tokens** limit which elements from the current selection are kept:

| Token | Effect |
|---|---|
| `head(n)` | Keep the first `n` matched elements |
| `tail(n)` | Keep the last `n` matched elements |

**Custom pseudo-classes** (extensions on top of standard CSS):

| Pseudo-class | Effect |
|---|---|
| `:contains("text")` | Match elements whose direct text child contains `text` |
| `:matches("regex")` | Match elements whose direct text child matches the regular expression |
| `:parent-of(selector)` | Match elements that have a direct child matching `selector` |

## Examples

| Command | Description |
|---|---|
| `cat page.html \| monk h1` | Print all `<h1>` elements from stdin |
| `monk -f page.html p` | Print all `<p>` elements from a file |
| `monk -u https://example.com a` | Print all `<a>` elements fetched from a URL |
| `monk -f page.html --text p` | Print the text content of every `<p>` |
| `monk -f page.html --attr href a` | Print the `href` value of every `<a>` |
| `monk -f page.html --json div.card` | Output matching nodes as JSON |
| `monk -f page.html -n p` | Print the number of `<p>` elements |
| `monk -f page.html -c h2` | Print `<h2>` elements with syntax colouring |
| `monk -f page.html -l 2 body` | Print `<body>` but only 2 levels deep |
| `monk -f page.html ul > li` | Print direct `<li>` children of `<ul>` |
| `monk -f page.html h2 + p` | Print a `<p>` immediately following an `<h2>` |
| `monk -f page.html h1,h2,h3` | Print all `<h1>`, `<h2>`, and `<h3>` elements |
| `monk -f page.html li head(3)` | Print the first 3 `<li>` elements |
| `monk -f page.html li tail(3)` | Print the last 3 `<li>` elements |
| `monk -f page.html 'li:contains("Go")'` | Print `<li>` elements containing the text "Go" |
| `monk -f page.html 'p:matches("^Hello")'` | Print `<p>` elements whose text matches a regex |
| `monk -f page.html 'ul:parent-of(li)'` | Print `<ul>` elements that have a direct `<li>` child |
| `monk -f page.html a attr{href}` | Print `href` values using an inline display token |
| `monk -u https://example.com p text{}` | Print paragraph text fetched from a URL |

## License

MIT
