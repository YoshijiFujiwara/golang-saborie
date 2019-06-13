package controllers

import (
	"net/http"
	"os"
	"portfolio/saborie/backend/models"
	"portfolio/saborie/backend/utils"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type MetooController struct{}

func (c MetooController) Index() http.HandlerFunc {
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

		metooUserList, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var metooUserList []models.User

			result, err = transaction.Run(
				"MATCH (s:Sabota)<-[e:METOO]-(u:User) WHERE ID(s) = $sabotaId RETURN ID(u), u.username ORDER BY ID(e) DESC;",
				map[string]interface{}{"sabotaId": sabotaId})

			if err != nil {
				return nil, err
			}

			for result.Next() {
				var user models.User
				user.ID = int(result.Record().GetByIndex(0).(int64)) // int64 -> intへの型キャスト
				user.Username = result.Record().GetByIndex(1).(string)

				metooUserList = append(metooUserList, user)
			}

			return metooUserList, result.Err()
		})

		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		// metooつけた人のuserのリストを返す
		utils.ResponseJSON(w, metooUserList)
	}
}

func (c MetooController) SwitchMetoo() http.HandlerFunc {
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

		// 自分の投稿にはつけれない
		count, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var count int

			result, err = transaction.Run(
				"MATCH (u:User)-[e:POST]->(s:Sabota) WHERE ID(s) = $sabotaId AND ID(u) = $userId RETURN count(e);",
				map[string]interface{}{
					"userId":   userId,
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				count = int(result.Record().GetByIndex(0).(int64))
			}

			return count, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}

		if count != 0 {
			validationError.Message = "自分の投稿にはつけられません"
			utils.RespondWithError(w, http.StatusBadRequest, validationError)
			return
		}

		_, err = session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var count int
			var metooEdgeId int

			result, err = transaction.Run(
				"MATCH (u:User)-[e:METOO]->(s:Sabota) WHERE ID(s) = $sabotaId AND ID(u) = $userId RETURN count(e), ID(e);",
				map[string]interface{}{
					"sabotaId": sabotaId,
					"userId":   userId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				count = int(result.Record().GetByIndex(0).(int64))
				metooEdgeId = int(result.Record().GetByIndex(1).(int64))
			}

			// すでにMetooをつけていたら、削除
			if count != 0 {
				result, err = transaction.Run(
					"MATCH (u:User)-[e:METOO]->(s:Sabota) WHERE ID(e) = $metooEdgeId DELETE e",
					map[string]interface{}{
						"metooEdgeId": metooEdgeId,
					})

				if err != nil {
					return nil, err
				}
			} else { // Metooがまだなら、つける

				result, err = transaction.Run(
					"MATCH (u:User), (s:Sabota) "+
						"WHERE ID(u) = $userId AND ID(s) = $sabotaId "+
						"CREATE (u)-[e:METOO]->(s) "+
						"RETURN e;",
					map[string]interface{}{
						"userId":   userId,
						"sabotaId": sabotaId,
					})

				if err != nil {
					return nil, err
				}
			}

			return count, result.Err()
		})
		if err != nil {
			validationError.Message = err.Error()
			utils.RespondWithError(w, http.StatusInternalServerError, validationError)
			return
		}
		return
	}
}
