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
	URL     string
	Title   string
	Phone   string
	Errors  map[string]string
	Success bool
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
			URL:     r.FormValue("u"),
			Title:   r.FormValue("t"),
			Success: false,
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
			data.Errors["Generic"] = "There was a problem extracting data from URL."
			render(w, "templates/submit.html", data)
			return
		}

		mp3Url, err := ttsapi.GetSpeech(response.Text)

		if err != nil {
			data.Errors["Generic"] = "There was a problem converting text to speech."
			render(w, "templates/submit.html", data)
			return
		}

		fmt.Println(mp3Url)

		c := db.session.DB("").C("requests")
		c.Insert(&Request{
			URL:      r.FormValue("url"),
			Phone:    r.FormValue("phone"),
			Title:    r.FormValue("title"),
			AudioURL: mp3Url,
		})

		if err = sendSMS(mp3Url, data.Phone); err != nil {
			data.Errors["Phone"] = "There was a problem delivering SMS to phone number."
			return
		}

		data.Success = true

		render(w, "templates/submit.html", data)
	}
}

func sendSMS(body string, phone string) error {
	// Initialize twilio client
	twilioClient := twilio.NewClient(twilioAccountSID, twilioAuthToken, nil)

	// Send Message
	params := twilio.MessageParams{
		Body: body,
	}

	twilioMessage, twilioResponse, err := twilioClient.Messages.Send(twilioNumber, phone, params)

	fmt.Println(twilioMessage, twilioResponse, err)

	return err
}

func (db *Database) twilioCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		to := r.FormValue("To")
		accountSid := r.FormValue("AccountSid")
		from := r.FormValue("From")
		body := r.FormValue("Body")

		if to != os.Getenv("TWILIO_NUMBER") || accountSid != os.Getenv("TWILIO_ACCOUNT_SID") {
			http.Error(w, "Invalid number or Sid", http.StatusInternalServerError)
			return
		}

		data := &submitData{
			URL:   r.FormValue("Body"),
			Phone: r.FormValue("From"),
		}

		if data.validate() == false {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		fmt.Println(from)
		fmt.Println(body)

		client := alchemy.New(alchemyAPIKey)

		options := alchemy.Options{}
		response, err := client.ExtractClean(data.URL, options)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		mp3Url, err := ttsapi.GetSpeech(response.Text)

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Println(mp3Url)

		c := db.session.DB("").C("requests")
		c.Insert(&Request{
			URL:      data.URL,
			Phone:    data.Phone,
			AudioURL: mp3Url,
		})

		if err = sendSMS(mp3Url, data.Phone); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
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
