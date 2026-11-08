package model

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Review - ulasan mahasiswa terhadap sebuah course (MongoDB).
type Review struct {
	ID           bson.ObjectID `bson:"_id,omitempty" json:"id"`
	CourseID     int           `bson:"course_id"     json:"course_id"`
	StudentID    int           `bson:"student_id"    json:"student_id"`
	Rating       int           `bson:"rating"        json:"rating"`
	Comment      string        `bson:"comment"       json:"comment"`
	Tags         []string      `bson:"tags,omitempty" json:"tags,omitempty"`
	Replies      []Reply       `bson:"replies"       json:"replies"`
	HelpfulBy    []int         `bson:"helpful_by"    json:"helpful_by"`
	HelpfulCount int           `bson:"helpful_count" json:"helpful_count"`
	CreatedAt    time.Time     `bson:"created_at"    json:"created_at"`
}

// Reply - balasan pada review.
type Reply struct {
	StudentID int       `bson:"student_id" json:"student_id"`
	Comment   string    `bson:"comment"    json:"comment"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// CreateReviewRequest - request membuat review.
type CreateReviewRequest struct {
	CourseID  int      `json:"course_id"  validate:"required,gt=0"`
	StudentID int      `json:"student_id" validate:"required,gt=0"`
	Rating    int      `json:"rating"     validate:"required,gte=1,lte=5"`
	Comment   string   `json:"comment"    validate:"required,max=1000"`
	Tags      []string `json:"tags"`
}

// AddReplyRequest - request membalas review.
type AddReplyRequest struct {
	StudentID int    `json:"student_id" validate:"required,gt=0"`
	Comment   string `json:"comment"    validate:"required,max=1000"`
}

// MarkHelpfulRequest - request menandai review membantu.
type MarkHelpfulRequest struct {
	StudentID int `json:"student_id" validate:"required,gt=0"`
}

// ReviewSummary - hasil agregasi ringkasan review per course.
type ReviewSummary struct {
	CourseID     int            `json:"course_id"`
	TotalReviews int            `json:"total_reviews"`
	AvgRating    float64        `json:"avg_rating"`
	TotalReplies int            `json:"total_replies"`
	TotalHelpful int            `json:"total_helpful"`
	RatingDist   map[string]int `json:"rating_distribution"`
}

// ReviewPage - response berhalaman untuk daftar review satu course.
type ReviewPage struct {
	CourseID int            `json:"course_id"`
	Reviews  []Review       `json:"reviews"`
	Meta     ReviewPageMeta `json:"meta"`
}

// ReviewPageMeta - metadata halaman ReviewPage.
type ReviewPageMeta struct {
	// Total jumlah review pada halaman ini (seluruh review course).
	Total int `json:"total"`
}
