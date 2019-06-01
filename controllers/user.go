package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"portfolio/saborie/utils"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/dgrijalva/jwt-go"
	"github.com/neo4j/neo4j-go-driver/neo4j"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/context"
)

type UserController struct{}

func (c UserController) Register() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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

		// すでにメールアドレスが登録されていないか検証する
		dbUser, err := utils.SearchUser(user.Email, "email")
		if err != nil {
			log.Fatal(err)
		}
		if dbUser != nil {
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

func (c UserController) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		if user.Password == "" {
			error.Message = "パスワードがありません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		// データベースからemailで検索する
		dbUser, err := utils.SearchUser(user.Email, "email")
		if err != nil {
			log.Fatal(err)
			return
		}
		fmt.Println(dbUser)
		if dbUser == nil {
			error.Message = "そのメールアドレスは登録されていません"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(dbUser.Password), []byte(user.Password))
		if err != nil {
			error.Message = "パスワードが正しくありません"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}

		// トークン取得
		token, err := utils.GenerateToken(user)
		if err != nil {
			error.Message = "認証エラー"
			utils.RespondWithError(w, http.StatusUnauthorized, error)
			return
		}
		w.WriteHeader(http.StatusOK)
		jwt.Token = token
		dbUser.Jwt = jwt

		utils.ResponseJSON(w, dbUser)
		return
	}
}

func (c UserController) Me() http.HandlerFunc {
	fmt.Println("me invoked")
	return func(w http.ResponseWriter, r *http.Request) {
		var error models.Error

		userId := r.Context().Value("userId") // ログインユーザーID
		fmt.Println("hogehoge")
		fmt.Println(userId)
		dbUser, err := utils.SearchUser(strconv.Itoa(userId.(int)), "id")
		if err != nil {
			log.Fatal(err)
		}
		if dbUser == nil {
			error.Message = "ユーザーを取得できません"
			utils.RespondWithError(w, http.StatusBadRequest, error)
			return
		}

		utils.ResponseJSON(w, dbUser)
		return
	}
}

func (c UserController) TokenVerifyMiddleware(next http.HandlerFunc) http.HandlerFunc {
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

			// todo トークン有効期限切れチェック
			if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {

				dbUser, err := utils.SearchUser(claims["email"].(string), "email")
				if err != nil {
					errorObject.Message = err.Error()
					utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
					return
				}
				if dbUser == nil {
					errorObject.Message = "ユーザーが見つかりません"
					utils.RespondWithError(w, http.StatusUnauthorized, errorObject)
					return
				}
				spew.Dump(dbUser)
				ctx := context.WithValue(r.Context(), "userId", dbUser.ID) // ログインユーザーIDを渡す
				r = r.WithContext(ctx)
				next.ServeHTTP(w, r) //ハンドラーへ返却
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

func (c UserController) GetMistakeSummary() http.HandlerFunc {
	fmt.Println("mistake summary invoked")
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error

		userId := r.Context().Value("userId") // ログインユーザーID
		dbUser, err := utils.SearchUser(strconv.Itoa(userId.(int)), "id")
		if err != nil {
			log.Fatal(err)
		}
		if dbUser == nil {
			validationError.Message = "ユーザーを取得できません"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		// ユーザーのミステイクのサマリーを取得する
		var (
			//err         error
			driver      neo4j.Driver
			session     neo4j.Session
			result      neo4j.Result
		)
		driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		defer driver.Close()

		session, err = driver.Session(neo4j.AccessModeWrite)
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		defer session.Close()

		mistakeSummary, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			summary := make(map[string](map[string]int))

			result, err = transaction.Run(
				"MATCH (m:Mistake)<-[:DONE]-(s:Sabota)<-[e:POST]-(u:User) WHERE ID(u) = $userId " +
				"RETURN m.name, sum(s.time) as sumTime, count(s.time)",
				map[string]interface{}{"userId": userId})

			if err != nil {
				return nil, err
			}

			for result.Next() {
				inner := make(map[string]int)
				inner["sumTime"] = int(result.Record().GetByIndex(1).(int64))
				inner["count"] = int(result.Record().GetByIndex(2).(int64))

				summary[result.Record().GetByIndex(0).(string)] = inner
			}

			return summary, err
		})

		utils.ResponseJSON(w, mistakeSummary)
		return
	}
}

func (c UserController) GetShouldDoneSummary() http.HandlerFunc {
	fmt.Println("mistake summary invoked")
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error

		userId := r.Context().Value("userId") // ログインユーザーID
		dbUser, err := utils.SearchUser(strconv.Itoa(userId.(int)), "id")
		if err != nil {
			log.Fatal(err)
		}
		if dbUser == nil {
			validationError.Message = "ユーザーを取得できません"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		// ユーザーのミステイクのサマリーを取得する
		var (
			//err         error
			driver      neo4j.Driver
			session     neo4j.Session
			result      neo4j.Result
		)
		driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		defer driver.Close()

		session, err = driver.Session(neo4j.AccessModeWrite)
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		defer session.Close()

		mistakeSummary, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			summary := make(map[string](map[string]int))

			result, err = transaction.Run(
				"MATCH (sd:ShouldDone)<-[:DONT]-(s:Sabota)<-[e:POST]-(u:User) WHERE ID(u) = $userId " +
					"RETURN sd.name, sum(s.time), count(s.time)",
				map[string]interface{}{"userId": userId})

			if err != nil {
				return nil, err
			}

			for result.Next() {
				inner := make(map[string]int)
				inner["sumTime"] = int(result.Record().GetByIndex(1).(int64))
				inner["count"] = int(result.Record().GetByIndex(2).(int64))

				summary[result.Record().GetByIndex(0).(string)] = inner
			}

			return summary, err
		})

		utils.ResponseJSON(w, mistakeSummary)
		return
	}
}