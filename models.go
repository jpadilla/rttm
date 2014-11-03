package main

import (
	"fmt"
	"html/template"
	"log"
	"os"
	"strings"
	"time"

	alchemyapi "github.com/jpadilla/alchemyapi-go"
	"github.com/jpadilla/rttm/services"
	"gopkg.in/mgo.v2/bson"
)

type Post struct {
	Id        bson.ObjectId `bson:"_id"`
	URL       string
	Title     string
	Text      string
	AudioURL  string
	Length    int
	CreatedAt time.Time
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
func (p Post) Create() {
	p.Id = bson.NewObjectId()
	p.CreatedAt = time.Now()

	PostCollection.Insert(p)
}

func (p Post) GetReadableText() template.HTML {
	s := strings.Replace(p.Text, "\n", "<br />", -1)
	return template.HTML(s)
}

func (p Post) GetShortDescription() string {
	var numRunes = 0
	text := strings.TrimSpace(p.Text)

	for index := range text {
		numRunes++
		if numRunes > 120 {
			return fmt.Sprintf("%v...", text[:index])
		}
	}

	return p.Text
}

func (r Request) Create() {
	r.Id = bson.NewObjectId()
	r.CreatedAt = time.Now()

	RequestCollection.Insert(r)
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

func CreateRequest(url string, phone string) {
	alchemyClient := alchemyapi.New(os.Getenv("ALCHEMY_API_KEY"))
	message := ""

	post, err := GetPostByURL(url)

	if err != nil {
		log.Println("Getting title...")
		titleResponse, err := alchemyClient.GetTitle(url, alchemyapi.GetTitleOptions{})
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("Getting text...")
		textResponse, err := alchemyClient.GetText(url, alchemyapi.GetTextOptions{})
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

		log.Println("Creating Post...")
		post = &Post{
			URL:      url,
			Title:    titleResponse.Title,
			Text:     strings.TrimSpace(textResponse.Text),
			AudioURL: mp3Url,
			Length:   len(playlist),
		}
		post.Create()

		message = titleResponse.Title + "\n" + mp3Url
	} else {
		message = post.Title + "\n" + post.AudioURL
	}

	log.Println("Creating Request...")
	request := &Request{
		PostId: post.Id,
		Phone:  phone,
	}
	request.Create()

	log.Println("Sending SMS...")
	go services.SendSMS(phone, message)
}
