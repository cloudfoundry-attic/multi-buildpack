package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func main() {
	http.HandleFunc("/", hello)
	fmt.Println("listening...")
	err := http.ListenAndServe(":"+os.Getenv("PORT"), nil)
	if err != nil {
		panic(err)
	}
}

func hello(res http.ResponseWriter, req *http.Request) {
	rubyVersion, err := exec.Command("ruby", "-v").Output()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(res, "The ruby version is: %s\n", rubyVersion)
}
