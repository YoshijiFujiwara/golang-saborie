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

	"github.com/davecgh/go-spew/spew"
	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type SabotaController struct {}

func (c SabotaController) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 時系列順でsabotaを取得する
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

		sabotaList, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var sabotaList []models.Sabota

			result, err = transaction.Run(
				"MATCH (n:Sabota) RETURN ID(n), n.shouldDone, n.mistake, n.time, n.body ORDER BY ID(n) DESC;",
				map[string]interface{}{})

			if err != nil {
				return nil, err
			}

			fmt.Println("hogehoge")

			for result.Next() {
				var sabota models.Sabota
				sabota.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				sabota.ShouldDone = result.Record().GetByIndex(1).(string)
				sabota.Mistake = result.Record().GetByIndex(2).(string)
				sabota.Time = result.Record().GetByIndex(3).(string)
				sabota.Body = result.Record().GetByIndex(4).(string)

				sabotaList = append(sabotaList, sabota)
			}

			return sabotaList, result.Err()
		})

		if err != nil {
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

		sabota, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var sabota models.Sabota

			result, err = transaction.Run(
				"MATCH (n:Sabota) WHERE ID(n) = $sabotaId RETURN ID(n), n.shouldDone, n.mistake, n.time, n.body;",
				map[string]interface{}{"sabotaId": sabotaId})

			if err != nil {
				return nil, err
			}

			fmt.Println("hogehoge")

			if result.Next() {
				sabota.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				sabota.ShouldDone = result.Record().GetByIndex(1).(string)
				sabota.Mistake = result.Record().GetByIndex(2).(string)
				sabota.Time = result.Record().GetByIndex(3).(string)
				sabota.Body = result.Record().GetByIndex(4).(string)

				return sabota, result.Err()
			} else {
				return nil, nil
			}
		})

		if err != nil {
			return
		}
		// sabotaリストをjsonで返す
		utils.ResponseJSON(w, sabota)
	}
}

func (c SabotaController) Store() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonSabota models.Sabota // post内容のsabota
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
		if jsonSabota.Time == "" {
			validationError.Message = "時間がありません"
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
			var newSabotaId int64

			result, err = transaction.Run(
				"CREATE (s:Sabota) SET " +
					"s.shouldDone = $shouldDone, "+
					"s.mistake = $mistake, "+
					"s.time = $time, "+
					"s.body = $body, "+
					"s.created_at = $created_at, " +
					"s.updated_at = $updated_at " +
					"RETURN ID(s);",
				map[string]interface{}{
					"shouldDone": jsonSabota.ShouldDone,
					"mistake": jsonSabota.Mistake,
					"time": jsonSabota.Time,
					"body": jsonSabota.Body,
					"created_at": time.Now().Format("2006-01-02 15:04:05"),
					"updated_at": time.Now().Format("2006-01-02 15:04:05"),
				})

			if err != nil {
				return nil, err
			}

			if result.Next() {
				newSabotaId = result.Record().GetByIndex(0).(int64)
			}

			result, err = transaction.Run(
				"MATCH (u:User), (sa:Sabota) " +
						"WHERE ID(u) = $userId AND ID(sa) = $sabotaId " +
						"CREATE (u)-[e:POST]->(sa) RETURN e;",
				map[string]interface{}{"userId": userId, "sabotaId": newSabotaId})

			return newSabotaId, result.Err()
		})
		if err != nil {
			return
		}
		fmt.Println("新規作成")
		spew.Dump(newSabotaId)

		fmt.Println(jsonSabota.ShouldDone)

		// Mistake、ShouldDoneノードとの間に、エッジをはる
		// 該当する名前のShouldDoneノードの存在確認
		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var count int64
			result, err = transaction.Run(
				"MATCH (n:ShouldDone) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.ShouldDone })

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
				fmt.Println(count)
			}

			// その名前のShouldDoneノードが存在しない場合
			if count == 0 {
				// ShouldDoneノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:ShouldDone) SET " +
						"n.name = $name, " +
						"n.created_at = $created_at, " +
						"n.updated_at = $updated_at " +
						"RETURN n",
					map[string]interface{}{
						"name": jsonSabota.ShouldDone,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONTエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (sd:ShouldDone) " +
					"WHERE ID(sa) = $sabotaId AND sd.name = $shouldDoneName " +
					"CREATE (sa)-[e:DONT]->(sd)" +
					"RETURN e",
				map[string]interface{}{
					"shouldDoneName": jsonSabota.ShouldDone,
					"sabotaId": newSabotaId,
				})

			if err != nil {
				return nil, err
			}

			result, err = transaction.Run(
				"MATCH (n:Mistake) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.Mistake })

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
				fmt.Println(count)
			}

			// その名前のMistakeノードが存在しない場合
			if count == 0 {
				// Mistakeノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:Mistake) SET " +
						"n.name = $name, " +
						"n.created_at = $created_at, " +
						"n.updated_at = $updated_at " +
						"RETURN n",
					map[string]interface{}{
						"name": jsonSabota.Mistake,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONEエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (m:Mistake) " +
					"WHERE ID(sa) = $sabotaId AND m.name = $mistakeName " +
					"CREATE (sa)-[e:DONE]->(m)" +
					"RETURN e",
				map[string]interface{}{
					"mistakeName": jsonSabota.Mistake,
					"sabotaId": newSabotaId,
				})
			return nil, nil
		})
		fmt.Println(err)
		return
	}
}

