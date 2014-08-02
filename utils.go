package main

import (
	"net/url"
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

	return false
}
