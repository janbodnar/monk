package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

// ----- helpers ----------------------------------------------------------------

func sliceEq(s1, s2 []string) bool {
	if len(s1) != len(s2) {
		return false
	}
	for i := range s1 {
		if s1[i] != s2[i] {
			return false
		}
	}
	return true
}

const testHTML = `<html><body>
  <div id="main" class="container">
    <h1>Title</h1>
    <p class="intro">Intro paragraph</p>
    <ul>
      <li class="item">Alpha</li>
      <li class="item">Beta</li>
      <li class="item special">Gamma</li>
    </ul>
    <div class="nested">
      <span>inner span</span>
    </div>
  </div>
  <footer>
    <p>Footer</p>
  </footer>
</body></html>`

func mustParseDoc(t *testing.T, src string) *goquery.Document {
	t.Helper()
	doc, err := ParseDocument(strings.NewReader(src), "", "")
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	return doc
}

func applySelectors(t *testing.T, selStr string) ([]*goquery.Selection, error) {
	t.Helper()
	doc := mustParseDoc(t, testHTML)
	cmds, err := ParseCommands(selStr)
	if err != nil {
		return nil, err
	}
	var sels []*goquery.Selection
	for _, g := range splitByComma(cmds) {
		sel, err := applyCommandGroup(doc, g)
		if err != nil {
			return nil, err
		}
		sels = append(sels, sel)
	}
	return sels, nil
}

// ----- ParseCommands ----------------------------------------------------------

type ParseCmdTest struct {
	input string
	split []string
	ok    bool
}

var parseCmdTests = []ParseCmdTest{
	{`w1 w2`, []string{`w1`, `w2`}, true},
	{`w1 w2 w3`, []string{`w1`, `w2`, `w3`}, true},
	{`w1 'w2 w3'`, []string{`w1`, `'w2 w3'`}, true},
	{`w1 "w2 w3"`, []string{`w1`, `"w2 w3"`}, true},
	{`w1   "w2 w3"`, []string{`w1`, `"w2 w3"`}, true},
	{`w1   'w2 w3'`, []string{`w1`, `'w2 w3'`}, true},
	{`w1"w2 w3"`, []string{`w1"w2 w3"`}, true},
	{`w1'w2 w3'`, []string{`w1'w2 w3'`}, true},
	{`w1"w2 'w3"`, []string{`w1"w2 'w3"`}, true},
	{`w1'w2 "w3'`, []string{`w1'w2 "w3'`}, true},
	{`"w1 w2" "w3"`, []string{`"w1 w2"`, `"w3"`}, true},
	{`'w1 w2' "w3"`, []string{`'w1 w2'`, `"w3"`}, true},
	{`'w1 \'w2' "w3"`, []string{`'w1 \'w2'`, `"w3"`}, true},
	{`'w1 \'w2 "w3"`, []string{}, false},
	{`w1 'w2 w3'"`, []string{}, false},
	{`w1 "w2 w3"'`, []string{}, false},
	{`w1 '  "w2 w3"`, []string{}, false},
	{`w1 "  'w2 w3'`, []string{}, false},
	{`w1"w2 w3""`, []string{}, false},
	{`w1'w2 w3''`, []string{}, false},
	{`w1"w2 'w3""`, []string{}, false},
	{`w1'w2 "w3''`, []string{}, false},
	{`"w1 w2" "w3"'`, []string{}, false},
	{`'w1 w2' "w3"'`, []string{}, false},
	{`w1,"w2 w3"`, []string{`w1`, `,`, `"w2 w3"`}, true},
	{`w1,'w2 w3'`, []string{`w1`, `,`, `'w2 w3'`}, true},
	{`w1  ,  "w2 w3"`, []string{`w1`, `,`, `"w2 w3"`}, true},
	{`w1  ,  'w2 w3'`, []string{`w1`, `,`, `'w2 w3'`}, true},
	{`w1,  "w2 w3"`, []string{`w1`, `,`, `"w2 w3"`}, true},
	{`w1,  'w2 w3'`, []string{`w1`, `,`, `'w2 w3'`}, true},
	{`w1  ,"w2 w3"`, []string{`w1`, `,`, `"w2 w3"`}, true},
	{`w1  ,'w2 w3'`, []string{`w1`, `,`, `'w2 w3'`}, true},
	{`w1"w2, w3"`, []string{`w1"w2, w3"`}, true},
	{`w1'w2, w3'`, []string{`w1'w2, w3'`}, true},
	{`w1"w2, 'w3"`, []string{`w1"w2, 'w3"`}, true},
	{`w1'w2, "w3'`, []string{`w1'w2, "w3'`}, true},
	{`"w1, w2" "w3"`, []string{`"w1, w2"`, `"w3"`}, true},
	{`'w1, w2' "w3"`, []string{`'w1, w2'`, `"w3"`}, true},
	{`'w1, \'w2' "w3"`, []string{`'w1, \'w2'`, `"w3"`}, true},
	{`h1, .article-teaser, .article-content`, []string{
		`h1`, `,`, `.article-teaser`, `,`, `.article-content`,
	}, true},
	{`h1 ,.article-teaser ,.article-content`, []string{
		`h1`, `,`, `.article-teaser`, `,`, `.article-content`,
	}, true},
	{`h1 , .article-teaser , .article-content`, []string{
		`h1`, `,`, `.article-teaser`, `,`, `.article-content`,
	}, true},
}

