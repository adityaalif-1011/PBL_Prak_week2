package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"api-students/app/model"
)

const (
	portfolioCollection = "portfolios"
	maxProjects         = 20
)

var (
	ErrPortfolioNotFound = errors.New("portfolio not found")
	ErrProjectLimit      = errors.New("project limit reached")
	ErrProjectNotFound   = errors.New("project not found")
)

// PortfolioRepository menyimpan portofolio mandiri student di MongoDB.
type PortfolioRepository struct {
	col *mongo.Collection
}

func NewPortfolioRepository(db *mongo.Database) *PortfolioRepository {
	return &PortfolioRepository{col: db.Collection(portfolioCollection)}
}

// EnsureIndexes membuat index unique student_id (satu student satu portofolio).
func (r *PortfolioRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "student_id", Value: 1}},
		Options: options.Index().SetUnique(true),
	})
	return err
}

// FindByStudentID - ambil portofolio milik satu student.
func (r *PortfolioRepository) FindByStudentID(ctx context.Context, studentID int) (model.Portfolio, error) {
	var p model.Portfolio
	err := r.col.FindOne(ctx, bson.D{{Key: "student_id", Value: studentID}}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, err
	}
	return p, nil
}

// Upsert - buat atau timpa seluruh portofolio (GET/PUT).
func (r *PortfolioRepository) Upsert(ctx context.Context, studentID int, req model.UpsertPortfolioRequest) (model.Portfolio, error) {
	now := time.Now().UTC()

	projects := req.Projects
	if projects == nil {
		projects = []model.Project{}
	}
	// Pastikan tiap project punya id & waktu.
	for i := range projects {
		if projects[i].ID.IsZero() {
			projects[i].ID = bson.NewObjectID()
		}
		if projects[i].CreatedAt.IsZero() {
			projects[i].CreatedAt = now
		}
	}
	skills := req.Skills
	if skills == nil {
		skills = []string{}
	}

	update := bson.D{
		{Key: "$set", Value: bson.D{
			{Key: "summary", Value: req.Summary},
			{Key: "skills", Value: skills},
			{Key: "projects", Value: projects},
			{Key: "updated_at", Value: now},
		}},
		{Key: "$setOnInsert", Value: bson.D{
			{Key: "student_id", Value: studentID},
			{Key: "created_at", Value: now},
		}},
	}
	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var p model.Portfolio
	err := r.col.FindOneAndUpdate(ctx, bson.D{{Key: "student_id", Value: studentID}}, update, opts).Decode(&p)
	if err != nil {
		return model.Portfolio{}, err
	}
	return p, nil
}

// AddProject - tambah satu proyek, dibatasi maksimal maxProjects.
// Guard "projects.19 tidak boleh ada" memastikan push hanya terjadi bila
// panjang projects masih di bawah 20 (indeks 0..19).
func (r *PortfolioRepository) AddProject(ctx context.Context, studentID int, req model.AddProjectRequest) (model.Portfolio, error) {
	project := model.Project{
		ID:           bson.NewObjectID(),
		Name:         req.Name,
		Description:  req.Description,
		URL:          req.URL,
		Technologies: req.Technologies,
		CreatedAt:    time.Now().UTC(),
	}

	filter := bson.D{
		{Key: "student_id", Value: studentID},
		{Key: "projects.19", Value: bson.D{{Key: "$exists", Value: false}}}, // belum ada proyek ke-20
	}
	update := bson.D{
		{Key: "$push", Value: bson.D{{Key: "projects", Value: project}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
	}

	var p model.Portfolio
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err := r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			var count int64
			count, err = r.col.CountDocuments(ctx, bson.D{{Key: "student_id", Value: studentID}})
			if err == nil && count == 0 {
				return model.Portfolio{}, ErrPortfolioNotFound
			}
			return model.Portfolio{}, ErrProjectLimit
		}
		return model.Portfolio{}, err
	}
	return p, nil
}

// DeleteProject - hapus satu proyek berdasar id proyek.
func (r *PortfolioRepository) DeleteProject(ctx context.Context, studentID int, projectID string) (model.Portfolio, error) {
	oid, err := bson.ObjectIDFromHex(projectID)
	if err != nil {
		return model.Portfolio{}, ErrProjectNotFound
	}

	filter := bson.D{{Key: "student_id", Value: studentID}}
	update := bson.D{
		{Key: "$pull", Value: bson.D{{Key: "projects", Value: bson.D{{Key: "_id", Value: oid}}}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
	}

	var p model.Portfolio
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)
	err = r.col.FindOneAndUpdate(ctx, filter, update, opts).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return model.Portfolio{}, ErrPortfolioNotFound
		}
		return model.Portfolio{}, err
	}

	// Bila jumlah proyek tidak berkurang, proyek itu tidak ditemukan.
	for _, proj := range p.Projects {
		if proj.ID == oid {
			return p, nil
		}
	}
	return model.Portfolio{}, ErrProjectNotFound
}

// TopTechnologies - hitung teknologi paling sering dipakai memakai 2x $unwind
// (pecah array projects, lalu pecah technologies di tiap project).
func (r *PortfolioRepository) TopTechnologies(ctx context.Context, studentID int) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.D{{Key: "student_id", Value: studentID}}}},
		{{Key: "$unwind", Value: "$projects"}},                  // unwrap projects
		{{Key: "$unwind", Value: "$projects.technologies"}},     // unwrap technologies
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$projects.technologies"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var stats []model.TechnologyStat
	for cur.Next(ctx) {
		var row struct {
			Tech  string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		stats = append(stats, model.TechnologyStat{Technology: row.Tech, Count: row.Count})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []model.TechnologyStat{}
	}
	return stats, nil
}

// TopTechnologiesAll - agregasi lintas seluruh portofolio untuk endpoint stats.
func (r *PortfolioRepository) TopTechnologiesAll(ctx context.Context) ([]model.TechnologyStat, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$unwind", Value: "$projects"}},
		{{Key: "$unwind", Value: "$projects.technologies"}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$projects.technologies"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}, {Key: "_id", Value: 1}}}},
		{{Key: "$limit", Value: 20}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var stats []model.TechnologyStat
	for cur.Next(ctx) {
		var row struct {
			Tech  string `bson:"_id"`
			Count int    `bson:"count"`
		}
		if err := cur.Decode(&row); err != nil {
			return nil, err
		}
		stats = append(stats, model.TechnologyStat{Technology: row.Tech, Count: row.Count})
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}
	if stats == nil {
		stats = []model.TechnologyStat{}
	}
	return stats, nil
}
