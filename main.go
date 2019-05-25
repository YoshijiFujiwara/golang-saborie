package main

import (
	"log"
	"net/http"
	"portfolio/saborie/controllers"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {
	controller := controllers.Controller{}

	// ルーティング
	router := mux.NewRouter()

	router.HandleFunc("/signup", controller.Signup()).Methods("POST")
	router.HandleFunc("/login", controller.Login()).Methods("POST")
	router.HandleFunc("/protected", controller.TokenVerifyMiddleware(controller.ProtectedEndpoint())).Methods("GET")
	router.HandleFunc("/sabota", controller.TokenVerifyMiddleware(controller.StoreSabota())).Methods("POST")

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}