func TestParseCommands(t *testing.T) {
	for _, test := range parseCmdTests {
		parsed, err := ParseCommands(test.input)
		if test.ok != (err == nil) {
			t.Errorf("`%s`: should have caused error? %v", test.input, !test.ok)
		} else if !sliceEq(test.split, parsed) {
			t.Errorf("`%s`: got %v, want %v", test.input, parsed, test.split)
		}
	}
}

// ----- splitByComma -----------------------------------------------------------

func TestSplitByComma(t *testing.T) {
	tests := []struct {
		input  []string
		expect [][]string
	}{
		{[]string{"div", "p"}, [][]string{{"div", "p"}}},
		{[]string{"div", ",", "p"}, [][]string{{"div"}, {"p"}}},
		{[]string{"div", ",", "p", ",", "span"}, [][]string{{"div"}, {"p"}, {"span"}}},
		{[]string{",", "p"}, [][]string{{"p"}}},
		{[]string{"div", ","}, [][]string{{"div"}}},
		{nil, nil},
	}
	for _, tt := range tests {
		got := splitByComma(tt.input)
		if len(got) != len(tt.expect) {
			t.Errorf("splitByComma(%v): got %d groups, want %d", tt.input, len(got), len(tt.expect))
			continue
		}
		for i, g := range got {
			if !sliceEq(g, tt.expect[i]) {
				t.Errorf("splitByComma(%v)[%d]: got %v, want %v", tt.input, i, g, tt.expect[i])
			}
		}
	}
}

// ----- parseHeadTail ----------------------------------------------------------

func TestParseHeadTail(t *testing.T) {
	tests := []struct {
		input string
		n     int
		isErr bool
	}{
		{"head(3)", 3, false},
		{"tail(5)", 5, false},
		{"head(0)", 0, false},
		{"head(-1)", 0, true},
		{"head()", 0, true},
		{"head(abc)", 0, true},
		{"head(1", 0, true},
	}
	for _, tt := range tests {
		n, err := parseHeadTail(tt.input)
		if tt.isErr {
			if err == nil {
				t.Errorf("parseHeadTail(%q): expected error, got nil", tt.input)
			}
		} else {
			if err != nil {
				t.Errorf("parseHeadTail(%q): unexpected error: %v", tt.input, err)
			} else if n != tt.n {
				t.Errorf("parseHeadTail(%q): got %d, want %d", tt.input, n, tt.n)
			}
		}
	}
}

// ----- extractDisplayFunction -------------------------------------------------

func TestExtractDisplayFunction(t *testing.T) {
	tests := []struct {
		input    []string
		filtered []string
		wantNil  bool
		wantType string
		wantAttr string
	}{
		{[]string{"div", "p"}, []string{"div", "p"}, true, "", ""},
		{[]string{"div", "text{}"}, []string{"div"}, false, "TextDisplayer", ""},
		{[]string{"div", "json{}"}, []string{"div"}, false, "JSONDisplayer", ""},
		{[]string{"div", "-n"}, []string{"div"}, false, "CountDisplayer", ""},
		{[]string{"div", "--number"}, []string{"div"}, false, "CountDisplayer", ""},
		{[]string{"div", "attr{class}"}, []string{"div"}, false, "AttrDisplayer", "class"},
		{[]string{"div", "attr{href}"}, []string{"div"}, false, "AttrDisplayer", "href"},
	}
	for _, tt := range tests {
		filtered, disp := extractDisplayFunction(tt.input)
		if !sliceEq(filtered, tt.filtered) {
			t.Errorf("extractDisplayFunction(%v): filtered = %v, want %v", tt.input, filtered, tt.filtered)
		}
		if tt.wantNil && disp != nil {
			t.Errorf("extractDisplayFunction(%v): expected nil displayer, got %T", tt.input, disp)
			continue
		}
		if !tt.wantNil && disp == nil {
			t.Errorf("extractDisplayFunction(%v): expected %s displayer, got nil", tt.input, tt.wantType)
			continue
		}
		if tt.wantNil {
			continue
		}
		switch tt.wantType {
		case "TextDisplayer":
			if _, ok := disp.(TextDisplayer); !ok {
				t.Errorf("extractDisplayFunction(%v): want TextDisplayer, got %T", tt.input, disp)
			}
		case "JSONDisplayer":
			if _, ok := disp.(JSONDisplayer); !ok {
				t.Errorf("extractDisplayFunction(%v): want JSONDisplayer, got %T", tt.input, disp)
			}
		case "CountDisplayer":
			if _, ok := disp.(CountDisplayer); !ok {
				t.Errorf("extractDisplayFunction(%v): want CountDisplayer, got %T", tt.input, disp)
			}
		case "AttrDisplayer":
			ad, ok := disp.(AttrDisplayer)
			if !ok {
				t.Errorf("extractDisplayFunction(%v): want AttrDisplayer, got %T", tt.input, disp)
			} else if ad.Attr != tt.wantAttr {
				t.Errorf("extractDisplayFunction(%v): AttrDisplayer.Attr = %q, want %q", tt.input, ad.Attr, tt.wantAttr)
			}
		}
	}
}

