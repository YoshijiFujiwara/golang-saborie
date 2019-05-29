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
	metooController := controllers.MetooController{}
	loveController := controllers.LoveController{}

	// ルーティング
	router := mux.NewRouter()
	prefix := "/api/v1"

	// auth
	router.HandleFunc(prefix + "/users/register", userController.Register()).Methods("POST")
	router.HandleFunc(prefix + "/users/login", userController.Login()).Methods("POST")
	router.HandleFunc(prefix + "/users/me", userController.TokenVerifyMiddleware(userController.Me())).Methods("GET")

	// sabota
	router.HandleFunc(prefix + "/sabotas", sabotaController.Index()).Methods("GET")
	router.HandleFunc(prefix + "/search_sabotas", sabotaController.SearchSabotas()).Methods("POST") // 検索
	router.HandleFunc(prefix + "/sabotas", userController.TokenVerifyMiddleware(sabotaController.Store())).Methods("POST") // 認証必要
	router.HandleFunc(prefix + "/sabotas/{sabotaId}", sabotaController.Show()).Methods("GET")
	router.HandleFunc(prefix + "/sabotas/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Update())).Methods("PUT") // 認証必要
	router.HandleFunc(prefix + "/sabotas/{sabotaId}", userController.TokenVerifyMiddleware(sabotaController.Destroy())).Methods("DELETE") // 認証必要

	// comment
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/comments", commentController.Index()).Methods("GET")
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/comments", userController.TokenVerifyMiddleware(commentController.Store())).Methods("POST") // 認証必要
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/comments/{commentId}", commentController.Show()).Methods("GET")
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/comments/{commentId}", userController.TokenVerifyMiddleware(commentController.Update())).Methods("PUT") // 認証必要
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/comments/{commentId}", userController.TokenVerifyMiddleware(commentController.Destroy())).Methods("DELETE") // 認証必要

	// metoo
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/metoos", metooController.Index()).Methods("GET")
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/switch_metoo", userController.TokenVerifyMiddleware(metooController.SwitchMetoo())).Methods("PUT") // 認証必要

	// love
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/loves", loveController.Index()).Methods("GET")
	router.HandleFunc(prefix + "/sabotas/{sabotaId}/switch_love", userController.TokenVerifyMiddleware(loveController.SwitchLove())).Methods("PUT") // 認証必要

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}
