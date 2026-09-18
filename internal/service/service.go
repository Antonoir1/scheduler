package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/Antonoir1/scheduler"
	"github.com/Antonoir1/scheduler/internal/job"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// Publisher sends job execution notifications.
type Publisher interface {
	Publish(string, []byte) error
}

// Store persists jobs and their latest execution results.
type Store interface {
	CreateJob(context.Context, *job.Job) error
	ListJobs(context.Context) ([]*job.Job, error)
	GetJob(context.Context, string) (*job.Job, error)
	UpdateJob(context.Context, *job.Job) error
	DeleteJob(context.Context, string) error
	SaveResult(context.Context, string, *job.JobResult) error
}

// Service manages job persistence, cron schedules, and webhook execution.
type Service struct {
	store     Store
	publisher Publisher
	cron      *cron.Cron
	client    *http.Client
	mu        sync.Mutex
	entries   map[string]cron.EntryID
	logger    zerolog.Logger
}

// New creates a scheduler service. A logger is optional for compatibility with callers that do not need output.
func New(store Store, publisher Publisher, loggers ...zerolog.Logger) *Service {
	logger := zerolog.Nop()
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return &Service{store: store, publisher: publisher, cron: cron.New(), client: &http.Client{Timeout: 30 * time.Second}, entries: make(map[string]cron.EntryID), logger: logger}
}

// Start restores persisted schedules and starts cron execution.
func (s *Service) Start(ctx context.Context) error {
	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("list jobs while starting scheduler")
		return err
	}
	for _, job := range jobs {
		if err := s.register(job); err != nil {
			wrappedErr := fmt.Errorf("register persisted schedule %s: %w", job.ID, err)
			s.logger.Error().Err(wrappedErr).Str("job_id", job.ID).Msg("register persisted schedule")
			return wrappedErr
		}
	}
	s.cron.Start()
	return nil
}

// Stop waits for scheduled executions to stop.
func (s *Service) Stop() {
	s.logger.Info().Msg("stop scheduler")
	<-s.cron.Stop().Done()
}

// CreateJob persists a job and registers its cron schedule.
func (s *Service) CreateJob(ctx context.Context, request scheduler.CreateJobRequest) (*job.Job, error) {
	now := time.Now().UTC()
	newJob := &job.Job{ID: newID(), Name: request.Name, Schedule: request.Schedule, WebhookURL: request.WebhookURL, Parameters: request.Parameters, Headers: request.Headers, Payload: request.Payload, CreatedAt: now, UpdatedAt: now}
	if _, err := cron.ParseStandard(newJob.Schedule); err != nil {
		wrappedErr := fmt.Errorf("invalid schedule: %w", err)
		s.logger.Error().Err(wrappedErr).Str("schedule", newJob.Schedule).Msg("create job")
		return nil, wrappedErr
	}
	if err := s.store.CreateJob(ctx, newJob); err != nil {
		s.logger.Error().Err(err).Str("job_id", newJob.ID).Msg("store new job")
		return nil, err
	}
	if err := s.register(newJob); err != nil {
		wrappedErr := fmt.Errorf("register schedule: %w", err)
		s.logger.Error().Err(wrappedErr).Str("job_id", newJob.ID).Msg("register job schedule")
		return nil, wrappedErr
	}
	return newJob, nil
}

// ListJobs returns all persisted jobs.
func (s *Service) ListJobs(ctx context.Context) ([]*job.Job, error) {
	jobs, err := s.store.ListJobs(ctx)
	if err != nil {
		s.logger.Error().Err(err).Msg("list jobs")
	}
	return jobs, err
}

// GetJob returns one persisted job by ID.
func (s *Service) GetJob(ctx context.Context, id string) (*job.Job, error) {
	result, err := s.store.GetJob(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("get job")
	}
	return result, err
}

// UpdateJob persists new job configuration and replaces its cron schedule.
func (s *Service) UpdateJob(ctx context.Context, id string, request scheduler.CreateJobRequest) (*job.Job, error) {
	updatedJob, err := s.store.GetJob(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("load job for update")
		return nil, err
	}
	if _, err := cron.ParseStandard(request.Schedule); err != nil {
		wrappedErr := fmt.Errorf("invalid schedule: %w", err)
		s.logger.Error().Err(wrappedErr).Str("job_id", id).Msg("update job")
		return nil, wrappedErr
	}
	updatedJob.Name = request.Name
	updatedJob.Schedule = request.Schedule
	updatedJob.WebhookURL = request.WebhookURL
	updatedJob.Parameters = request.Parameters
	updatedJob.Headers = request.Headers
	updatedJob.Payload = request.Payload
	updatedJob.UpdatedAt = time.Now().UTC()
	if err := s.store.UpdateJob(ctx, updatedJob); err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("store updated job")
		return nil, err
	}
	s.removeSchedule(id)
	if err := s.register(updatedJob); err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("register updated job schedule")
		return nil, err
	}
	return updatedJob, nil
}

// DeleteJob removes a persisted job and its cron schedule.
func (s *Service) DeleteJob(ctx context.Context, id string) error {
	if err := s.store.DeleteJob(ctx, id); err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("delete job")
		return err
	}
	s.removeSchedule(id)
	return nil
}

func (s *Service) register(job *job.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entryID, err := s.cron.AddFunc(job.Schedule, func() { s.execute(job.ID) })
	if err == nil {
		s.entries[job.ID] = entryID
	}
	return err
}

func (s *Service) removeSchedule(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if entryID, ok := s.entries[id]; ok {
		s.cron.Remove(entryID)
		delete(s.entries, id)
	}
}

func (s *Service) execute(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	internalJob, err := s.store.GetJob(ctx, id)
	if err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("load job for execution")
		return
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, internalJob.WebhookURL, bytes.NewReader([]byte(internalJob.Payload)))
	result := &job.JobResult{JobID: internalJob.ID, ExecutedAt: time.Now().UTC()}
	if err == nil {
		for key, value := range internalJob.Headers {
			request.Header.Set(key, value)
		}
		response, requestErr := s.client.Do(request)
		if requestErr != nil {
			result.Error = requestErr.Error()
			s.logger.Error().Err(requestErr).Str("job_id", id).Msg("call job webhook")
		} else {
			result.StatusCode = response.StatusCode
			result.Response, _ = io.ReadAll(io.LimitReader(response.Body, 1<<20))
			_ = response.Body.Close()
		}
	} else {
		result.Error = err.Error()
		s.logger.Error().Err(err).Str("job_id", id).Msg("build webhook request")
	}
	if err := s.store.SaveResult(ctx, id, result); err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("store job result")
		return
	}
	if s.publisher == nil {
		return
	}
	message, _ := json.Marshal(result)
	if err := s.publisher.Publish("scheduler.job.executed", message); err != nil {
		s.logger.Error().Err(err).Str("job_id", id).Msg("publish job execution notification")
	}
}

func newID() string {
	return fmt.Sprintf("job-%d", time.Now().UnixNano())
}
