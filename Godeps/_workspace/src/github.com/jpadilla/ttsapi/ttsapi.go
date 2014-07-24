package ttsapi

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

const (
	TTSAPI = "http://tts-api.com/tts.mp3"
)

func GetSpeech(text string) (string, error) {
	addr := TTSAPI + "?"

	params := url.Values{}
	params.Add("q", text)
	params.Add("return_url", "1")

	addr += params.Encode()

	// Make request
	resp, err := http.Get(addr)

	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode >= 500 {
		return "", fmt.Errorf("Got non 200 status code: %s %q", resp.Status, body)
	}

	return string(body), nil
}
