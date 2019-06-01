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

type SabotaController struct{}

func (c SabotaController) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error

		// 時系列順でsabotaを取得する
		var (
			err         error
			driver      neo4j.Driver
			session     neo4j.Session
			result      neo4j.Result
			countResult neo4j.Result
		)
		driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

		if err != nil {
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

		sabotaList, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var sabotaList []models.Sabota

			result, err = transaction.Run(
				"MATCH (n:Sabota)<-[e:POST]-(u:User)"+
					" RETURN ID(n), "+
					"n.shouldDone, "+
					"n.mistake, "+
					"n.time, "+
					"n.body, "+
					"n.created_at, "+
					"n.updated_at, "+
					"ID(u), "+
					"u.username "+
					"ORDER BY ID(n) DESC;",
				map[string]interface{}{})

			if err != nil {
				return nil, err
			}

			fmt.Println("sabota index invoked 2")

			for result.Next() {
				var sabota models.Sabota
				var user models.User

				var metooUserIds []int
				var loveUserIds []int
				var commentUserIds []int

				sabota.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				sabota.ShouldDone = result.Record().GetByIndex(1).(string)
				sabota.Mistake = result.Record().GetByIndex(2).(string)
				sabota.Time = int(result.Record().GetByIndex(3).(int64))
				sabota.Body = result.Record().GetByIndex(4).(string)
				sabota.CreatedAt = result.Record().GetByIndex(5).(string)
				sabota.UpdatedAt = result.Record().GetByIndex(6).(string)
				user.ID = int(result.Record().GetByIndex(7).(int64))
				user.Username = result.Record().GetByIndex(8).(string)

				sabota.PostUser = user

				// metooの数をcount, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:METOO]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					metooUserIds = append(metooUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				// loveの数, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:LOVE]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					loveUserIds = append(loveUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				// commentの数をカウント, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[:COMMENT]-(com:Comment)<-[:POST]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					commentUserIds = append(commentUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				sabota.LoveUserIds = loveUserIds
				sabota.MetooUserIds = metooUserIds
				sabota.CommentUserIds = commentUserIds

				sabotaList = append(sabotaList, sabota)
			}

			return sabotaList, result.Err()
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, sabotaList)
	}
}

func (c SabotaController) SearchSabotas() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonSabotaSearch models.SabotaSearch // post内容のsabotaの検索ワード
		var whereQuery string = " WHERE "        // サイファークエリのwhere句の部分
		var keyWordQuery string = ""
		var shouldDoneQuery string = ""
		var mistakeQuery string = ""
		var timeQuery string = ""

		json.NewDecoder(r.Body).Decode(&jsonSabotaSearch)

		// 検索ワードから、サイファークエリのWHERE句の部分を組み立てる
		if jsonSabotaSearch.KeyWord != "" { // キーワードが指定されている
			keyWordQuery = " (s.shouldDone =~ '.*" + jsonSabotaSearch.KeyWord + ".*' OR " +
				"s.mistake =~ '.*" + jsonSabotaSearch.KeyWord + ".*' OR " +
				"s.body =~ '.*" + jsonSabotaSearch.KeyWord + ".*' ) "
		}
		if jsonSabotaSearch.ShouldDone != "" {
			if keyWordQuery != "" {
				shouldDoneQuery += " AND "
			}
			shouldDoneQuery += " (s.shouldDone =~ '.*" + jsonSabotaSearch.ShouldDone + ".*')"
		}
		if jsonSabotaSearch.Mistake != "" {
			if keyWordQuery != "" || shouldDoneQuery != "" {
				mistakeQuery += " AND "
			}
			mistakeQuery += " (s.mistake =~ '.*" + jsonSabotaSearch.Mistake + ".*')"
		}
		if jsonSabotaSearch.Time != 0 {
			if keyWordQuery != "" || shouldDoneQuery != "" || mistakeQuery != "" {
				timeQuery += " AND "
			}
			timeQuery += " (s.time = " + strconv.Itoa(jsonSabotaSearch.Time) + ")"
		}
		whereQuery += keyWordQuery + shouldDoneQuery + mistakeQuery + timeQuery

		fmt.Println("MATCH (s:Sabota)<-[e:POST]-(u:User) " + whereQuery +
			" RETURN ID(s), " +
			"s.shouldDone, " +
			"s.mistake, " +
			"s.time, " +
			"s.body, " +
			"s.created_at, " +
			"s.updated_at, " +
			"ID(u), " +
			"u.username " +
			"ORDER BY ID(s) DESC;")

		// 時系列順でsabotaを取得する
		var (
			err         error
			driver      neo4j.Driver
			session     neo4j.Session
			result      neo4j.Result
			countResult neo4j.Result
		)
		driver, err = neo4j.NewDriver(os.Getenv("db_url"), neo4j.BasicAuth(os.Getenv("db_user"), os.Getenv("db_pass"), ""))

		if err != nil {
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

		sabotaList, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var sabotaList []models.Sabota

			result, err = transaction.Run(
				"MATCH (s:Sabota)<-[e:POST]-(u:User) "+whereQuery+
					" RETURN ID(s), "+
					"s.shouldDone, "+
					"s.mistake, "+
					"s.time, "+
					"s.body, "+
					"s.created_at, "+
					"s.updated_at, "+
					"ID(u), "+
					"u.username "+
					"ORDER BY ID(s) DESC;",
				map[string]interface{}{})

			if err != nil {
				return nil, err
			}

			for result.Next() {
				var sabota models.Sabota
				var user models.User

				var metooUserIds []int
				var loveUserIds []int
				var commentUserIds []int

				sabota.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				sabota.ShouldDone = result.Record().GetByIndex(1).(string)
				sabota.Mistake = result.Record().GetByIndex(2).(string)
				sabota.Time = int(result.Record().GetByIndex(3).(int64))
				sabota.Body = result.Record().GetByIndex(4).(string)
				sabota.CreatedAt = result.Record().GetByIndex(5).(string)
				sabota.UpdatedAt = result.Record().GetByIndex(6).(string)
				user.ID = int(result.Record().GetByIndex(7).(int64))
				user.Username = result.Record().GetByIndex(8).(string)

				sabota.PostUser = user

				// metooの数をcount, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:METOO]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					metooUserIds = append(metooUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				// loveの数, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:LOVE]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					loveUserIds = append(loveUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				// commentの数をカウント, userIdのリストを作成
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[:COMMENT]-(com:Comment)<-[:POST]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					commentUserIds = append(commentUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				sabota.LoveUserIds = loveUserIds
				sabota.MetooUserIds = metooUserIds
				sabota.CommentUserIds = commentUserIds

				sabotaList = append(sabotaList, sabota)
			}

			return sabotaList, result.Err()
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, sabotaList)
	}
}

