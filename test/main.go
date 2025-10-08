package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {

	mux := http.NewServeMux()

	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		data, err := io.ReadAll(r.Body)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println(string(data))

		w.Write([]byte("testing"))
	}))

	if err := http.ListenAndServe(":4000", mux); err != nil {
		fmt.Println("HTTP server failed:", err)
	}
}
