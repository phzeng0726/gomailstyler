package service

import (
	"errors"
	"html/template"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"testing"
	texttemplate "text/template"
	"time"

	"github.com/stretchr/testify/suite"
)

type funcsData struct {
	Name   string
	Padded string
	Phrase string
	Date   time.Time
	Blank  string
	HTML   string
}

func newFuncsData() funcsData {
	return funcsData{
		Name:   "Rita",
		Padded: "  hi  ",
		Phrase: "hello world",
		Date:   time.Date(2026, 11, 16, 9, 30, 0, 0, time.UTC),
		Blank:  "   ",
		HTML:   "<b>x</b>",
	}
}

// Expected output of testdata/funcs.html rendered with newFuncsData()
const expectedFuncsOutput = "3|RITA|rita|hi|Hello World|2026-11-16|true|false|<b>x</b>|&lt;b&gt;x&lt;/b&gt;|42"

type TemplatesServiceSuite struct {
	suite.Suite
	svc *TemplatesService
}

func TestTemplatesServiceSuite(t *testing.T) {
	suite.Run(t, new(TemplatesServiceSuite))
}

func (s *TemplatesServiceSuite) SetupSuite() {
	log.SetOutput(io.Discard)
}

func (s *TemplatesServiceSuite) TearDownSuite() {
	log.SetOutput(os.Stderr)
}

func (s *TemplatesServiceSuite) SetupTest() {
	s.svc = NewTemplatesService("testdata")
}

func (s *TemplatesServiceSuite) customFuncs() []template.FuncMap {
	return []template.FuncMap{
		{
			"toUpper": func(v string) string { return "custom:" + v },
			"greet":   func(v string) string { return "hi " + v },
		},
		{
			"extra": func() string { return "never registered" },
		},
	}
}

// --- RenderTemplate ---

func (s *TemplatesServiceSuite) TestRenderTemplate_Success() {
	got, err := s.svc.RenderTemplate("basic.html", map[string]string{"Name": "Rita"})

	s.Require().NoError(err)
	s.Equal("<p>Hello, Rita!</p>", got)
}

func (s *TemplatesServiceSuite) TestRenderTemplate_EscapesData() {
	got, err := s.svc.RenderTemplate("basic.html", map[string]string{"Name": "<script>"})

	s.Require().NoError(err)
	s.Equal("<p>Hello, &lt;script&gt;!</p>", got)
}