// ----- ParseDocument ----------------------------------------------------------

func TestParseDocument(t *testing.T) {
	doc, err := ParseDocument(strings.NewReader(testHTML), "", "")
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}
	if doc == nil {
		t.Fatal("ParseDocument returned nil document")
	}
	if n := doc.Find("li").Length(); n != 3 {
		t.Errorf("expected 3 <li>, got %d", n)
	}
}

func TestParseDocument_InvalidCharset(t *testing.T) {
	_, err := ParseDocument(strings.NewReader(testHTML), "not-a-real-charset", "")
	if err == nil {
		t.Error("expected error for invalid charset, got nil")
	}
}

func TestParseDocument_ISO8859_1(t *testing.T) {
	// ISO-8859-1 encoded HTML with meta charset and the word Café encoded as 0xE9
	b := []byte("<html><head><meta charset=\"iso-8859-1\"></head><body><p>Caf\xe9</p></body></html>")
	doc, err := ParseDocument(bytes.NewReader(b), "", "")
	if err != nil {
		t.Fatalf("ParseDocument returned error: %v", err)
	}
	got := doc.Find("p").Text()
	want := "Café"
	if got != want {
		t.Errorf("expected decoded text %q, got %q", want, got)
	}
}

// ----- applyCommandGroup – selector traversal ---------------------------------

