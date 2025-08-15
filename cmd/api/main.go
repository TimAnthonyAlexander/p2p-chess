package main

import (
	"net/http"

	ihttp "p2p-chess/internal/http"
)

func main() {
	router := ihttp.NewRouter()
	http.ListenAndServe(":8080", router)
}
