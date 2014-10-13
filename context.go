package main

import (
	"net/http"

	"gopkg.in/mgo.v2"
)

type Context struct {
	Database          *mgo.Database
	RequestCollection *mgo.Collection
}

func (c *Context) Close() {
	c.Database.Session.Close()
}

func NewContext(req *http.Request) (*Context, error) {
	return &Context{
		Database:          session.Clone().DB(""),
		RequestCollection: session.DB("").C("requests"),
	}, nil
}
