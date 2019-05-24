package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	//"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"

	//"github.com/gorilla/mux"
	//"github.com/joho/godotenv"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"golang.org/x/crypto/bcrypt"
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
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	//helloWorld(os.Getenv("db_url"), os.Getenv("db_user"), os.Getenv("db_pass"))
	//result, err := searchUserByEmail("www@www.com")

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

func responseJSON(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(data)
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

func createUser(user User) (string, error) {
	var (
		err			error
		driver   neo4j.Driver
		session  neo4j.Session
		result   neo4j.Result
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

func GenerateToken(user User) (string, error) {
	var err error
	secret := "secret"

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"iss": "course",
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatal(err)
	}

	return tokenString, nil
}

func login(w http.ResponseWriter, r *http.Request) {
	var user User
	var jwt JWT
	var error Error

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
		driver neo4j.Driver
		session neo4j.Session
		result neo4j.Result
		err error
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