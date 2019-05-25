package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"portfolio/saborie/models"

	"github.com/davecgh/go-spew/spew"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

// todo どこをtokenミドルウェア通すかを決めておく

func (c Controller) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (c Controller) Show() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (c Controller) StoreSabota() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ログインユーザーID
		userId := r.Context().Value("userId")
		var jsonSabota models.Sabota // post内容のsabota

		// リクエスト内容をデコードして作成するsabotaデータを取り出す
		json.NewDecoder(r.Body).Decode(&jsonSabota)
		spew.Dump(jsonSabota)

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

		// sabota新規作成
		newSabotaId, err := session.WriteTransaction(func(transaction neo4j.Transaction) (interface{}, error) {
			var newSabotaId int64

			result, err = transaction.Run(
				"CREATE (s:Sabota) SET " +
					"s.time = $time, "+
					"s.body = $body "+
					"RETURN ID(s);",
				map[string]interface{}{"time": jsonSabota.Time, "body": jsonSabota.Body})

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
					"CREATE (n:ShouldDone) SET n.name = $name RETURN n",
					map[string]interface{}{"name": jsonSabota.ShouldDone })

				if err != nil {
					return nil, err
				}

			}
			// DONTノードを作成
			result, err = transaction.Run(
				"MATCH (sa:Sabota), (sd:ShouldDone) " +
					"WHERE ID(sa) = $sabotaId AND sd.name = $shouldDoneName " +
					"CREATE (sa)-[e:DONT]->(sd)" +
					"RETURN e",
				map[string]interface{}{"shouldDoneName": jsonSabota.ShouldDone, "sabotaId": newSabotaId })

			if err != nil {
				return nil, err
			}
			return nil, nil
		})
		fmt.Println(err)
		return
	}
}

func (c Controller) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (c Controller) Destroy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
