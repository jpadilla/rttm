package main

import (
	"net/http"
)

type handler func(http.ResponseWriter, *http.Request, *Context)

func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	//create the context
	ctx, err := NewContext(req)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	defer ctx.Close()

	//run the handler
	h(w, req, ctx)
}
