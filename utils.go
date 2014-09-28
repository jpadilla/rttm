package main

import (
	"net/url"
	"strings"
)

func IsValidURL(str string) bool {
	u, err := url.Parse(str)

	if err != nil {
		return false
	}

	if u.Scheme == "" {
		return false
	}

	if u.Host == "" {
		return false
	}

	return true
}

func SmartTruncate(str string, length int, suffix string) string {
	if len(str) <= length {
		return str
	}

	splitted := strings.Split(str[:length+1], " ")

	return strings.Join(splitted[:len(splitted)-1], " ") + suffix
}
