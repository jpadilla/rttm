package services

import (
	ivona "github.com/jpadilla/ivona-go"
	"log"
	"os"
)

var (
	ivonaClient    *ivona.Ivona
	ivonaAccessKey = os.Getenv("IVONA_ACCESS_KEY")
	ivonaSecretKey = os.Getenv("IVONA_SECRET_KEY")
)

func init() {
	ivonaClient = ivona.New(ivonaAccessKey, ivonaSecretKey)
}

// TextToSpeech paginates text and returns appended audio bytes
func TextToSpeech(text string) ([]byte, error) {
	log.Println("Splitting text...")
	max := 4096
	count := 0
	bucket := 0
	results := make([]string, (len(text)/max)+1)
	a := []rune(text)

	for _, s := range a {
		results[bucket] = results[bucket] + string(s)
		count++

		if count == max {
			bucket++
			count = 0
		}
	}

	playlist := make([]byte, len(results))

	for _, s := range results {
		log.Println("Creating speech...")

		ivonaOptions := ivona.NewSpeechOptions(s)
		ir, err := ivonaClient.CreateSpeech(ivonaOptions)

		log.Println("RequestID = ", ir.RequestID)

		if err != nil {
			log.Println(err)
			return nil, err
		}

		playlist = append(playlist, ir.Audio...)
	}

	return playlist, nil
}
