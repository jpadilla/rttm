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
	PostCollection    *mgo.Collection
	RequestCollection *mgo.Collection
)

func main() {
	var err error

	// Configure database
	session, err := mgo.Dial(os.Getenv("MONGOHQ_URL"))
	if err != nil {
		panic(err)
	}
	session.SetSafe(&mgo.Safe{})
	defer session.Close()

	PostCollection = session.DB("").C("posts")
	RequestCollection = session.DB("").C("requests")

	// Configure router
	router := mux.NewRouter()
	router.HandleFunc("/api/rttm", APIHandler).Methods("POST")
	router.HandleFunc("/feed/{phone}", FeedHandler).Methods("GET")
	router.HandleFunc("/submit", SubmitHandler).Methods("GET", "POST")
	router.HandleFunc("/twilio/callback", TwilioCallbackHandler).Methods("POST")
	router.HandleFunc("/favicon.ico", IconHandler).Methods("GET")
	router.HandleFunc("/{id}", ViewHandler).Methods("GET")

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
