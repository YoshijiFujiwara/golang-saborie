package models

type SabotaSearch struct {
	ShouldDone string `json:"shouldDone"`
	Mistake    string `json:"mistake"`
	Time       int    `json:"time"`
	KeyWord    string `json:"keyWord"`
}
