package models

type Sabota struct {
	ID         int    `json:"id"`
	ShouldDone string `json:"shouldDone"`
	Mistake    string `json:"mistake"`
	Time       int    `json:"time"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`

	PostUser User `json:"postUser"`

	MetooUserIds   []int `json:"metooUserIds"`
	LoveUserIds    []int `json:"loveUserIds"`
	CommentUserIds []int `json:"commentUserIds"`

	Comments []Comment `json:"comments"`
}
