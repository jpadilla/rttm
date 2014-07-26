package main

import (
	"html/template"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/jpadilla/alchemy"
	"github.com/jpadilla/rttm/services"
	"github.com/jpadilla/ttsapi"
)

var (
	alchemyAPIKey = os.Getenv("ALCHEMY_API_KEY")
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
			Success: false,
		}

		render(w, "templates/submit.html", data)
		return
	case "POST":
		url := r.FormValue("url")
		phone := r.FormValue("phone")

		data := &submitData{
			URL:   url,
			Phone: phone,
		}

		if data.validate() == false {
			render(w, "templates/submit.html", data)
			return
		}

		titleResponse, err := client.GetTitle(data.URL, alchemy.GetTitleOptions{})

		if err != nil {
			data.Errors["Generic"] = "There was a problem extracting data from URL."
			render(w, "templates/submit.html", data)
			return
		}

		textResponse, err := client.GetText(data.URL, alchemy.GetTextOptions{})

		if err != nil {
			data.Errors["Generic"] = "There was a problem extracting data from URL."
			render(w, "templates/submit.html", data)
			return
		}

		mp3Url, err := ttsapi.GetSpeech(textResponse.Text)

		if err != nil {
			data.Errors["Generic"] = "There was a problem converting text to speech."
			render(w, "templates/submit.html", data)
			return
		}

		data.URL = ""
		data.Success = true
		render(w, "templates/submit.html", data)

		c := db.session.DB("").C("requests")
		c.Insert(&Request{
			URL:      url,
			Phone:    phone,
			Title:    title,
			AudioURL: mp3Url,
		})

		go services.SendSMS(phone, title + "\n" + mp3Url)
	}
}

func (db *Database) twilioCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		to := r.FormValue("To")
		accountSid := r.FormValue("AccountSid")

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

		go func(url string, phone string) {
			client := alchemy.New(alchemyAPIKey)

			titleResponse, err := client.GetTitle(url, alchemy.GetTitleOptions{})

			if err != nil {
				return
			}

			textResponse, err := client.GetText(url, alchemy.GetTextOptions{})

			if err != nil {
				fmt.Println(err)
				return
			}

			mp3Url, err := ttsapi.GetSpeech(textResponse.Text)

			if err != nil {
				fmt.Println(err)
				return
			}

			c := db.session.DB("").C("requests")
			c.Insert(&Request{
				URL:      url,
				Title:    titleResponse.Title,
				Phone:    phone,
				AudioURL: mp3Url,
			})

			go services.SendSMS(phone, titleResponse.Title + "\n" + mp3Url)
		}(data.URL, data.Phone)
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