func (c SabotaController) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var validationError models.Error
		var jsonSabota models.Sabota // post内容のsabota
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
		if jsonSabota.Time == "" {
			validationError.Message = "時間がありません"
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
						"MATCH (s:Sabota)" +
						"WHERE ID(s) = $sabotaId SET " +
						"s.shouldDone = $shouldDone, "+
						"s.mistake = $mistake, "+
						"s.time = $time, "+
						"s.body = $body, "+
						"s.created_at = $created_at, " +
						"s.updated_at = $updated_at " +
						"RETURN ID(s);",
				map[string]interface{}{
					"sabotaId": sabotaId,
					"shouldDone": jsonSabota.ShouldDone,
					"mistake": jsonSabota.Mistake,
					"time": jsonSabota.Time,
					"body": jsonSabota.Body,
					"created_at": time.Now().Format("2006-01-02 15:04:05"),
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
				map[string]interface{}{"name": jsonSabota.ShouldDone })

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
				fmt.Println(count)
			}

			// その名前のShouldDoneノードが存在しない場合
			if count == 0 {
				// ShouldDoneノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:ShouldDone) SET " +
						"n.name = $name, " +
						"n.created_at = $created_at, " +
						"n.updated_at = $updated_at " +
						"RETURN n",
					map[string]interface{}{
						"name": jsonSabota.ShouldDone,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONTエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (sd:ShouldDone) " +
					"WHERE ID(sa) = $sabotaId AND sd.name = $shouldDoneName " +
					"CREATE (sa)-[e:DONT]->(sd)" +
					"RETURN e",
				map[string]interface{}{
					"shouldDoneName": jsonSabota.ShouldDone,
					"sabotaId": updatedSabotaId,
				})

			if err != nil {
				return nil, err
			}

			result, err = transaction.Run(
				"MATCH (n:Mistake) WHERE n.name = $name RETURN count(n)",
				map[string]interface{}{"name": jsonSabota.Mistake })

			if err != nil {
				return nil, err
			}

			if result.Next() {
				count = result.Record().GetByIndex(0).(int64)
				fmt.Println(count)
			}

			// その名前のMistakeノードが存在しない場合
			if count == 0 {
				// Mistakeノードを作成した後、エッジを作成
				result, err = transaction.Run(
					"CREATE (n:Mistake) SET " +
						"n.name = $name, " +
						"n.created_at = $created_at, " +
						"n.updated_at = $updated_at " +
						"RETURN n",
					map[string]interface{}{
						"name": jsonSabota.Mistake,
						"created_at": time.Now().Format("2006-01-02 15:04:05"),
						"updated_at": time.Now().Format("2006-01-02 15:04:05"),
					})

				if err != nil {
					return nil, err
				}

			}
			// DONEエッジを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (m:Mistake) " +
					"WHERE ID(sa) = $sabotaId AND m.name = $mistakeName " +
					"CREATE (sa)-[e:DONE]->(m)" +
					"RETURN e",
				map[string]interface{}{
					"mistakeName": jsonSabota.Mistake,
					"sabotaId": updatedSabotaId,
				})
			return nil, nil
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		return
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
			fmt.Println(postUserId)


			return postUserId, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		fmt.Println(postUserId)

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
