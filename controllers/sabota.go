package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"portfolio/saborie/models"
	"portfolio/saborie/utils"

	"github.com/davecgh/go-spew/spew"
	"github.com/neo4j/neo4j-go-driver/neo4j"
)

type SabotaController struct {}


func (c SabotaController) Index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ログインユーザーID
		//userId := r.Context().Value("userId")

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

		// sabota新規作成
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

	}
}

func (c SabotaController) Store() http.HandlerFunc {
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
					"s.shouldDone = $shouldDone, "+
					"s.mistake = $mistake, "+
					"s.time = $time, "+
					"s.body = $body "+
					"RETURN ID(s);",
				map[string]interface{}{"shouldDone": jsonSabota.ShouldDone, "mistake": jsonSabota.Mistake, "time": jsonSabota.Time, "body": jsonSabota.Body})

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

func (c SabotaController) Update() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}

func (c SabotaController) Destroy() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

	}
}
