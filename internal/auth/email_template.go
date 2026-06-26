package auth

import (
	"bytes"
	"embed"
	htmltemplate "html/template"
	texttemplate "text/template"
)

// templatesFS holds the email templates. Files live under templates/ so they
// can later be organized by language or A/B variant without touching code.
//
//go:embed templates/*.tmpl
var templatesFS embed.FS

var (
	magicLinkTextTmpl = texttemplate.Must(texttemplate.ParseFS(templatesFS, "templates/magic_link.txt.tmpl"))
	magicLinkHTMLTmpl = htmltemplate.Must(htmltemplate.ParseFS(templatesFS, "templates/magic_link.html.tmpl"))
)

// magicLinkData is the data passed to the magic-link email templates.
type magicLinkData struct {
	Link   string
	Expiry string
}

// renderMagicLink renders the plain-text and HTML bodies of the magic-link
// email. The HTML template auto-escapes its data.
func renderMagicLink(data magicLinkData) (text, html string, err error) {
	var textBuf, htmlBuf bytes.Buffer
	if err := magicLinkTextTmpl.Execute(&textBuf, data); err != nil {
		return "", "", err
	}
	if err := magicLinkHTMLTmpl.Execute(&htmlBuf, data); err != nil {
		return "", "", err
	}
	return textBuf.String(), htmlBuf.String(), nil
}
