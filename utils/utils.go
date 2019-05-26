package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

func RespondWithError(w http.ResponseWriter, status int, error models.Error) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(error)
}

func ResponseJSON(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(data)
}

func CreateUser(user models.User) (string, error) {
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
			"CREATE (u:User) SET " +
				"u.email = $email, " +
				"u.username = $username, " +
				"u.password = $password, " +
				"u.created_at = $created_at, " +
				"u.updated_at = $updated_at " +
				"RETURN u.email + ', from node ' + id(u)",
			map[string]interface{}{
				"email": user.Email,
				"username": user.Username,
				"password": user.Password,
				"created_at": time.Now().Format("2006-01-02 15:04:05"),
				"updated_at": time.Now().Format("2006-01-02 15:04:05"),
			})
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

func SearchUserByEmail(email string) (*models.User, error) {
	var (
		driver   neo4j.Driver
		session  neo4j.Session
		result   neo4j.Result
		err      error
		user models.User
	)

	if driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), "")); err != nil {
		return nil, err // handle error
	}
	defer driver.Close()

	if session, err = driver.Session(neo4j.AccessModeWrite); err != nil {
		return nil, err
	}
	defer session.Close()

	result, err = session.Run("MATCH (u:User {email: $email}) return id(u), u.email, u.username, u.password;", map[string]interface{}{
		"email": email,
	})

	if err != nil {
		return nil, err // handle error
	}
	if result.Next() {
		user.ID = int(result.Record().GetByIndex(0).(int64))
		user.Email = result.Record().GetByIndex(1).(string)
		user.Username = result.Record().GetByIndex(2).(string)
		user.Password = result.Record().GetByIndex(3).(string)
		fmt.Printf("Matched user with Id = '%d' and Email = '%s' and Password = '%T'\n", result.Record().GetByIndex(0).(int64), result.Record().GetByIndex(1).(string), result.Record().GetByIndex(2).(string))
	} else {
		// 該当するユーザーが以内場合は、nilを返す
		return nil, err
	}
	if err = result.Err(); err != nil {
		return nil, err // handle error
	}
	return &user, err
}


func GenerateToken(user models.User) (string, error) {
	var err error
	secret := os.Getenv("token_secret")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": user.Email,
		"iss": os.Getenv("token_iss"),
	})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		log.Fatal(err)
	}

	return tokenString, nil
}