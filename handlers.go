package main

import (
	"encoding/xml"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/feeds"
	"github.com/gorilla/mux"
)

const (
	itunesRFC822 = "Mon, 2 Jan 2006 15:04:05 MST"
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

func SubmitHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		getSubmitHandler(w, r)
	case "POST":
		postSubmitHandler(w, r)
	}
}

func getSubmitHandler(w http.ResponseWriter, r *http.Request) {
	data := &submitData{
		URL:     r.FormValue("u"),
		Success: false,
	}

	render(w, "templates/submit.html", data)
}

func postSubmitHandler(w http.ResponseWriter, r *http.Request) {
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

	data.URL = ""
	data.Success = true
	render(w, "templates/submit.html", data)

	log.Println("Running goroutine...")
	go CreateRequest(url, phone)
}

func TwilioCallbackHandler(w http.ResponseWriter, r *http.Request) {
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
		go CreateRequest(data.URL, data.Phone)
	}
}

func ViewHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	result, err := GetRequestById(params["id"])
	if err != nil {
		log.Println("Errors", err)
		http.NotFound(w, r)
		return
	}

	render(w, "templates/view.html", result)
}

func FeedHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	requests, err := FindRequestsByPhone(params["phone"])
	log.Println(requests, err)
	if err != nil {
		log.Println("Not found", err)
		http.NotFound(w, r)
		return
	}

	feed := &feeds.RssFeed{
		Title:       "RTTM",
		Link:        "http://rttm.herokuapp.com",
		Description: "Read this to me",
		PubDate:     time.Now().Format(itunesRFC822),
	}

	for _, request := range requests {
		item := &feeds.RssItem{
			Title:       request.Post.Title,
			Link:        request.Post.URL,
			Description: request.Post.GetShortDescription(),
			PubDate:     request.CreatedAt.Format(itunesRFC822),
			Enclosure: &feeds.RssEnclosure{
				Url:    request.Post.AudioURL,
				Length: strconv.Itoa(request.Post.Length),
				Type:   "audio/mpeg",
			},
		}

		feed.Items = append(feed.Items, item)
	}

	x := feed.FeedXml()

	// write default xml header, without the newline
	if _, err := w.Write([]byte(xml.Header[:len(xml.Header)-1])); err != nil {
		renderError(w, err)
		return
	}

	e := xml.NewEncoder(w)
	e.Indent("", "  ")

	e.Encode(x)
}

func IconHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
	return
}

func render(w http.ResponseWriter, filename string, data interface{}) {
	tmpl, err := template.ParseFiles(filename)

	if err != nil {
		renderError(w, err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		renderError(w, err)
	}
}

func renderError(w http.ResponseWriter, err error) {
	log.Println(err)
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
