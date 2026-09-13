package model

import "time"

type Achievement struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	Name          string    `json:"name"`
	ChampionLevel string    `json:"champion_level"`
	CreatedAt     time.Time `json:"created_at"`
}

type CreateAchievementRequest struct {
	StudentID     int    `json:"student_id"`
	Name          string `json:"name"`
	ChampionLevel string `json:"champion_level"`
}

type UpdateAchievementRequest struct {
	Name          string `json:"name"`
	ChampionLevel string `json:"champion_level"`
}

type PatchAchievementRequest struct {
	Name          *string `json:"name,omitempty"`
	ChampionLevel *string `json:"champion_level,omitempty"`
}