func (s *TemplatesServiceSuite) TestRenderTemplate_FileNotFound() {
	got, err := s.svc.RenderTemplate("missing.html", nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderTemplate_SyntaxError() {
	got, err := s.svc.RenderTemplate("syntax_error.html", nil)

	s.EqualError(err, "template: syntax_error.html:1: unexpected EOF")
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderTemplate_ExecError() {
	got, err := s.svc.RenderTemplate("exec_error.html", map[string]string{"Name": "Rita"})

	s.assertExecError(err, "exec_error.html")
	s.Empty(got)
}

// --- RenderHTMLContent ---

func (s *TemplatesServiceSuite) TestRenderHTMLContent_Success() {
	got, err := s.svc.RenderHTMLContent("<h1>{{.Title}}</h1>", map[string]string{"Title": "Notice"})

	s.Require().NoError(err)
	s.Equal("<h1>Notice</h1>", got)
}

func (s *TemplatesServiceSuite) TestRenderHTMLContent_DefaultFuncsNotAvailable() {
	got, err := s.svc.RenderHTMLContent("{{toUpper .}}", "x")

	s.EqualError(err, `template: content:1: function "toUpper" not defined`)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderHTMLContent_SyntaxError() {
	got, err := s.svc.RenderHTMLContent("{{if .}}unclosed", nil)

	s.EqualError(err, "template: content:1: unexpected EOF")
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderHTMLContent_ExecError() {
	got, err := s.svc.RenderHTMLContent("{{.Name.Missing}}", map[string]string{"Name": "Rita"})

	s.assertExecError(err, "content")
	s.Empty(got)
}

// --- Default template functions ---

func (s *TemplatesServiceSuite) TestDefaultFuncs_EachFunction() {
	data := newFuncsData()

	cases := []struct {
		name    string
		content string
		want    string
	}{
		{"add", "{{add 1 2}}", "3"},
		{"add negative", "{{add -5 2}}", "-3"},
		{"toUpper", "{{toUpper .Name}}", "RITA"},
		{"toLower", "{{toLower .Name}}", "rita"},
		{"trim", "[{{trim .Padded}}]", "[hi]"},
		{"title", "{{title .Phrase}}", "Hello World"},
		{"formatDate", `{{formatDate .Date "2006-01-02 15:04"}}`, "2026-11-16 09:30"},
		{"isEmpty whitespace only", "{{isEmpty .Blank}}", "true"},
		{"isEmpty empty string", `{{isEmpty ""}}`, "true"},
		{"isEmpty non-empty", "{{isEmpty .Name}}", "false"},
		{"safeHTML not escaped", "{{safeHTML .HTML}}", "<b>x</b>"},
		{"without safeHTML escaped", "{{.HTML}}", "&lt;b&gt;x&lt;/b&gt;"},
		{"inc", "{{inc 0}}", "1"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got, err := s.svc.RenderHTMLContentWithFuncs(tc.content, data, nil)

			s.Require().NoError(err)
			s.Equal(tc.want, got)
		})
	}
}

func (s *TemplatesServiceSuite) TestDefaultFuncs_AllInTemplateFile() {
	got, err := s.svc.RenderTemplateWithFuncs("funcs.html", newFuncsData(), nil)

	s.Require().NoError(err)
	s.Equal(expectedFuncsOutput, got)
}

// --- RenderTemplateWithFuncs ---

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_CustomOverridesAndAdds() {
	got, err := s.svc.RenderTemplateWithFuncs("custom_funcs.html", map[string]string{"Name": "Rita"}, s.customFuncs())

	s.Require().NoError(err)
	s.Equal("custom:Rita|hi Rita", got)
}

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_DefaultsKeptWhenNotOverridden() {
	custom := []template.FuncMap{{"greet": func(v string) string { return "hi " + v }}}

	got, err := s.svc.RenderTemplateWithFuncs("custom_funcs.html", map[string]string{"Name": "Rita"}, custom)

	s.Require().NoError(err)
	s.Equal("RITA|hi Rita", got)
}

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_CustomFuncMissing() {
	// custom_funcs.html uses "greet", which only exists when a custom FuncMap provides it
	got, err := s.svc.RenderTemplateWithFuncs("custom_funcs.html", map[string]string{"Name": "Rita"}, nil)

	s.EqualError(err, `template: custom_funcs.html:1: function "greet" not defined`)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_OnlyFirstFuncMapUsed() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(filepath.Join(dir, "extra.html"), []byte("{{extra}}"), 0o644))
	svc := NewTemplatesService(dir)

	got, err := svc.RenderTemplateWithFuncs("extra.html", nil, s.customFuncs())

	s.EqualError(err, `template: extra.html:1: function "extra" not defined`)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_FileNotFound() {
	got, err := s.svc.RenderTemplateWithFuncs("missing.html", nil, nil)

	s.ErrorIs(err, fs.ErrNotExist)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderTemplateWithFuncs_ExecError() {
	got, err := s.svc.RenderTemplateWithFuncs("exec_error.html", map[string]string{"Name": "Rita"}, nil)

	s.assertExecError(err, "exec_error.html")
	s.Empty(got)
}

// --- RenderHTMLContentWithFuncs ---

func (s *TemplatesServiceSuite) TestRenderHTMLContentWithFuncs_CustomOverridesAndAdds() {
	got, err := s.svc.RenderHTMLContentWithFuncs("{{toUpper .}}|{{greet .}}", "Rita", s.customFuncs())

	s.Require().NoError(err)
	s.Equal("custom:Rita|hi Rita", got)
}

func (s *TemplatesServiceSuite) TestRenderHTMLContentWithFuncs_OnlyFirstFuncMapUsed() {
	got, err := s.svc.RenderHTMLContentWithFuncs("{{extra}}", nil, s.customFuncs())

	s.EqualError(err, `template: content:1: function "extra" not defined`)
	s.Empty(got)
}

func (s *TemplatesServiceSuite) TestRenderHTMLContentWithFuncs_ExecError() {
	got, err := s.svc.RenderHTMLContentWithFuncs("{{.Name.Missing}}", map[string]string{"Name": "Rita"}, nil)

	s.assertExecError(err, "content")
	s.Empty(got)
}

// assertExecError verifies err is a text/template execution error from the named template.
// html/template returns text/template's ExecError for runtime failures.
func (s *TemplatesServiceSuite) assertExecError(err error, templateName string) {
	s.T().Helper()

	execErr, ok := errors.AsType[texttemplate.ExecError](err)
	s.Require().True(ok, "expected text/template.ExecError, got %T: %v", err, err)
	s.Equal(templateName, execErr.Name)
}
