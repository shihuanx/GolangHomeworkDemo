package model

type Student struct {
	ID         string             `json:"id" validate:"required"`
	Name       string             `json:"name" validate:"required"`
	Gender     string             `json:"gender" validate:"required"`
	Class      string             `json:"class" validate:"required"`
	Grades     map[string]float64 `json:"grades"`
	Expiration int64              `json:"expiration"`
}

type StudentDB struct {
	ID     string `json:"id" validate:"required" gorm:"primaryKey"`
	Name   string `json:"name" validate:"required"`
	Gender string `json:"gender" validate:"required"`
	Class  string `json:"class" validate:"required"`
}

type Grade struct {
	ID        string  `json:"id" validate:"required"`
	Subject   string  `json:"subject" validate:"required"`
	Score     float64 `json:"score" validate:"required"`
	StudentId string  `json:"student_id" validate:"required"`
}

type StudentCount struct {
	ID        string `json:"id" validate:"required"`
	StudentId string `json:"student_id" validate:"required"`
	Count     int32  `json:"count" validate:"required"`
}
