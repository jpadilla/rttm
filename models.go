package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"time"

	"github.com/jpadilla/rttm/services"
	"gopkg.in/mgo.v2/bson"
)

type Post struct {
	Id        bson.ObjectId `bson:"_id"`
	AudioURL  string
	Length    int
	CreatedAt time.Time

	OriginalURL     string    `json:"original_url"`
	URL             string    `json:"url"`
	Type            string    `json:"type"`
	Safe            bool      `json:"safe"`
	SafeType        string    `json:"safe_type,omitempty"`
	SafeMessage     string    `json:"safe_message,omitempty"`
	CacheAge        int       `json:"cache_age,omitempty"`
	ProviderName    string    `json:"provider_name"`
	ProviderURL     string    `json:"provider_url"`
	ProviderDisplay string    `json:"provider_display"`
	FaviconURL      string    `json:"favicon_url"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	Authors         []Author  `json:"authors"`
	Media           Media     `json:"media"`
	Published       int64     `json:"published"`
	Offset          int64     `json:"offset"`
	Lead            string    `json:"lead"`
	Content         string    `json:"content"`
	Keywords        []Keyword `json:"keywords"`
	Entities        []Entity  `json:"entities"`
	Images          []Image   `json:"images"`
}

type Author struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Media struct {
	Type   string `json:"type"`
	URL    string `json:"url,omitempty"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	HTML   string `json:"html,omitempty"`
}

type Keyword struct {
	Score int    `json:"score"`
	Name  string `json:"name"`
}

type Entity struct {
	Count int    `json:"count"`
	Name  string `json:"name"`
}

type Image struct {
	Caption string  `json:"caption"`
	URL     string  `json:"url"`
	Width   int     `json:"width"`
	Height  int     `json:"height"`
	Colors  []Color `json:"colors"`
	Entropy float64 `json:"entropy"`
	Size    int     `json:"size"`
}

type Color struct {
	Color  []int   `json:"color"`
	Weight float64 `json:"weight"`
}

type Request struct {
	Id        bson.ObjectId `bson:"_id"`
	PostId    bson.ObjectId `bson:"post_id"`
	Post      *Post         `bson:"-"`
	Phone     string
	CreatedAt time.Time
}

func GetPostById(id bson.ObjectId) (*Post, error) {
	post := &Post{}
	err := PostCollection.FindId(id).One(&post)

	if err != nil {
		return nil, err
	}

	return post, err
}

func GetPostByURL(url string) (*Post, error) {
	post := &Post{}
	err := PostCollection.Find(bson.M{"url": url}).One(&post)

	if err != nil {
		return nil, err
	}

	return post, err
}

func (p Post) GetReadableText() template.HTML {
	return template.HTML(p.Content)
}

func (p Post) GetShortDescription() string {
	return p.Description
}

func FindRequestsByPhone(phone string) ([]Request, error) {
	var requests []Request

	err := RequestCollection.Find(bson.M{"phone": phone}).All(&requests)

	if err != nil {
		return nil, err
	}

	for i := range requests {
		post, err := GetPostById(requests[i].PostId)

		if err != nil {
			return nil, err
		}

		requests[i].Post = post
	}

	return requests, nil
}

func GetRequestById(id string) (*Request, error) {
	if bson.IsObjectIdHex(id) == false {
		return nil, fmt.Errorf("Invalid Id: %s", id)
	}

	request := &Request{}
	err := RequestCollection.FindId(bson.ObjectIdHex(id)).One(&request)

	if err != nil {
		return nil, err
	}

	return request, err
}

func CreatePost(url string) (*Post, error) {
	log.Println("Extracting...")
	extractResponse, err := services.Extract(url)
	if err != nil {
		log.Println(extractResponse.ErrorMessage)
		log.Println(extractResponse.ErrorCode)
		return nil, err
	}

	log.Println("Getting playlist...")
	playlist, err := services.TextToSpeech(extractResponse.Content)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	log.Println("UploadPublicFile")
	path := fmt.Sprintf("%d.mp3", int32(time.Now().Unix()))
	mp3Url := services.UploadPublicFile(path, playlist, "audio/mpeg")

	log.Println("Uploaded public file to ", mp3Url)

	log.Println("Creating Post...")
	b, err := json.Marshal(extractResponse)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	post := &Post{
		Id:        bson.NewObjectId(),
		AudioURL:  mp3Url,
		Length:    len(playlist),
		CreatedAt: time.Now(),
	}

	if err = json.Unmarshal(b, post); err != nil {
		log.Println(err)
		return nil, err
	}

	if err = PostCollection.Insert(post); err != nil {
		log.Println(err)
		return nil, err
	}

	return post, nil
}

func CreateRequest(url string, phone string) {
	post, err := GetPostByURL(url)

	if err != nil {
		post, err = CreatePost(url)

		if err != nil {
			return
		}
	}

	log.Println("Creating Request...")
	request := &Request{
		Id:        bson.NewObjectId(),
		PostId:    post.Id,
		Phone:     phone,
		CreatedAt: time.Now(),
	}

	if err = RequestCollection.Insert(request); err != nil {
		log.Println(err)
		return
	}

	log.Println("Sending SMS...")
	message := post.Title + "\n" + post.AudioURL
	go services.SendSMS(phone, message)
}