func (c SabotaController) Show() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])

		// 時系列順でsabotaを取得する
		var (
			err         error
			driver      neo4j.Driver
			session     neo4j.Session
			result      neo4j.Result
			countResult neo4j.Result
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

		sabota, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var sabota models.Sabota

			result, err = transaction.Run(
				"MATCH (n:Sabota)<-[e:POST]-(u:User) WHERE ID(n) = $sabotaId RETURN "+
					"ID(n), "+
					"n.shouldDone, "+
					"n.mistake, "+
					"n.time, "+
					"n.body, "+
					"n.created_at, "+
					"n.updated_at, "+
					"ID(u), "+
					"u.username ",
				map[string]interface{}{"sabotaId": sabotaId})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				var user models.User
				var metooUserIds []int
				var loveUserIds []int
				var commentUserIds []int

				sabota.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				sabota.ShouldDone = result.Record().GetByIndex(1).(string)
				sabota.Mistake = result.Record().GetByIndex(2).(string)
				sabota.Time = int(result.Record().GetByIndex(3).(int64))
				sabota.Body = result.Record().GetByIndex(4).(string)
				sabota.CreatedAt = result.Record().GetByIndex(5).(string)
				sabota.UpdatedAt = result.Record().GetByIndex(6).(string)

				user.ID = int(result.Record().GetByIndex(7).(int64))
				user.Username = result.Record().GetByIndex(8).(string)

				sabota.PostUser = user

				// metooの数をcount
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:METOO]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				if countResult.Next() {
					metooUserIds = append(metooUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				// loveの数
				countResult, err = transaction.Run(
					"MATCH (n:Sabota)<-[e:LOVE]-(u:User) "+
						"WHERE ID(n) = $sabotaId "+
						"RETURN ID(u) ",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				if countResult.Next() {
					loveUserIds = append(loveUserIds, int(countResult.Record().GetByIndex(0).(int64)))
				}

				var commentList []models.Comment

				// commentの一覧を取得する
				countResult, err = transaction.Run(
					"MATCH (sa:Sabota)<-[:COMMENT]-(com:Comment)<-[:POST]-(u:User) "+
						"WHERE ID(sa) = $sabotaId "+
						"RETURN count(com), ID(com), com.body, com.created_at, com.updated_at, ID(u), u.username "+
						"ORDER BY com.created_at DESC;",
					map[string]interface{}{"sabotaId": sabota.ID})
				if err != nil {
					return nil, err
				}
				for countResult.Next() {
					var comment models.Comment
					var postUser models.User

					comment.ID = int(countResult.Record().GetByIndex(1).(int64))
					comment.Body = countResult.Record().GetByIndex(2).(string)
					comment.CreatedAt = countResult.Record().GetByIndex(3).(string)

					postUser.ID = int(countResult.Record().GetByIndex(5).(int64))
					postUser.Username = countResult.Record().GetByIndex(6).(string)

					commentUserIds = append(commentUserIds, int(countResult.Record().GetByIndex(5).(int64)))

					comment.PostUser = postUser
					commentList = append(commentList, comment)
				}

				sabota.LoveUserIds = loveUserIds
				sabota.MetooUserIds = metooUserIds
				sabota.CommentUserIds = commentUserIds

				sabota.Comments = commentList

				return sabota, result.Err()
			} else {
				return nil, nil
			}
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, sabota)
	}
}