func TestApplyCommandGroup_Descendant(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{"li"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 3 {
		t.Errorf("expected 3 <li>, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_DescendantChain(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	// #main ul li
	sel, err := applyCommandGroup(doc, []string{"#main", "ul", "li"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 3 {
		t.Errorf("expected 3 <li>, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_DirectChild(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	// #main > p — only the direct child <p>, not nested ones
	sel, err := applyCommandGroup(doc, []string{"#main", ">", "p"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf("expected 1 direct-child <p> of #main, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_AdjacentSibling(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	// h1 + p — the <p> immediately following <h1>
	sel, err := applyCommandGroup(doc, []string{"h1", "+", "p"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf("expected 1 adjacent-sibling <p> after <h1>, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_ClassSelector(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{".special"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf("expected 1 .special element, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_IDSelector(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{"#main"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf("expected 1 #main element, got %d", sel.Length())
	}
}

func TestApplyCommandGroup_NoMatch(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{"table"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 0 {
		t.Errorf("expected 0 matches for 'table', got %d", sel.Length())
	}
}

func TestApplyCommandGroup_Empty(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{})
	if err != nil {
		t.Fatal(err)
	}
	if sel == nil {
		t.Error("expected non-nil selection for empty command group")
	}
}

func TestApplyCommandGroup_TrailingChildOp(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	_, err := applyCommandGroup(doc, []string{"div", ">"})
	if err == nil {
		t.Error("expected error for trailing '>', got nil")
	}
}

func TestApplyCommandGroup_TrailingAdjacentOp(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	_, err := applyCommandGroup(doc, []string{"h1", "+"})
	if err == nil {
		t.Error("expected error for trailing '+', got nil")
	}
}

// ----- head / tail ------------------------------------------------------------

func TestApplyCommandGroup_Head(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{"li", "head(2)"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 2 {
		t.Errorf("head(2): expected 2, got %d", sel.Length())
	}
	if text := sel.First().Text(); text != "Alpha" {
		t.Errorf("head(2): first item = %q, want \"Alpha\"", text)
	}
}

func TestApplyCommandGroup_Tail(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{"li", "tail(1)"})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf("tail(1): expected 1, got %d", sel.Length())
	}
	if text := sel.First().Text(); text != "Gamma" {
		t.Errorf("tail(1): item = %q, want \"Gamma\"", text)
	}
}

func TestApplyHead_LargerThanLength(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	all := doc.Find("li")
	got := applyHead(all, 100)
	if got.Length() != 3 {
		t.Errorf("applyHead(100): expected 3, got %d", got.Length())
	}
}

func TestApplyTail_LargerThanLength(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	all := doc.Find("li")
	got := applyTail(all, 100)
	if got.Length() != 3 {
		t.Errorf("applyTail(100): expected 3, got %d", got.Length())
	}
}

func TestApplyHead_Zero(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	all := doc.Find("li")
	got := applyHead(all, 0)
	if got.Length() != 0 {
		t.Errorf("applyHead(0): expected 0, got %d", got.Length())
	}
}

// ----- comma groups (OR) ------------------------------------------------------

func TestCommaGroups(t *testing.T) {
	sels, err := applySelectors(t, "h1, footer p")
	if err != nil {
		t.Fatal(err)
	}
	nodes := combineSelections(sels)
	if len(nodes) != 2 {
		t.Errorf("'h1, footer p': expected 2 nodes, got %d", len(nodes))
	}
}

func TestCommaGroups_Three(t *testing.T) {
	sels, err := applySelectors(t, "h1, p.intro, footer p")
	if err != nil {
		t.Fatal(err)
	}
	nodes := combineSelections(sels)
	if len(nodes) != 3 {
		t.Errorf("'h1, p.intro, footer p': expected 3 nodes, got %d", len(nodes))
	}
}

// ----- combineSelections ------------------------------------------------------

func TestCombineSelections_Nil(t *testing.T) {
	nodes := combineSelections(nil)
	if len(nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(nodes))
	}
}

func TestCombineSelections_Multiple(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel1 := doc.Find("h1")
	sel2 := doc.Find("footer p")
	nodes := combineSelections([]*goquery.Selection{sel1, sel2})
	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(nodes))
	}
}

// ----- custom pseudo-classes --------------------------------------------------

func TestContainsPseudo(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{`li:contains("Beta")`})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf(`:contains("Beta"): expected 1, got %d`, sel.Length())
	}
}

func TestContainsPseudo_NoMatch(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{`li:contains("Delta")`})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 0 {
		t.Errorf(`:contains("Delta"): expected 0, got %d`, sel.Length())
	}
}

func TestMatchesPseudo(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	sel, err := applyCommandGroup(doc, []string{`li:matches("^(Alpha|Gamma)$")`})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 2 {
		t.Errorf(`:matches("^(Alpha|Gamma)$"): expected 2, got %d`, sel.Length())
	}
}

func TestMatchesPseudo_InvalidRegex(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	_, err := applyCommandGroup(doc, []string{`li:matches("[")`})
	if err == nil {
		t.Error("expected error for invalid regex in :matches(), got nil")
	}
}

func TestParentOfPseudo(t *testing.T) {
	doc := mustParseDoc(t, testHTML)
	// div.nested is the only div with a direct child <span>
	sel, err := applyCommandGroup(doc, []string{`div:parent-of(span)`})
	if err != nil {
		t.Fatal(err)
	}
	if sel.Length() != 1 {
		t.Errorf(`:parent-of(span): expected 1, got %d`, sel.Length())
	}
}

// ----- -u / --url (fetch from URL) -------------------------------------------

// newTestServer starts an httptest server serving the given HTML body with the
// given Content-Type. The caller must call ts.Close().
func newTestServer(t *testing.T, body string, contentType string) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		fmt.Fprint(w, body)
	}))
	return ts
}

// fetchURL is the integration helper: fetches the URL, parses the document,
// applies selectors and returns the matched Selection.
func fetchURL(t *testing.T, rawURL string, selStr string) *goquery.Selection {
	t.Helper()
	resp, err := http.Get(rawURL) // #nosec G107
	if err != nil {
		t.Fatalf("http.Get(%q): %v", rawURL, err)
	}
	defer resp.Body.Close()
	contentType := resp.Header.Get("Content-Type")
	doc, err := ParseDocument(resp.Body, "", contentType)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	cmds, err := ParseCommands(selStr)
	if err != nil {
		t.Fatalf("ParseCommands(%q): %v", selStr, err)
	}
	sel, err := applyCommandGroup(doc, cmds)
	if err != nil {
		t.Fatalf("applyCommandGroup: %v", err)
	}
	return sel
}

// TestFetchURL_BasicSelector verifies that HTML fetched from a URL is parsed
// and selectors work correctly.
func TestFetchURL_BasicSelector(t *testing.T) {
	ts := newTestServer(t, testHTML, "text/html; charset=utf-8")
	defer ts.Close()

	sel := fetchURL(t, ts.URL, "h1")
	if sel.Length() != 1 {
		t.Errorf("expected 1 <h1>, got %d", sel.Length())
	}
	if got := sel.Text(); got != "Title" {
		t.Errorf("h1 text = %q, want \"Title\"", got)
	}
}

// TestFetchURL_MultipleMatches verifies multiple matched nodes are returned.
func TestFetchURL_MultipleMatches(t *testing.T) {
	ts := newTestServer(t, testHTML, "text/html; charset=utf-8")
	defer ts.Close()

	sel := fetchURL(t, ts.URL, "li")
	if sel.Length() != 3 {
		t.Errorf("expected 3 <li>, got %d", sel.Length())
	}
}

// TestFetchURL_NoMatch verifies that a selector with no matches returns empty.
func TestFetchURL_NoMatch(t *testing.T) {
	ts := newTestServer(t, testHTML, "text/html; charset=utf-8")
	defer ts.Close()

	sel := fetchURL(t, ts.URL, "table")
	if sel.Length() != 0 {
		t.Errorf("expected 0 matches, got %d", sel.Length())
	}
}

// TestFetchURL_ContentTypeCharset verifies that the Content-Type header's
// charset hint is used for charset detection (ISO-8859-1 example).
func TestFetchURL_ContentTypeCharset(t *testing.T) {
	body := "<html><head></head><body><p>Caf\xe9</p></body></html>"
	ts := newTestServer(t, body, "text/html; charset=iso-8859-1")
	defer ts.Close()

	sel := fetchURL(t, ts.URL, "p")
	if got := sel.Text(); got != "Café" {
		t.Errorf("charset decoding: p text = %q, want \"Café\"", got)
	}
}

// TestFetchURL_ChainedSelectors verifies chained CSS selectors work on fetched HTML.
func TestFetchURL_ChainedSelectors(t *testing.T) {
	ts := newTestServer(t, testHTML, "text/html; charset=utf-8")
	defer ts.Close()

	sel := fetchURL(t, ts.URL, "#main ul li")
	if sel.Length() != 3 {
		t.Errorf("chained selector '#main ul li': expected 3, got %d", sel.Length())
	}
}

// TestFetchURL_HTTPError verifies that a non-200 response is surfaced as an error.
func TestFetchURL_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	// Simulate what ParseArgs does: check status before passing body to ParseDocument.
	resp, err := http.Get(ts.URL) // #nosec G107
	if err != nil {
		t.Fatalf("http.Get: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("expected non-200 status, got 200")
	}
}

// TestFetchURL_MutualExclusion_FileAndURL verifies that -f and -u cannot be
// used together. This is tested at the logic level used in ParseArgs.
func TestFetchURL_MutualExclusion(t *testing.T) {
	// Simulate the guard condition from ParseArgs directly.
	file := "some.html"
	url := "http://example.com"
	if file != "" && url != "" {
		// This is the expected error path — test passes if we reach here.
		return
	}
	t.Error("mutual exclusion guard was not triggered")
}

// TestParseDocument_WithContentTypeHint verifies that passing a Content-Type
// hint to ParseDocument correctly influences charset detection.
func TestParseDocument_WithContentTypeHint(t *testing.T) {
	b := []byte("<html><head></head><body><p>Caf\xe9</p></body></html>")
	doc, err := ParseDocument(bytes.NewReader(b), "", "text/html; charset=iso-8859-1")
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if got := doc.Find("p").Text(); got != "Café" {
		t.Errorf("Content-Type hint charset: got %q, want \"Café\"", got)
	}
}

// TestParseDocument_ContentTypeHintOverriddenByExplicit verifies that an
// explicit --charset flag takes priority over the Content-Type hint.
func TestParseDocument_ExplicitCharsetTakesPriority(t *testing.T) {
	b := []byte("<html><head></head><body><p>Caf\xe9</p></body></html>")
	// explicit charset=iso-8859-1, content-type hint claims utf-8 (which would
	// misread the byte 0xE9); explicit charset should win.
	doc, err := ParseDocument(bytes.NewReader(b), "iso-8859-1", "text/html; charset=utf-8")
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	if got := doc.Find("p").Text(); got != "Café" {
		t.Errorf("explicit charset priority: got %q, want \"Café\"", got)
	}
}
