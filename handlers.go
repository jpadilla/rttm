package main

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jpadilla/alchemy"
	"github.com/jpadilla/ttsapi"
	"github.com/subosito/twilio"
)

var (
	alchemyAPIKey    = os.Getenv("ALCHEMY_API_KEY")
	twilioAccountSID = os.Getenv("TWILIO_ACCOUNT_SID")
	twilioAuthToken  = os.Getenv("TWILIO_AUTH_TOKEN")
	twilioNumber     = os.Getenv("TWILIO_NUMBER")
)

type submitData struct {
	URL    string
	Title  string
	Phone  string
	Errors map[string]string
}

func (data *submitData) validate() bool {
	data.Errors = make(map[string]string)

	// Validate Number
	if strings.TrimSpace(data.Phone) == "" {
		data.Errors["Phone"] = "Required"
	}

	// Validate URL
	if strings.TrimSpace(data.URL) == "" {
		data.Errors["URL"] = "Required"
	}

	if _, err := url.ParseRequestURI(data.URL); err != nil {
		data.Errors["URL"] = "Invalid URL"
	}

	return len(data.Errors) == 0
}

func (db *Database) submitHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		data := &submitData{
			URL:   r.FormValue("u"),
			Title: r.FormValue("t"),
		}

		render(w, "templates/submit.html", data)
	case "POST":
		data := &submitData{
			URL:   r.FormValue("url"),
			Phone: r.FormValue("phone"),
			Title: r.FormValue("title"),
		}

		if data.validate() == false {
			render(w, "templates/submit.html", data)
			return
		}

		client := alchemy.New(alchemyAPIKey)

		options := alchemy.Options{}
		response, err := client.ExtractClean(data.URL, options)

		if err != nil {
			data.Errors["URL"] = "There was a problem extracting data from URL."
			render(w, "templates/submit.html", data)
			return
		}

		mp3Url, err := ttsapi.GetSpeech(response.Text)

		if err != nil {
			data.Errors["URL"] = "There was a problem converting text to speech."
			render(w, "templates/submit.html", data)
			return
		}

		fmt.Println(mp3Url)

		// Initialize twilio client
		twilioClient := twilio.NewClient(twilioAccountSID, twilioAuthToken, nil)

		// Send Message
		params := twilio.MessageParams{
			Body: mp3Url,
		}

		twilioMessage, twilioResponse, err := twilioClient.Messages.Send(twilioNumber, data.Phone, params)

		fmt.Println(twilioMessage, twilioResponse, err)

		if err != nil {
			data.Errors["Phone"] = "There was a problem delivering SMS to phone number."
			render(w, "templates/submit.html", data)
			return
		}

		c := db.session.DB("rttm").C("requests")
		err = c.Insert(&Request{
			URL:   r.FormValue("url"),
			Phone: r.FormValue("phone"),
			Title: r.FormValue("title"),
		})

		if err != nil {
    	panic(err)
    }

		render(w, "templates/submit.html", nil)
	}
}

func iconHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
	return
}

func render(w http.ResponseWriter, filename string, data interface{}) {
	tmpl, err := template.ParseFiles(filename)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
