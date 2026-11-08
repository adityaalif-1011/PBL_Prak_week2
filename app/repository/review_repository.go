package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

const (
	reviewCollection = "course_reviews"
	maxReplies       = 49
)

var (
	ErrReviewNotFound = errors.New("review not found")
	ErrReplyLimit     = errors.New("reply limit reached")
	ErrAlreadyHelpful = errors.New("review already marked helpful")
)

// ReviewRepository adalah KONTRAK penyimpanan ulasan course di MongoDB.
// Berbentuk interface agar bisa dipalsukan (fake) di unit test.
type ReviewRepository interface {
	// EnsureIndexes membuat index yang dibutuhkan (idempoten).
	EnsureIndexes(ctx context.Context) error
	// Create menyimpan review baru (replies, helpful_by, tags diisi kosong).
	Create(ctx context.Context, req model.CreateReviewRequest) (model.Review, error)
	// FindByCourse mengambil semua review satu course, terbaru di atas.
	FindByCourse(ctx context.Context, courseID int) ([]model.Review, error)
	// Summary menghitung ringkasan review satu course.
	Summary(ctx context.Context, courseID int) (model.ReviewSummary, error)
	// AddReply menambah balasan pada review.
	AddReply(ctx context.Context, reviewID bson.ObjectID, req model.AddReplyRequest) (model.Review, error)
	// MarkHelpful menandai review membantu.
	MarkHelpful(ctx context.Context, reviewID bson.ObjectID, studentID int) (model.Review, error)
}

// Implementasi MongoDB.
type mongoReviewRepository struct {
	col *mongo.Collection
}

func NewReviewRepository(db *mongo.Database) ReviewRepository {
	return &mongoReviewRepository{col: db.Collection(reviewCollection)}
}

// EnsureIndexes membuat index yang dibutuhkan (idempoten).
func (r *mongoReviewRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys: bson.D{{Key: "course_id", Value: 1}, {Key: "created_at", Value: -1}},
		},
		{
			// Satu student hanya boleh mengulas satu course sekali.
			Keys:    bson.D{{Key: "course_id", Value: 1}, {Key: "student_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})
	return err
}

// Create - simpan review baru (replies & helpful_by diisi kosong).
func (r *mongoReviewRepository) Create(ctx context.Context, req model.CreateReviewRequest) (model.Review, error) {
	review := model.Review{
		ID:        bson.NewObjectID(),
		CourseID:  req.CourseID,
		StudentID: req.StudentID,
		Rating:    req.Rating,
		Comment:   req.Comment,
		Tags:      req.Tags,
		Replies:   []model.Reply{},
		HelpfulBy: []int{},
		CreatedAt: time.Now().UTC(),
	}

	if _, err := r.col.InsertOne(ctx, review); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return model.Review{}, ErrDuplicate
		}
		return model.Review{}, err
	}
	return review, nil
}

