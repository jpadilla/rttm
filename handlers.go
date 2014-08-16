package main

import (
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/jpadilla/alchemy"
	"github.com/jpadilla/ivona"
	"github.com/jpadilla/rttm/services"
)

var (
	alchemyAPIKey  = os.Getenv("ALCHEMY_API_KEY")
	ivonaAccessKey = os.Getenv("IVONA_ACCESS_KEY")
	ivonaSecretKey = os.Getenv("IVONA_SECRET_KEY")
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
		db.getSubmitHandler(w, r)
	case "POST":
		db.postSubmitHandler(w, r)
	}
}

func (db *Database) getSubmitHandler(w http.ResponseWriter, r *http.Request) {
	data := &submitData{
		URL:     r.FormValue("u"),
		Success: false,
	}

	render(w, "templates/submit.html", data)
}

func (db *Database) postSubmitHandler(w http.ResponseWriter, r *http.Request) {
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

	alchemyClient := alchemy.New(alchemyAPIKey)

	log.Println("Getting title...")
	titleResponse, err := alchemyClient.GetTitle(data.URL, alchemy.GetTitleOptions{})
	if err != nil {
		data.Errors["Generic"] = "There was a problem extracting data from URL."
		render(w, "templates/submit.html", data)
		return
	}

	log.Println("Getting text...")
	textResponse, err := alchemyClient.GetText(data.URL, alchemy.GetTextOptions{})
	if err != nil {
		data.Errors["Generic"] = "There was a problem extracting data from URL."
		render(w, "templates/submit.html", data)
		return
	}

	data.URL = ""
	data.Success = true
	render(w, "templates/submit.html", data)

	log.Println("Running goroutine...")
	go func() {
		log.Println("CreateSpeech")
		ivonaClient := ivona.New(ivonaAccessKey, ivonaSecretKey)
		ivonaOptions := ivona.NewSpeechOptions(textResponse.Text)
		ir, err := ivonaClient.CreateSpeech(ivonaOptions)

		if err != nil {
			log.Println(err)
			return
		}

		log.Println("UploadPublicFile")
		path := ir.RequestId + ".mp3"
		mp3Url := services.UploadPublicFile(path, ir.Audio, ir.ContentType)

		log.Println("SendSMS")
		go services.SendSMS(phone, titleResponse.Title+"\n"+mp3Url)

		log.Println("Store request")
		c := db.session.DB("").C("requests")
		c.Insert(&Request{
			URL:      url,
			Phone:    phone,
			Title:    titleResponse.Title,
			AudioURL: mp3Url,
			Text:     textResponse.Text,
		})
	}()
}

func (db *Database) twilioCallbackHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		body := ""
		to := r.FormValue("To")
		accountSid := r.FormValue("AccountSid")

		if to != os.Getenv("TWILIO_NUMBER") || accountSid != os.Getenv("TWILIO_ACCOUNT_SID") {
			http.Error(w, "Invalid number or Sid", http.StatusInternalServerError)
			return
		}

		words := regexp.MustCompile(`(\s+)`).Split(r.FormValue("Body"), -1)

		// Look for and extract valid URL in Body
		for _, word := range words {
			if IsValidURL(word) {
				body = word
			}
		}

		if body == "" {
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		data := &submitData{
			URL:   body,
			Phone: r.FormValue("From"),
		}

		if data.validate() == false {
			log.Println("Errors", data.Errors)
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		log.Println("Running goroutine...")
		go func() {
			alchemyClient := alchemy.New(alchemyAPIKey)

			log.Println("Getting title...")
			titleResponse, err := alchemyClient.GetTitle(data.URL, alchemy.GetTitleOptions{})
			if err != nil {
				return
			}

			log.Println("Getting text...")
			textResponse, err := alchemyClient.GetText(data.URL, alchemy.GetTextOptions{})
			if err != nil {
				log.Println(err)
				return
			}

			log.Println("Creating speech...")
			ivonaClient := ivona.New(ivonaAccessKey, ivonaSecretKey)
			ivonaOptions := ivona.NewSpeechOptions(textResponse.Text)
			ir, err := ivonaClient.CreateSpeech(ivonaOptions)

			if err != nil {
				log.Println(err)
				return
			}

			log.Println("Uploading public file....")
			path := ir.RequestId + ".mp3"
			mp3Url := services.UploadPublicFile(path, ir.Audio, ir.ContentType)

			log.Println("Uploaded public file to ", mp3Url)

			log.Println("Sending SMS...")
			go services.SendSMS(data.Phone, titleResponse.Title+"\n"+mp3Url)

			log.Println("Storing request...")
			c := db.session.DB("").C("requests")
			c.Insert(&Request{
				URL:      data.URL,
				Phone:    data.Phone,
				Title:    titleResponse.Title,
				AudioURL: mp3Url,
				Text:     textResponse.Text,
			})
		}()
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
