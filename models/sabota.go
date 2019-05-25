package models

type Sabota struct {
	ID         int    `json:"id"`
	ShouldDone string `json:"shouldDone"`
	Mistake    string `json:"mistake"`
	Time       string `json:"time"`
	Body       string `json:"body"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
