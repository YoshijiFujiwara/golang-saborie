package controllers

import (
	"encoding/json"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"portfolio/saborie/utils"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type CommentController struct {}

func (c CommentController) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クエリパラメータから、sabotaIDを取得する
		//params := mux.Vars(r)
		//sabotaId, _ := strconv.Atoi(params["sabotaId"])


	}
}

func (c CommentController) Store() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonComment models.Comment // post内容のsabota

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])
		userId := r.Context().Value("userId") // ログインユーザーID

		// リクエスト内容をデコードして作成するcommentデータを取り出す
		json.NewDecoder(r.Body).Decode(&jsonComment)
		// 検証
		if jsonComment.Body == "" {
			validationError.Message = "コメントが空です"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		var (
			err     error
			driver  neo4j.Driver
			session neo4j.Session
			result  neo4j.Result
		)
		driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

		if err != nil {
			return
		}
		defer driver.Close()

		session, err = driver.Session(neo4j.AccessModeWrite)
		if err != nil {
			return
		}
		defer session.Close()

		// コメント新規作成
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var newCommentId int64

			result, err = transaction.Run(
				"CREATE (c:Comment) SET " +
					"c.body = $body, "+
					"c.created_at = $created_at, " +
					"c.updated_at = $updated_at " +
					"RETURN ID(c);",
				map[string]interface{}{
					"body": jsonComment.Body,
					"created_at": time.Now().Format("2006-01-02 15:04:05"),
					"updated_at": time.Now().Format("2006-01-02 15:04:05"),
				})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				newCommentId = result.Record().GetByIndex(0).(int64)
			}


			// sabotaとの関連付け
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (com:Comment) " +
					"WHERE ID(sa) = $sabotaId AND ID(com) = $commentId " +
					"CREATE (sa)<-[e:COMMENT]-(com) RETURN e;",
				map[string]interface{}{"sabotaId": sabotaId, "commentId": newCommentId})
			if err != nil {
				return nil, err
			}

			// 投稿者との関連づけ
			result, err = transaction.Run(
				"MATCH (u:User), (com:Comment) " +
					"WHERE ID(u) = $userId AND ID(com) = $commentId " +
					"CREATE (u)-[e:POST]->(com) RETURN e;",
				map[string]interface{}{"userId": userId, "commentId": newCommentId})
			if err != nil{
				return nil, err
			}


			return newCommentId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}

		return
	}
}

func (c CommentController) Show() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クエリパラメータから、sabotaIDを取得する
		//params := mux.Vars(r)
		//sabotaId, _ := strconv.Atoi(params["sabotaId"])


	}
}

func (c CommentController) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クエリパラメータから、sabotaIDを取得する
		//params := mux.Vars(r)
		//sabotaId, _ := strconv.Atoi(params["sabotaId"])
		//userId := r.Context().Value("userId") // ログインユーザーID


	}
}

func (c CommentController) Destroy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クエリパラメータから、sabotaIDを取得する
		//params := mux.Vars(r)
		//sabotaId, _ := strconv.Atoi(params["sabotaId"])
		//userId := r.Context().Value("userId") // ログインユーザーID


	}
}