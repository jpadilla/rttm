package services

import (
	"os"

	"github.com/poptip/embedly"
)

func Extract(url string) (*embedly.Response, error) {
	c := embedly.NewClient(os.Getenv("EMBEDLY_API_KEY"))
	options := embedly.Options{}

	return c.ExtractOne(url, options)
}
