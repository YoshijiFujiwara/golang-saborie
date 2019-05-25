package controllers

import (
	"encoding/json"
	"fmt"
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
		var validationError models.Error

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])

		// 時系列順でコメントを取得する
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

		commentList, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var commentList []models.Comment

			result, err = transaction.Run(
				"MATCH (s:Sabota)<-[:COMMENT]-(com:Comment) WHERE ID(s) = $sabotaId RETURN ID(com), com.body, com.created_at, com.updated_at ORDER BY ID(s) DESC;",
				map[string]interface{}{"sabotaId": sabotaId})

			if err != nil {
				return nil, err
			}


			for result.Next() {
				var comment models.Comment
				comment.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				comment.Body = result.Record().GetByIndex(1).(string)
				comment.CreatedAt = result.Record().GetByIndex(2).(string)
				comment.UpdatedAt = result.Record().GetByIndex(3).(string)

				commentList = append(commentList, comment)
			}

			return commentList, result.Err()
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, commentList)
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
		newCommentId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var newCommentId int

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
				newCommentId = int(result.Record().GetByIndex(0).(int64))
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

		jsonComment.ID = newCommentId.(int)
		utils.ResponseJSON(w, jsonComment)
	}
}

func (c CommentController) Show() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])
		commentId, _ := strconv.Atoi(params["commentId"])

		// 時系列順でコメントを取得する
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

		comment, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var comment models.Comment

			result, err = transaction.Run(
				"MATCH (s:Sabota)<-[:COMMENT]-(com:Comment) " +
					"WHERE ID(s) = $sabotaId AND ID(com) = $commentId " +
					"RETURN ID(com), " +
					"com.body, " +
					"com.created_at, " +
					"com.updated_at " +
					"ORDER BY ID(s) DESC;",
				map[string]interface{}{
					"sabotaId": sabotaId,
					"commentId": commentId,
				})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				comment.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				comment.Body = result.Record().GetByIndex(1).(string)
				comment.CreatedAt = result.Record().GetByIndex(2).(string)
				comment.UpdatedAt = result.Record().GetByIndex(3).(string)
			}

			return comment, result.Err()
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, comment)
	}
}

func (c CommentController) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("update invoked")
		var validationError models.Error
		var jsonComment models.Sabota // post内容のcomment
		userId := r.Context().Value("userId") // ログインユーザーID

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		commentId, _ := strconv.Atoi(params["commentId"])

		// リクエスト内容をデコードして作成するsabotaデータを取り出す
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


		// そのcommentの投稿者とtoken経由のuserIdが一致するか確認
		postUserId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var postUserId int

			result, err = transaction.Run(
				"MATCH (u:User)-[:POST]->(c:Comment) WHERE ID(c) = $commentId RETURN ID(u);",
				map[string]interface{}{
					"commentId": commentId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				postUserId = int(result.Record().GetByIndex(0).(int64))
			}
			fmt.Println(postUserId)


			return postUserId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		if postUserId != userId {
			validationError.Message = "不正なリクエストです"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		// 該当するcommentをupdateする
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var updatedCommentId int64

			result, err = transaction.Run(
				"MATCH (c:Comment)" +
					"WHERE ID(c) = $commentId SET " +
					"c.body = $body, "+
					"c.updated_at = $updated_at " +
					"RETURN ID(c);",
				map[string]interface{}{
					"commentId": commentId,
					"body": jsonComment.Body,
					"updated_at": time.Now().Format("2006-01-02 15:04:05"),
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				updatedCommentId = result.Record().GetByIndex(0).(int64)
			}

			return updatedCommentId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		jsonComment.ID = commentId
		jsonComment.UpdatedAt = time.Now().Format("2006-01-02 15:04:05") // うそ
		utils.ResponseJSON(w, jsonComment)
	}
}

func (c CommentController) Destroy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// クエリパラメータから、sabotaIDを取得する
		var validationError models.Error
		userId := r.Context().Value("userId") // ログインユーザーID

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		commentId, _ := strconv.Atoi(params["commentId"])

		var (
			err     error
			driver  neo4j.Driver
			session neo4j.Session
			result  neo4j.Result
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


		// そのcommentの投稿者とtoken経由のuserIdが一致するか確認
		postUserId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var postUserId int

			result, err = transaction.Run(
				"MATCH (u:User)-[:POST]->(c:Comment) WHERE ID(c) = $commentId RETURN ID(u);",
				map[string]interface{}{
					"commentId": commentId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				postUserId = int(result.Record().GetByIndex(0).(int64))
			}
			fmt.Println(postUserId)


			return postUserId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		if postUserId != userId {
			validationError.Message = "不正なリクエストです"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		// 対象のcommentを消す
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			result, err = transaction.Run(
				"MATCH (c:Comment) WHERE ID(c) = $commentId DETACH DELETE c",
				map[string]interface{}{
					"commentId": commentId,
				})

			if err != nil {
				return nil, err
			}

			return nil, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		return
	}
}