// FindByCourse - ambil semua review satu course, terbaru di atas.
func (r *mongoReviewRepository) FindByCourse(ctx context.Context, courseID int) ([]model.Review, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cur, err := r.col.Find(ctx, bson.D{{Key: "course_id", Value: courseID}}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var list []model.Review
	if err := cur.All(ctx, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// AddReply - tambah balasan dengan $push, dibatasi maksimal maxReplies.
// Filter "$expr: size < 49" memastikan push hanya terjadi bila masih muat.
func (r *mongoReviewRepository) AddReply(ctx context.Context, reviewID bson.ObjectID, req model.AddReplyRequest) (model.Review, error) {
	reply := model.Reply{
		StudentID: req.StudentID,
		Comment:   req.Comment,
		CreatedAt: time.Now().UTC(),
	}

	filter := bson.D{
		{Key: "_id", Value: reviewID},
		{Key: "$expr", Value: bson.D{{Key: "$lt", Value: bson.A{bson.D{{Key: "$size", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$replies", bson.A{}}}}}}, maxReplies}}}},
	}
	update := bson.D{{Key: "$push", Value: bson.D{{Key: "replies", Value: reply}}}}

	var updated model.Review
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			// Bedakan review tidak ada vs kuota balasan penuh.
			var count int64
			count, err = r.col.CountDocuments(ctx, bson.D{{Key: "_id", Value: reviewID}})
			if err == nil && count == 0 {
				return model.Review{}, ErrReviewNotFound
			}
			return model.Review{}, ErrReplyLimit
		}
		return model.Review{}, err
	}
	return updated, nil
}

// MarkHelpful - tandai review membantu memakai $addToSet + $inc.
// $addToSet mencegah duplikasi; $inc menaikkan hitungan.
func (r *mongoReviewRepository) MarkHelpful(ctx context.Context, reviewID bson.ObjectID, studentID int) (model.Review, error) {
	filter := bson.D{
		{Key: "_id", Value: reviewID},
		{Key: "helpful_by", Value: bson.D{{Key: "$ne", Value: studentID}}}, // tolak bila sudah menandai
	}
	update := bson.D{
		{Key: "$addToSet", Value: bson.D{{Key: "helpful_by", Value: studentID}}},
		{Key: "$inc", Value: bson.D{{Key: "helpful_count", Value: 1}}},
	}

	var updated model.Review
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&updated)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			var count int64
			count, err = r.col.CountDocuments(ctx, bson.D{{Key: "_id", Value: reviewID}})
			if err == nil && count == 0 {
				return model.Review{}, ErrReviewNotFound
			}
			return model.Review{}, ErrAlreadyHelpful
		}
		return model.Review{}, err
	}
	return updated, nil
}

// Summary - agregasi ringkasan per course memakai $facet + $unwind.
// $facet menjalankan beberapa cabang hitung dalam satu pass;
// $unwind memecah Replies agar tiap balasan bisa dihitung.
func (r *mongoReviewRepository) Summary(ctx context.Context, courseID int) (model.ReviewSummary, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "course_id", Value: courseID}}}},
		{{Key: "$facet", Value: bson.D{
			{Key: "overview", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: nil},
					{Key: "total_reviews", Value: bson.D{{Key: "$sum", Value: 1}}},
					{Key: "avg_rating", Value: bson.D{{Key: "$avg", Value: "$rating"}}},
					{Key: "total_helpful", Value: bson.D{{Key: "$sum", Value: "$helpful_count"}}},
				}}},
			}},
			{Key: "replies", Value: bson.A{
				bson.D{{Key: "$unwind", Value: "$replies"}}, // pecah array Replies
				bson.D{{Key: "$count", Value: "total"}},
			}},
			{Key: "distribution", Value: bson.A{
				bson.D{{Key: "$group", Value: bson.D{
					{Key: "_id", Value: "$rating"},
					{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
				}}},
			}},
		}}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return model.ReviewSummary{}, err
	}
	defer cur.Close(ctx)

	var raw []struct {
		Overview []struct {
			TotalReviews int     `bson:"total_reviews"`
			AvgRating    float64 `bson:"avg_rating"`
			TotalHelpful int     `bson:"total_helpful"`
		} `bson:"overview"`
		Replies []struct {
			Total int `bson:"total"`
		} `bson:"replies"`
		Distribution []struct {
			Rating int `bson:"_id"`
			Count  int `bson:"count"`
		} `bson:"distribution"`
	}
	if err := cur.All(ctx, &raw); err != nil {
		return model.ReviewSummary{}, err
	}

	summary := model.ReviewSummary{
		CourseID:   courseID,
		RatingDist: map[string]int{},
	}
	if len(raw) > 0 {
		if len(raw[0].Overview) > 0 {
			summary.TotalReviews = raw[0].Overview[0].TotalReviews
			summary.AvgRating = raw[0].Overview[0].AvgRating
			summary.TotalHelpful = raw[0].Overview[0].TotalHelpful
		}
		if len(raw[0].Replies) > 0 {
			summary.TotalReplies = raw[0].Replies[0].Total
		}
		for _, d := range raw[0].Distribution {
			summary.RatingDist[strconv.Itoa(d.Rating)] = d.Count
		}
	}
	return summary, nil
}
