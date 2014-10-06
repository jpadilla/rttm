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
	Length   int
}

func (r Request) ReadableText() template.HTML {
	s := strings.Replace(r.Text, "\n", "<br />", -1)
	return template.HTML(s)
}
