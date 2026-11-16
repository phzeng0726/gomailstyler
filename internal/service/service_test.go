package service

import (
	"io"
	"io/fs"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ServicesSuite struct {
	suite.Suite
}

func TestServicesSuite(t *testing.T) {
	suite.Run(t, new(ServicesSuite))
}

func (s *ServicesSuite) SetupSuite() {
	log.SetOutput(io.Discard)
}

func (s *ServicesSuite) TearDownSuite() {
	log.SetOutput(os.Stderr)
}

func (s *ServicesSuite) TestNewServices_WiresConcreteServices() {
	svc := NewServices(Deps{TmplPath: "testdata", CSSPath: "testdata"})

	tmplSvc, ok := svc.Templates.(*TemplatesService)
	s.Require().True(ok, "Templates should be *TemplatesService, got %T", svc.Templates)
	s.Equal("testdata", tmplSvc.templatePath)

	cssSvc, ok := svc.CSSTools.(*CSSToolsService)
	s.Require().True(ok, "CSSTools should be *CSSToolsService, got %T", svc.CSSTools)
	s.Equal("testdata", cssSvc.tmplPath)
	s.Same(tmplSvc, cssSvc.tmplSvc)
}

func (s *ServicesSuite) TestNewServices_CSSResolvedAgainstCSSPath() {
	svc := NewServices(Deps{TmplPath: "testdata", CSSPath: s.T().TempDir()})

	// Templates still resolve against TmplPath
	got, err := svc.Templates.RenderTemplate("basic.html", map[string]string{"Name": "Rita"})
	s.Require().NoError(err)
	s.Equal("<p>Hello, Rita!</p>", got)

	// CSS resolves against CSSPath, which does not contain styles.css
	got, err = svc.CSSTools.RenderTemplateWithCSS("styled.html", "styles.css", map[string]string{"Name": "Rita"})
	s.ErrorIs(err, fs.ErrNotExist)
	s.True(strings.HasPrefix(err.Error(), "failed to read CSS file styles.css: "), err.Error())
	s.Empty(got)
}
