package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"strings"

	//"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	//"github.com/gorilla/mux"
	//"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"golang.org/x/crypto/bcrypt"
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {

	// ルーティング
	router := mux.NewRouter()

	router.HandleFunc("/signup", signup).Methods("POST")
	router.HandleFunc("/login", login).Methods("POST")
	router.HandleFunc("/protected", TokenVerifyMiddleWare(ProtectedEndpoint)).Methods("GET")

	log.Println("Listen on port 8000...")
	log.Fatal(http.ListenAndServe(":8000", router))
}

func respondWithError(w http.ResponseWriter, status int, error models.Error) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(error)
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(data)
}

func signup(w http.ResponseWriter, r *http.Request) {
	var user models.User
	var error models.Error
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

	// すでにメールアドレスが登録されていないか検証する
	hashedPassword, err := searchUserByEmail(user.Email)
	if err != nil {
		log.Fatal(err)
	}
	if hashedPassword != "" {
		error.Message = "そのメールアドレスはすでに使用されています"
		respondWithError(w, http.StatusBadRequest, error)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
	if err != nil {
		log.Fatal(err)
	}

	user.Password = string(hash)

	// neo4jノードに登録する
	result, err := createUser(user)
	if result == "" || err != nil {
		error.Message = "サーバーエラーです"
		respondWithError(w, http.StatusInternalServerError, error)
		return
	}

	user.Password = ""
	w.Header().Set("Content-Type", "appliaction/json")
	responseJSON(w, user)
}

func createUser(user models.User) (string, error) {
	var (
		err     error
		driver  neo4j.Driver
		session neo4j.Session
		result  neo4j.Result
		newUser interface{}
	)

	driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

	if err != nil {
		return "", err
	}
	defer driver.Close()

	session, err = driver.Session(neo4j.AccessModeWrite)
	if err != nil {
		return "", err
	}
	defer session.Close()

	newUser, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
		result, err = transaction.Run(
			"CREATE (u:User) SET u.email = $email, u.password = $password RETURN u.email + ', from node ' + id(u)",
			map[string]interface{}{"email": user.Email, "password": user.Password})
		if err != nil {
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

	return newUser.(string), nil
}

func GenerateToken(user models.User) (string, error) {
	var err error
	secret := os.Getenv("token_secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"iss":   "course",
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatal(err)
	}

	return tokenString, nil
}

func login(w http.ResponseWriter, r *http.Request) {
	var user models.User
	var jwt models.JWT
	var error models.Error

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
	fmt.Println(user.Email)

	// データベースからemailで検索する
	hashedPassword, err := searchUserByEmail(user.Email)
	if err != nil {
		log.Fatal(err)
	}
	bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
	if err != nil {
		error.Message = "パスワードが正しくありません"
		respondWithError(w, http.StatusUnauthorized, error)
		return
	}

	// トークン取得
	token, err := GenerateToken(user)
	if err != nil {
		log.Fatal(err)
	}
	w.WriteHeader(http.StatusOK)
	jwt.Token = token

	responseJSON(w, jwt)

}

func searchUserByEmail(email string) (string, error) {
	var (
		driver   neo4j.Driver
		session  neo4j.Session
		result   neo4j.Result
		err      error
		password string
	)

	if driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), "")); err != nil {
		return "", err // handle error
	}
	defer driver.Close()

	if session, err = driver.Session(neo4j.AccessModeWrite); err != nil {
		return "", err
	}
	defer session.Close()

	result, err = session.Run("MATCH (u:User {email: $email}) return id(u), u.email, u.password;", map[string]interface{}{
		"email": email,
	})

	if err != nil {
		return "", err // handle error
	}
	if result.Next() {
		password = result.Record().GetByIndex(2).(string)
		fmt.Printf("Matched user with Id = '%d' and Email = '%s' and Password = '%T'\n", result.Record().GetByIndex(0).(int64), result.Record().GetByIndex(1).(string), result.Record().GetByIndex(2).(string))
	}
	if err = result.Err(); err != nil {
		return "", err // handle error
	}
	return password, err
}

func ProtectedEndpoint(w http.ResponseWriter, r *http.Request) {
	fmt.Println("protected endpoint invoked")
}

func TokenVerifyMiddleWare(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var errorObject models.Error
		authHeader := r.Header.Get("Authorization")
		bearerToken := strings.Split(authHeader, " ")

		if len(bearerToken) == 2 {
			authToken := bearerToken[1]
			token, error := jwt.Parse(authToken, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("トークン系エラーです")
				}

				return []byte(os.Getenv("token_secret")), nil
			})

			if error != nil {
				errorObject.Message = error.Error()
				respondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}

			if token.Valid {
				next.ServeHTTP(w, r)
			} else {
				errorObject.Message = error.Error()
				respondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}
		} else {
			errorObject.Message = "トークンの形式が不正です"
			respondWithError(w, http.StatusUnauthorized, errorObject)
			return
		}
	})
}
