package controllers

import (
	"fmt"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"portfolio/saborie/utils"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type MetooController struct {}

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
					"userId": userId,
					"sabotaId": sabotaId,
				})

			if err != nil {
				return nil, err
			}
			if result.Next() {
				count = int(result.Record().GetByIndex(0).(int64))
			}

			fmt.Println(count)
			fmt.Println(sabotaId)
			fmt.Println(userId)

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
				"MATCH (u:User)-[e:METOO]->(s:Sabota) WHERE ID(s) = $sabotaId RETURN count(e), ID(e);",
				map[string]interface{}{
					"sabotaId": sabotaId,
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
					"MATCH (u:User), (s:Sabota) " +
							"WHERE ID(u) = $userId AND ID(s) = $sabotaId " +
							"CREATE (u)-[e:METOO]->(s) " +
							"RETURN e;",
					map[string]interface{}{
						"userId": userId,
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