package main

import (
	"encoding/json"
	"net/http"
)

type Response struct {
	Endpoint     string `json:"endpoint"`
	Year         int    `json:"year"`
	Organization string `json:"organization"`
}

func cache(w http.ResponseWriter, r *http.Request) {

	w.Header().Add("cachable", "ttl=30")

	response := Response{
		Endpoint:     "Cachable",
		Year:         2021,
		Organization: "GoTurkiye",
	}

	b, _ := json.Marshal(response)
	w.Write(b)

}

func noCache(w http.ResponseWriter, r *http.Request) {

	response := Response{
		Endpoint:     "NOT cachable",
		Year:         2021,
		Organization: "GoTurkiye",
	}

	b, _ := json.Marshal(response)
	w.Write(b)
}

func main() {

	http.HandleFunc("/cache", cache)
	http.HandleFunc("/no-cache", noCache)

	http.ListenAndServe(":9191", nil)

}
