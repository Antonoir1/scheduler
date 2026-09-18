package handlers

import (
	"errors"
	"net/http"

	"github.com/Antonoir1/scheduler"
	internaljob "github.com/Antonoir1/scheduler/internal/job"
	"github.com/Antonoir1/scheduler/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RegisterHandlers attaches the scheduler API routes to router.
func RegisterHandlers(router *gin.Engine, jobs *service.Service, loggers ...zerolog.Logger) {
	logger := zerolog.Nop()
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	router.GET("/jobs", listJobs(jobs, logger))
	router.POST("/jobs", createJob(jobs, logger))
	router.GET("/jobs/:id", getJob(jobs, logger))
	router.PUT("/jobs/:id", updateJob(jobs, logger))
	router.DELETE("/jobs/:id", deleteJob(jobs, logger))
	router.GET("/jobs/:id/results", getResult(jobs, logger))
}

func listJobs(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "list_jobs").Msg("handler called")
		result, err := jobs.ListJobs(c.Request.Context())
		if err != nil {
			logger.Error().Err(err).Msg("list jobs handler failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, publicJobs(result))
	}
}

func createJob(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "create_job").Msg("handler called")
		var request scheduler.CreateJobRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Error().Err(err).Msg("decode create job request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		job, err := jobs.CreateJob(c.Request.Context(), request)
		if err != nil {
			logger.Error().Err(err).Msg("create job handler failed")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, publicJob(job))
	}
}

func getJob(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "get_job").Str("job_id", c.Param("id")).Msg("handler called")
		job, err := jobs.GetJob(c.Request.Context(), c.Param("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		if err != nil {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("get job handler failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, publicJob(job))
	}
}

func updateJob(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "update_job").Str("job_id", c.Param("id")).Msg("handler called")
		var request scheduler.CreateJobRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("decode update job request")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		job, err := jobs.UpdateJob(c.Request.Context(), c.Param("id"), request)
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		if err != nil {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("update job handler failed")
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, publicJob(job))
	}
}

func deleteJob(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "delete_job").Str("job_id", c.Param("id")).Msg("handler called")
		err := jobs.DeleteJob(c.Request.Context(), c.Param("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		if err != nil {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("delete job handler failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.Status(http.StatusNoContent)
	}
}

func getResult(jobs *service.Service, logger zerolog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		logger.Info().Str("handler", "get_job_result").Str("job_id", c.Param("id")).Msg("handler called")
		job, err := jobs.GetJob(c.Request.Context(), c.Param("id"))
		if errors.Is(err, mongo.ErrNoDocuments) {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("job not found")
			c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
			return
		}
		if err != nil {
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("get job result handler failed")
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		if job.LastResult == nil {
			err := errors.New("job has no result")
			logger.Error().Err(err).Str("job_id", c.Param("id")).Msg("get job result handler failed")
			c.JSON(http.StatusNotFound, gin.H{"error": "job has no result"})
			return
		}
		c.JSON(http.StatusOK, publicResult(job.LastResult))
	}
}

func publicJobs(jobs []*internaljob.Job) []*scheduler.Job {
	result := make([]*scheduler.Job, 0, len(jobs))
	for _, item := range jobs {
		result = append(result, publicJob(item))
	}
	return result
}

func publicJob(item *internaljob.Job) *scheduler.Job {
	if item == nil {
		return nil
	}
	return &scheduler.Job{ID: item.ID, Name: item.Name, Schedule: item.Schedule, WebhookURL: item.WebhookURL, Parameters: item.Parameters, Payload: item.Payload, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func publicResult(item *internaljob.JobResult) *scheduler.JobResult {
	if item == nil {
		return nil
	}
	return &scheduler.JobResult{JobID: item.JobID, StatusCode: item.StatusCode, Response: item.Response, Error: item.Error, ExecutedAt: item.ExecutedAt}
}
