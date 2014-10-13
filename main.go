package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/mgo.v2"
)

var (
	session  *mgo.Session
	database string
)

func main() {
	// Configure database
	var err error
	session, err = mgo.Dial(os.Getenv("MONGOHQ_URL"))
	if err != nil {
		panic(err)
	}
	session.SetSafe(&mgo.Safe{})
	database = session.DB("").Name

	// Configure router
	router := mux.NewRouter()
	router.Handle("/feed/{phone}", handler(FeedHandler)).Methods("GET")
	router.Handle("/submit", handler(SubmitHandler)).Methods("GET", "POST")
	router.Handle("/twilio/callback", handler(TwilioCallbackHandler)).Methods("POST")
	router.Handle("/favicon.ico", handler(IconHandler)).Methods("GET")
	router.Handle("/{id}", handler(ViewHandler)).Methods("GET")

	http.Handle("/", router)

	if err = http.ListenAndServe(getPort(), nil); err != nil {
		log.Fatal("ListenAndServe: ", err)
		return
	}
}

func getPort() string {
	var port = os.Getenv("PORT")

	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
	}

	log.Println("INFO: Listening to http://localhost:" + port)

	return ":" + port
}
