package main

import (
	"fmt"
	"html/template"
	"strings"
	"time"
)

type Request struct {
	URL       string
	Title     string
	Text      string
	Phone     string
	AudioURL  string
	Length    int
	CreatedAt time.Time
}

func (r Request) GetReadableText() template.HTML {
	s := strings.Replace(r.Text, "\n", "<br />", -1)
	return template.HTML(s)
}

func (r Request) GetShortDescription() string {
	var numRunes = 0
	text := strings.TrimSpace(r.Text)

	for index := range text {
		numRunes++
		if numRunes > 120 {
			return fmt.Sprintf("%v...", text[:index])
		}
	}

	return r.Text
}
