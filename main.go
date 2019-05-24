package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type JWT struct {
	Token string `json:"token"`
}

type Error struct {
	Message string `json:"message"`
}

func main() {
	helloWorld(os.Getenv("db_url"), os.Getenv("db_user"), os.Getenv("db_pass"))

	// ルーティング
	router := mux.NewRouter()

	router.HandleFunc("/signup", signup).Methods("POST")
	router.HandleFunc("/login", login).Methods("POST")
	router.HandleFunc("/protected", TokenVerifyMiddleWare(ProtectedEndpoint)).Methods("GET")

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func respondWithError(w http.ResponseWriter, status int, error Error) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(error)
}

func signup(w http.ResponseWriter, r *http.Request) {
	var user User
	var error Error
	// リクエスト内容のデコード
	json.NewDecoder(r.Body).Decode(&user)

	// 検証
	if user.Email == "" {
		error.Message = "メールアドレスがありません"
		respondWithError(w, http.StatusBadRequest, error)
		return
	}
	if user.Password == "" {
		error.Message = "パスワードがありません"
		respondWithError(w, http.StatusBadRequest, error)
		return
	}
}

func login(w http.ResponseWriter, r *http.Request) {
	fmt.Println("login invoked")
	w.Write([]byte("successfully called login"))
}

func ProtectedEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Println("protected endpoint invoked")
}

func TokenVerifyMiddleWare(next http.HandlerFunc) http.HandlerFunc {
	fmt.Println("Toeknverify middleware invoked")
	return nil
}

func helloWorld(uri, username, password string) (string, error) {
	fmt.Println("hello world")
	var (
		err      error
		driver   neo4j.Driver
		session  neo4j.Session
		result   neo4j.Result
		greeting interface{}
	)

	driver, err = neo4j.NewDriver(uri, neo4j.BasicAuth(username, password, ""))
	fmt.Println(driver)

	if err != nil {
		fmt.Println(driver)
		return "", err
	}
	defer driver.Close()

	session, err = driver.Session(neo4j.AccessModeWrite)
	if err != nil {
		return "", err
	}
	defer session.Close()

	greeting, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err = transaction.Run(
			"CREATE (a:Greeting) SET a.message = $message RETURN a.message + ', from node ' + id(a)",
			map[string]interface{}{"message": "hello, world"})
		fmt.Println(result)
		if err != nil {
			fmt.Println("error")
			return nil, err
		}

		if result.Next() {
			return result.Record().GetByIndex(0), nil
		}

		return nil, result.Err()
	})
	if err != nil {
		return "", err
	}

	return greeting.(string), nil
}