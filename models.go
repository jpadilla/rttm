package main

import (
	"html/template"
	"strings"
)

type Request struct {
	URL      string
	Title    string
	Text     string
	Phone    string
	AudioURL string
}

func (r Request) ReadableText() string {
	safe := template.HTMLEscapeString(r.Text)
	return strings.Replace(safe, "\n", "<br />", -1)
}
