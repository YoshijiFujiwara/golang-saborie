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
	commentController := controllers.CommentController{}

	// ルーティング
	router := mux.NewRouter()

	// auth
	router.HandleFunc("/signup", userController.Signup()).Methods("POST")
	router.HandleFunc("/login", userController.Login()).Methods("POST")

	// sabota
	router.HandleFunc("/sabotas", sabotaController.Index()).Methods("GET")
	router.HandleFunc("/sabotas", userController.TokenVerifyMiddleware(sabotaController.Store())).Methods("POST") // 認証必要
	router.HandleFunc("/sabotas/{sabotaId}", sabotaController.Show()).Methods("GET")
	router.HandleFunc("/sabotas/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Update())).Methods("PUT") // 認証必要
	router.HandleFunc("/sabotas/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Destroy())).Methods("DELETE") // 認証必要

	// comment
	router.HandleFunc("/sabotas/{sabotaId}/comments", commentController.Index()).Methods("GET")
	router.HandleFunc("/sabotas/{sabotaId}/comments", userController.TokenVerifyMiddleware(commentController.Store())).Methods("POST") // 認証必要
	router.HandleFunc("/sabotas/{sabotaId}/comments/{commentId}", commentController.Show()).Methods("GET")
	router.HandleFunc("/sabotas/{sabotaId}/comments/{commentId}", userController.TokenVerifyMiddleware(commentController.Update())).Methods("PUT") // 認証必要
	router.HandleFunc("/sabotas/{sabotaId}/comments/{commentId}", userController.TokenVerifyMiddleware(commentController.Destroy())).Methods("DELETE") // 認証必要

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}
