package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Portfolio - portofolio mandiri milik satu student (MongoDB).
// student_id dijamin unik lewat index unique.
type Portfolio struct {
	ID         bson.ObjectID `bson:"_id,omitempty" json:"id"`
	StudentID  int           `bson:"student_id"   json:"student_id"`
	Summary    string        `bson:"summary"      json:"summary"`
	Skills     []string      `bson:"skills"       json:"skills"`
	Projects   []Project     `bson:"projects"     json:"projects"`
	UpdatedAt  time.Time     `bson:"updated_at"   json:"updated_at"`
	CreatedAt  time.Time     `bson:"created_at"   json:"created_at"`
}

// Project - satu proyek dalam portofolio (maksimal 20).
type Project struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string        `bson:"name"          json:"name"`
	Description  string        `bson:"description"   json:"description"`
	URL          string        `bson:"url"           json:"url"`
	Technologies []string      `bson:"technologies"  json:"technologies"`
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

// UpsertPortfolioRequest - request PUT /students/:id/portfolio.
type UpsertPortfolioRequest struct {
	Summary  string    `json:"summary"  validate:"max=2000"`
	Skills   []string  `json:"skills"`
	Projects []Project `json:"projects" validate:"max=20,dive"`
}

// AddProjectRequest - request menambah satu proyek (maksimal 20 proyek).
type AddProjectRequest struct {
	Name         string   `json:"name"         validate:"required,max=100"`
	Description  string   `json:"description"  validate:"max=1000"`
	URL          string   `json:"url"          validate:"max=500"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=10"`
}

// TechnologyStat - hasil group TopTechnologies.
type TechnologyStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
