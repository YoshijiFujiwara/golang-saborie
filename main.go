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
	userController := controllers.UserController{}
	sabotaController := controllers.SabotaController{}

	// ルーティング
	router := mux.NewRouter()

	// auth
	router.HandleFunc("/signup", userController.Signup()).Methods("POST")
	router.HandleFunc("/login", userController.Login()).Methods("POST")

	// sabota
	router.HandleFunc("/sabota", sabotaController.Index()).Methods("GET")
	router.HandleFunc("/sabota", userController.TokenVerifyMiddleware(sabotaController.Store())).Methods("POST") // 認証必要
	router.HandleFunc("/sabota/{sabotaId}", sabotaController.Show()).Methods("GET")
	router.HandleFunc("/sabota/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Update())).Methods("PUT")
	router.HandleFunc("/sabota/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Destroy())).Methods("DELETE")

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}