func (c SabotaController) Store() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonSabota models.Sabota          // post内容のsabota
		userId := r.Context().Value("userId") // ログインユーザーID

		// リクエスト内容をデコードして作成するsabotaデータを取り出す
		json.NewDecoder(r.Body).Decode(&jsonSabota)
		// 検証
		if jsonSabota.ShouldDone == "" {
			validationError.Message = "やるべきだったことが抜けています"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}
		if jsonSabota.Mistake == "" {
			validationError.Message = "やっちゃったことが抜けています"
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

		// sabota新規作成
		newSabotaId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var newSabotaId int

			result, err = transaction.Run(
				"CREATE (s:Sabota) SET "+
					"s.shouldDone = $shouldDone, "+
					"s.mistake = $mistake, "+
					"s.time = $time, "+
					"s.body = $body, "+
					"s.created_at = $created_at, "+
					"s.updated_at = $updated_at "+
					"RETURN ID(s);",
				map[string]interface{}{
					"shouldDone": jsonSabota.ShouldDone,
					"mistake":    jsonSabota.Mistake,
					"time":       jsonSabota.Time,
					"body":       jsonSabota.Body,
					"created_at": time.Now().Format("2006-01-02 15:04:05"),
					"updated_at": time.Now().Format("2006-01-02 15:04:05"),
				})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				newSabotaId = int(result.Record().GetByIndex(0).(int64))
			}

			result, err = transaction.Run(
				"MATCH (u:User), (sa:Sabota) "+
					"WHERE ID(u) = $userId AND ID(sa) = $sabotaId "+
					"CREATE (u)-[e:POST]->(sa) RETURN e;",
				map[string]interface{}{"userId": userId, "sabotaId": newSabotaId})

			return newSabotaId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		// Mistake、ShouldDoneノードとの間に、エッジをはる
		// 該当する名前のShouldDoneノードの存在確認
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var count int64
			result, err = transaction.Run(
				"MATCH (n:ShouldDone) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.ShouldDone})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
			}

			// その名前のShouldDoneノードが存在しない場合
			if count == 0 {
				// ShouldDoneノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:ShouldDone) SET "+
						"n.name = $name, "+
						"n.created_at = $created_at, "+
						"n.updated_at = $updated_at "+
						"RETURN n",
					map[string]interface{}{
						"name":       jsonSabota.ShouldDone,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONTエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (sd:ShouldDone) "+
					"WHERE ID(sa) = $sabotaId AND sd.name = $shouldDoneName "+
					"CREATE (sa)-[e:DONT]->(sd)"+
					"RETURN e",
				map[string]interface{}{
					"shouldDoneName": jsonSabota.ShouldDone,
					"sabotaId":       newSabotaId,
				})

			if err != nil {
				return nil, err
			}

			result, err = transaction.Run(
				"MATCH (n:Mistake) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.Mistake})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
			}

			// その名前のMistakeノードが存在しない場合
			if count == 0 {
				// Mistakeノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:Mistake) SET "+
						"n.name = $name, "+
						"n.created_at = $created_at, "+
						"n.updated_at = $updated_at "+
						"RETURN n",
					map[string]interface{}{
						"name":       jsonSabota.Mistake,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONEエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (m:Mistake) "+
					"WHERE ID(sa) = $sabotaId AND m.name = $mistakeName "+
					"CREATE (sa)-[e:DONE]->(m)"+
					"RETURN e",
				map[string]interface{}{
					"mistakeName": jsonSabota.Mistake,
					"sabotaId":    newSabotaId,
				})
			return nil, nil
		})

		jsonSabota.ID = newSabotaId.(int)
		jsonSabota.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
		jsonSabota.UpdatedAt = time.Now().Format("2006-01-02 15:04:05")
		utils.ResponseJSON(w, jsonSabota)
		return
	}
}

