package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
	alchemyapi "github.com/jpadilla/alchemyapi-go"
	"github.com/jpadilla/rttm/services"
	"gopkg.in/mgo.v2/bson"
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

	if IsValidURL(data.URL) == false {
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

	alchemyClient := alchemyapi.New(alchemyAPIKey)

	log.Println("Getting title...")
	titleResponse, err := alchemyClient.GetTitle(data.URL, alchemyapi.GetTitleOptions{})
	if err != nil {
		data.Errors["Generic"] = "There was a problem extracting data from URL."
		render(w, "templates/submit.html", data)
		return
	}

	log.Println("Getting text...")
	textResponse, err := alchemyClient.GetText(data.URL, alchemyapi.GetTextOptions{})
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
		playlist, err := services.TextToSpeech(textResponse.Text)

		if err != nil {
			log.Println(err)
			return
		}

		log.Println("UploadPublicFile")
		path := fmt.Sprintf("%d.mp3", int32(time.Now().Unix()))
		mp3Url := services.UploadPublicFile(path, playlist, "audio/mpeg")

		log.Println("Uploaded public file to ", mp3Url)

		log.Println("Sending SMS...")
		go services.SendSMS(phone, titleResponse.Title+"\n"+mp3Url)

		log.Println("Storing request...")
		db.RequestCollection.Insert(&Request{
			URL:       url,
			Title:     titleResponse.Title,
			Text:      strings.TrimSpace(textResponse.Text),
			Phone:     phone,
			AudioURL:  mp3Url,
			Length:    len(playlist),
			CreatedAt: time.Now(),
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
			alchemyClient := alchemyapi.New(alchemyAPIKey)

			log.Println("Getting title...")
			titleResponse, err := alchemyClient.GetTitle(data.URL, alchemyapi.GetTitleOptions{})
			if err != nil {
				return
			}

			log.Println("Getting text...")
			textResponse, err := alchemyClient.GetText(data.URL, alchemyapi.GetTextOptions{})
			if err != nil {
				log.Println(err)
				return
			}

			log.Println("Getting playlist...")
			playlist, err := services.TextToSpeech(textResponse.Text)
			if err != nil {
				log.Println(err)
				return
			}

			log.Println("UploadPublicFile")
			path := fmt.Sprintf("%d.mp3", int32(time.Now().Unix()))
			mp3Url := services.UploadPublicFile(path, playlist, "audio/mpeg")

			log.Println("Uploaded public file to ", mp3Url)

			log.Println("Sending SMS...")
			go services.SendSMS(data.Phone, titleResponse.Title+"\n"+mp3Url)

			log.Println("Storing request...")
			db.RequestCollection.Insert(&Request{
				URL:       data.URL,
				Title:     titleResponse.Title,
				Text:      strings.TrimSpace(textResponse.Text),
				Phone:     data.Phone,
				AudioURL:  mp3Url,
				Length:    len(playlist),
				CreatedAt: time.Now(),
			})
		}()
	}
}

func (db *Database) viewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	result := Request{}

	if bson.IsObjectIdHex(params["id"]) == false {
		http.NotFound(w, r)
		return
	}

	id := bson.ObjectIdHex(params["id"])
	err := db.RequestCollection.FindId(id).One(&result)
	if err != nil {
		log.Println("Errors", err)
		http.NotFound(w, r)
		return
	}

	render(w, "templates/view.html", result)
}

func (db *Database) feedHandler(w http.ResponseWriter, r *http.Request) {
	var requests []Request
	params := mux.Vars(r)

	err := db.RequestCollection.Find(bson.M{"phone": params["phone"]}).All(&requests)

	if err != nil {
		log.Println("Errors", err)
		http.NotFound(w, r)
		return
	}

	feed := &feeds.Feed{
		Title:       "RTTM",
		Link:        &feeds.Link{Href: "http://rttm.herokuapp.com"},
		Description: "Read this to me",
		Created:     time.Now(),
	}

	items := []*feeds.Item{}

	for _, request := range requests {
		item := &feeds.Item{
			Title:       request.Title,
			Link:        &feeds.Link{Href: request.URL},
			Description: request.GetShortDescription(),
			Created:     request.CreatedAt,
		}

		items = append(items, item)
	}

	feed.Items = items

	feed.WriteRss(w)
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
