package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/joho/godotenv/autoload"
	"gopkg.in/mgo.v2"
)

type Database struct {
	session *mgo.Session
}

func main() {
	session, sessionErr := mgo.Dial(os.Getenv("MONGOHQ_URL"))

	if sessionErr != nil {
		panic(sessionErr)
	}

	defer session.Close()

	session.SetSafe(&mgo.Safe{})

	db := &Database{session: session}

	http.HandleFunc("/submit", db.submitHandler)
	http.HandleFunc("/favicon.ico", iconHandler)

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
