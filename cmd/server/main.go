package main

import (
	"buildberry/internal/config"
	"buildberry/internal/db"
	"fmt"
	"log"
	"net/http"
)

func main() {
	cfg := config.LoadConfig()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong")
	})

	fmt.Println(cfg.PORT)
	fmt.Println(cfg.DBUSER)
	fmt.Println(cfg.DBPASSWORD)
	fmt.Println(cfg.DBHOST)
	fmt.Println(cfg.DBPORT)
	fmt.Println(cfg.DBNAME)
	fmt.Println("app started")

	log.Println("server running on :8080")
	http.ListenAndServe(":8080", nil)

}
