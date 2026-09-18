package storage

import (
	"context"

	"github.com/Antonoir1/scheduler/internal/job"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Store interface {
	CreateJob(context.Context, *job.Job) error
	ListJobs(context.Context) ([]*job.Job, error)
	GetJob(context.Context, string) (*job.Job, error)
	UpdateJob(context.Context, *job.Job) error
	DeleteJob(context.Context, string) error
	SaveResult(context.Context, string, *job.JobResult) error
}

type MongoStore struct {
	client     *mongo.Client
	collection *mongo.Collection
}

func NewMongoStore(ctx context.Context, uri, database, collection string) (*MongoStore, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(uri))
	if err != nil {
		return nil, err
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, err
	}
	return &MongoStore{client: client, collection: client.Database(database).Collection(collection)}, nil
}

func (s *MongoStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

func (s *MongoStore) CreateJob(ctx context.Context, job *job.Job) error {
	_, err := s.collection.InsertOne(ctx, job)
	return err
}

func (s *MongoStore) GetJob(ctx context.Context, id string) (*job.Job, error) {
	var result job.Job
	if err := s.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *MongoStore) ListJobs(ctx context.Context) ([]*job.Job, error) {
	cursor, err := s.collection.Find(ctx, bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	var jobs []*job.Job
	if err := cursor.All(ctx, &jobs); err != nil {
		return nil, err
	}
	return jobs, nil
}

func (s *MongoStore) UpdateJob(ctx context.Context, job *job.Job) error {
	result, err := s.collection.ReplaceOne(ctx, bson.M{"_id": job.ID}, job)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) DeleteJob(ctx context.Context, id string) error {
	result, err := s.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return mongo.ErrNoDocuments
	}
	return nil
}

func (s *MongoStore) SaveResult(ctx context.Context, id string, result *job.JobResult) error {
	_, err := s.collection.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"last_result": result,
		"updated_at":  result.ExecutedAt,
	}})
	return err
}
