package service

import (
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/suite"
)

// Untestable error branches:
// - insertCSSIntoHead: goquery.NewDocumentFromReader never fails on a strings.Reader,
//   and doc.Html() only fails on writer errors (it renders into a bytes.Buffer).
// - insertCSSIntoHead: the `head.Length() == 0` branch is unreachable because the
//   HTML5 parser always synthesizes <head>; TestInsertCSSIntoHead_NoHeadTag documents this.
// - convertCSSIntoInline: premailer.NewPremailerFromString and Transform only fail on
//   internal parser errors that valid goquery output cannot trigger.

const (
	expectedTitleStyle = "color:red;font-weight:bold"
	expectedNoteStyle  = "color:blue !important"
)

type CSSToolsServiceSuite struct {
	suite.Suite
	svc *CSSToolsService
}

func TestCSSToolsServiceSuite(t *testing.T) {
	suite.Run(t, new(CSSToolsServiceSuite))
}

func (s *CSSToolsServiceSuite) SetupSuite() {
	log.SetOutput(io.Discard)
}

func (s *CSSToolsServiceSuite) TearDownSuite() {
	log.SetOutput(os.Stderr)
}

func (s *CSSToolsServiceSuite) SetupTest() {
	s.svc = NewCSSToolsService("testdata", NewTemplatesService("testdata"))
}

func (s *CSSToolsServiceSuite) parse(html string) *goquery.Document {
	s.T().Helper()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	s.Require().NoError(err)
	return doc
}

// assertInlined verifies a <p> got the expected inline style and lost its class attribute
func (s *CSSToolsServiceSuite) assertInlined(sel *goquery.Selection, wantText, wantStyle string) {
	s.T().Helper()

	s.Require().Equal(1, sel.Length())
	s.Equal(wantText, sel.Text())

	style, ok := sel.Attr("style")
	s.True(ok, "style attribute missing")
	s.Equal(wantStyle, style)

	_, hasClass := sel.Attr("class")
	s.False(hasClass, "class attribute should be removed")
}

// --- RenderTemplateWithCSS ---

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_InlinesStyles() {
	got, err := s.svc.RenderTemplateWithCSS("styled.html", "styles.css", map[string]string{"Name": "Rita"})
	s.Require().NoError(err)

	doc := s.parse(got)
	paragraphs := doc.Find("body p")
	s.Require().Equal(2, paragraphs.Length())
	s.assertInlined(paragraphs.Eq(0), "Rita", expectedTitleStyle)
	s.assertInlined(paragraphs.Eq(1), "note", expectedNoteStyle)

	// Existing head content is kept and the <style> block is consumed by premailer
	s.Equal("T", doc.Find("head title").Text())
	s.Equal(0, doc.Find("style").Length())
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_TemplateWithoutHead() {
	got, err := s.svc.RenderTemplateWithCSS("no_head.html", "styles.css", map[string]string{"Name": "Rita"})
	s.Require().NoError(err)

	doc := s.parse(got)
	s.Equal(1, doc.Find("head").Length())
	s.assertInlined(doc.Find("body p"), "Rita", expectedTitleStyle)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_CSSFileNotFound() {
	got, err := s.svc.RenderTemplateWithCSS("styled.html", "missing.css", map[string]string{"Name": "Rita"})

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to read CSS file missing.css: "), err.Error())
	s.Empty(got)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_TemplateNotFound() {
	got, err := s.svc.RenderTemplateWithCSS("missing.html", "styles.css", nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to render template: "), err.Error())
	s.Empty(got)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_TemplateErrorReportedBeforeCSSError() {
	got, err := s.svc.RenderTemplateWithCSS("missing.html", "missing.css", nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to render template: "), err.Error())
	s.Empty(got)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithCSS_UsesOwnPathForCSS() {
	// CSS is resolved against the CSS path, templates against the template path
	svc := NewCSSToolsService(s.T().TempDir(), NewTemplatesService("testdata"))

	got, err := svc.RenderTemplateWithCSS("styled.html", "styles.css", map[string]string{"Name": "Rita"})

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to read CSS file styles.css: "), err.Error())
	s.Empty(got)
}

// --- RenderTemplateWithFuncsAndCSS ---

func (s *CSSToolsServiceSuite) TestRenderTemplateWithFuncsAndCSS_DefaultFuncs() {
	got, err := s.svc.RenderTemplateWithFuncsAndCSS("styled_funcs.html", "styles.css", map[string]string{"Name": "Rita"}, nil)
	s.Require().NoError(err)

	s.assertInlined(s.parse(got).Find("body p"), "RITA", expectedTitleStyle)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithFuncsAndCSS_CustomFuncs() {
	custom := []template.FuncMap{{"toUpper": func(v string) string { return "custom:" + v }}}

	got, err := s.svc.RenderTemplateWithFuncsAndCSS("styled_funcs.html", "styles.css", map[string]string{"Name": "Rita"}, custom)
	s.Require().NoError(err)

	s.assertInlined(s.parse(got).Find("body p"), "custom:Rita", expectedTitleStyle)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithFuncsAndCSS_CSSFileNotFound() {
	got, err := s.svc.RenderTemplateWithFuncsAndCSS("styled_funcs.html", "missing.css", map[string]string{"Name": "Rita"}, nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to read CSS file missing.css: "), err.Error())
	s.Empty(got)
}

func (s *CSSToolsServiceSuite) TestRenderTemplateWithFuncsAndCSS_TemplateNotFound() {
	got, err := s.svc.RenderTemplateWithFuncsAndCSS("missing.html", "styles.css", nil, nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to render template: "), err.Error())
	s.Empty(got)
}

// --- insertCSSIntoHead ---

func (s *CSSToolsServiceSuite) TestInsertCSSIntoHead_ExistingHead() {
	got, err := s.svc.insertCSSIntoHead("<html><head><title>T</title></head><body><p>x</p></body></html>", ".a{color:red}")

	s.Require().NoError(err)
	s.Equal("<html><head><title>T</title><style>.a{color:red}</style></head><body><p>x</p></body></html>", got)
}

func (s *CSSToolsServiceSuite) TestInsertCSSIntoHead_NoHeadTag() {
	// The HTML5 parser adds <html><head><body> around a fragment, so the style still lands in <head>
	got, err := s.svc.insertCSSIntoHead("<p>x</p>", ".a{}")

	s.Require().NoError(err)
	s.Equal("<html><head><style>.a{}</style></head><body><p>x</p></body></html>", got)
}

// --- convertCSSIntoInline ---

func (s *CSSToolsServiceSuite) TestConvertCSSIntoInline_KeepsImportantAndRemovesClasses() {
	got, err := s.svc.convertCSSIntoInline(`<html><head><style>.a{color:red} .b{color:blue !important}</style></head><body><p class="a">1</p><p class="b">2</p></body></html>`)
	s.Require().NoError(err)

	paragraphs := s.parse(got).Find("body p")
	s.Require().Equal(2, paragraphs.Length())
	s.assertInlined(paragraphs.Eq(0), "1", "color:red")
	s.assertInlined(paragraphs.Eq(1), "2", "color:blue !important")
}
