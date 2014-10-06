package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/mgo.v2"
)

type Database struct {
	Session           *mgo.Session
	RequestCollection *mgo.Collection
}

func main() {
	session, sessionErr := mgo.Dial(os.Getenv("MONGOHQ_URL"))

	if sessionErr != nil {
		panic(sessionErr)
	}

	defer session.Close()

	session.SetSafe(&mgo.Safe{})

	db := &Database{
		Session:           session,
		RequestCollection: session.DB("").C("requests"),
	}

	router := mux.NewRouter()
	router.HandleFunc("/submit", db.submitHandler).Methods("GET", "POST")
	router.HandleFunc("/twilio/callback", db.twilioCallbackHandler).Methods("POST")
	router.HandleFunc("/favicon.ico", iconHandler).Methods("GET")
	router.HandleFunc("/{id}", db.viewHandler).Methods("GET")

	http.Handle("/", router)

	err := http.ListenAndServe(getPort(), nil)

	if err != nil {
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
