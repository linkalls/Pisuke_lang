package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		query := make(map[string]interface{})
		for k, v := range r.URL.Query() {
			if len(v) > 0 { query[k] = v[0] }
		}
		req := make(map[string]interface{})
		req["query"] = query
		returnValue := "Welcome to Pisuke Web!"
		fmt.Fprint(w, returnValue)
	})
	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		query := make(map[string]interface{})
		for k, v := range r.URL.Query() {
			if len(v) > 0 { query[k] = v[0] }
		}
		req := make(map[string]interface{})
		req["query"] = query
		var name = req["query"]["name"]
		_ = name
		returnValue := "Hello, " + name.(string)
		fmt.Fprint(w, returnValue)
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