func (c SabotaController) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonSabota models.Sabota          // post内容のsabota
		userId := r.Context().Value("userId") // ログインユーザーID

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])

		// リクエスト内容をデコードして作成するsabotaデータを取り出す
		json.NewDecoder(r.Body).Decode(&jsonSabota)
		// 検証
		if jsonSabota.ShouldDone == "" {
			validationError.Message = "やるべきだったことが抜けています"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}
		if jsonSabota.Mistake == "" {
			validationError.Message = "やっちゃったことが抜けています"
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

		// そのsabotaの投稿者とtoken経由のuserIdが一致するか確認
		postUserId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var postUserId int

			result, err = transaction.Run(
				"MATCH (u:User)-[:POST]->(s:Sabota) WHERE ID(s) = $sabotaId RETURN ID(u);",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				postUserId = int(result.Record().GetByIndex(0).(int64))
			}

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

		// アルゴリズム的に、先にエッジを消しておいた方が楽なのでそうするか
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			result, err = transaction.Run(
				"MATCH (s:Sabota)-[e:DONE]->() WHERE ID(s) = $sabotaId DELETE e",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			result, err = transaction.Run(
				"MATCH (s:Sabota)-[e:DONT]->() WHERE ID(s) = $sabotaId DELETE e",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

			return nil, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}

		// 該当するsabotaをupdateする
		updatedSabotaId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var newSabotaId int64

			result, err = transaction.Run(
				"MATCH (s:Sabota)"+
					"WHERE ID(s) = $sabotaId SET "+
					"s.shouldDone = $shouldDone, "+
					"s.mistake = $mistake, "+
					"s.time = $time, "+
					"s.body = $body, "+
					"s.updated_at = $updated_at "+
					"RETURN ID(s);",
				map[string]interface{}{
					"sabotaId":   sabotaId,
					"shouldDone": jsonSabota.ShouldDone,
					"mistake":    jsonSabota.Mistake,
					"time":       jsonSabota.Time,
					"body":       jsonSabota.Body,
					"updated_at": time.Now().Format("2006-01-02 15:04:05"),
				})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				newSabotaId = result.Record().GetByIndex(0).(int64)
			}

			return newSabotaId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}

		// Mistake、ShouldDoneノードとの間に、エッジをはる
		// 該当する名前のShouldDoneノードの存在確認
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var count int64
			result, err = transaction.Run(
				"MATCH (n:ShouldDone) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.ShouldDone})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
			}

			// その名前のShouldDoneノードが存在しない場合
			if count == 0 {
				// ShouldDoneノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:ShouldDone) SET "+
						"n.name = $name, "+
						"n.created_at = $created_at, "+
						"n.updated_at = $updated_at "+
						"RETURN n",
					map[string]interface{}{
						"name":       jsonSabota.ShouldDone,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONTエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (sd:ShouldDone) "+
					"WHERE ID(sa) = $sabotaId AND sd.name = $shouldDoneName "+
					"CREATE (sa)-[e:DONT]->(sd)"+
					"RETURN e",
				map[string]interface{}{
					"shouldDoneName": jsonSabota.ShouldDone,
					"sabotaId":       updatedSabotaId,
				})

			if err != nil {
				return nil, err
			}

			result, err = transaction.Run(
				"MATCH (n:Mistake) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.Mistake})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
			}

			// その名前のMistakeノードが存在しない場合
			if count == 0 {
				// Mistakeノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:Mistake) SET "+
						"n.name = $name, "+
						"n.created_at = $created_at, "+
						"n.updated_at = $updated_at "+
						"RETURN n",
					map[string]interface{}{
						"name":       jsonSabota.Mistake,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONEエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (m:Mistake) "+
					"WHERE ID(sa) = $sabotaId AND m.name = $mistakeName "+
					"CREATE (sa)-[e:DONE]->(m)"+
					"RETURN e",
				map[string]interface{}{
					"mistakeName": jsonSabota.Mistake,
					"sabotaId":    updatedSabotaId,
				})
			return nil, nil
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		jsonSabota.ID = sabotaId
		jsonSabota.UpdatedAt = time.Now().Format("2006-01-02 15:04:05") // うそ
		utils.ResponseJSON(w, jsonSabota)
	}
}

// todo 論理削除は後回しで良いでしょう
func (c SabotaController) Destroy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		userId := r.Context().Value("userId") // ログインユーザーID

		// クエリパラメータから、sabotaIDを取得する
		params := mux.Vars(r)
		sabotaId, _ := strconv.Atoi(params["sabotaId"])

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

		// そのsabotaの投稿者とtoken経由のuserIdが一致するか確認
		postUserId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var postUserId int

			result, err = transaction.Run(
				"MATCH (u:User)-[:POST]->(s:Sabota) WHERE ID(s) = $sabotaId RETURN ID(u);",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				postUserId = int(result.Record().GetByIndex(0).(int64))
			}

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

		// 対象のsabotaを消す
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			result, err = transaction.Run(
				"MATCH (s:Sabota) WHERE ID(s) = $sabotaId DETACH DELETE s",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			result, err = transaction.Run(
				"MATCH (s:Sabota)-[e:DONT]->() WHERE ID(s) = $sabotaId DELETE e",
				map[string]interface{}{
					"sabotaId": sabotaId,
				})

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
