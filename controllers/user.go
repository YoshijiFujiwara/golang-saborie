package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"portfolio/saborie/utils"
	"strings"

	"github.com/dgrijalva/jwt-go"
	"golang.org/x/crypto/bcrypt"
)

type Controller struct {}

func (c Controller) Signup() http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		var user models.User
		var error models.Error
		// リクエスト内容のデコード
		json.NewDecoder(r.Body).Decode(&user)

		// 検証
		if user.Email == "" {
			error.Message = "メールアドレスがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}
		if user.Password == "" {
			error.Message = "パスワードがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// すでにメールアドレスが登録されていないか検証する
		hashedPassword, err := utils.SearchUserByEmail(user.Email)
		if err != nil {
			log.Fatal(err)
		}
		if hashedPassword != "" {
			error.Message = "そのメールアドレスはすでに使用されています"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		hash, err := bcrypt.GenerateFromPassword([]byte(user.Password), 10)
		if err != nil {
			log.Fatal(err)
		}

		user.Password = string(hash)

		// neo4jノードに登録する
		result, err := utils.CreateUser(user)
		if result == "" || err != nil {
			error.Message = "サーバーエラーです"
			utils.RespondWithError(w, http.StatusInternalServerError, error)
			return
		}

		user.Password = ""
		w.Header().Set("Content-Type", "appliaction/json")
		utils.ResponseJSON(w, user)
	}
}

func (c Controller) Login() http.HandlerFunc {
	return func (w http.ResponseWriter, r *http.Request) {
		var user models.User
		var jwt models.JWT
		var error models.Error

		json.NewDecoder(r.Body).Decode(&user)

		// 検証
		if user.Email == "" {
			error.Message = "メールアドレスがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}
		if user.Username == "" {
			error.Message = "ユーザーネームがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}
		if user.Password == "" {
			error.Message = "パスワードがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}
		fmt.Println(user.Email)

		// データベースからemailで検索する
		hashedPassword, err := utils.SearchUserByEmail(user.Email)
		if err != nil {
			log.Fatal(err)
		}
		bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(user.Password))
		if err != nil {
			error.Message = "パスワードが正しくありません"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// トークン取得
		token, err := utils.GenerateToken(user)
		if err != nil {
			log.Fatal(err)
		}
		w.WriteHeader(http.StatusOK)
		jwt.Token = token

		utils.ResponseJSON(w, jwt)
	}
}

func (c Controller) TokenVerifyMiddleware(next http.HandlerFunc) http.HandlerFunc {
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
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}

			if token.Valid {
				next.ServeHTTP(w, r)
			} else {
				errorObject.Message = error.Error()
				utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
				return
			}
		} else {
			errorObject.Message = "トークンの形式が不正です"
			utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
			return
		}
	})
}