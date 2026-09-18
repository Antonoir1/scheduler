package client

import (
	"fmt"
	"net/url"

	scheduler "github.com/Antonoir1/scheduler"
	"github.com/go-resty/resty/v2"
)

// Client calls the scheduler HTTP API.
type Client struct {
	http *resty.Client
}

// APIError describes a non-success response from the scheduler API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("scheduler API returned %d: %s", e.StatusCode, e.Message)
}

// New creates a scheduler API client using baseURL.
func New(baseURL string) *Client {
	return &Client{http: resty.New().SetBaseURL(baseURL)}
}

// NewWithResty creates a scheduler API client from an existing Resty client.
func NewWithResty(restyClient *resty.Client) *Client {
	return &Client{http: restyClient}
}

// ListJobs returns all scheduled jobs.
func (c *Client) ListJobs() ([]*scheduler.Job, error) {
	var jobs []*scheduler.Job
	response, err := c.http.R().SetResult(&jobs).Get("/jobs")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return jobs, nil
}

// CreateJob creates and schedules a new job.
func (c *Client) CreateJob(request scheduler.CreateJobRequest) (*scheduler.Job, error) {
	var job scheduler.Job
	response, err := c.http.R().SetBody(request).SetResult(&job).Post("/jobs")
	if err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return &job, nil
}

// GetJob returns a job without embedding its result.
func (c *Client) GetJob(id string) (*scheduler.Job, error) {
	var job scheduler.Job
	response, err := c.http.R().SetResult(&job).Get(jobPath(id))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return &job, nil
}

// UpdateJob replaces a job's schedule and webhook configuration.
func (c *Client) UpdateJob(id string, request scheduler.CreateJobRequest) (*scheduler.Job, error) {
	var job scheduler.Job
	response, err := c.http.R().SetBody(request).SetResult(&job).Put(jobPath(id))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return &job, nil
}

// DeleteJob removes a scheduled job.
func (c *Client) DeleteJob(id string) error {
	response, err := c.http.R().Delete(jobPath(id))
	if err != nil {
		return err
	}
	return checkResponse(response)
}

// GetJobResult returns only the latest result for a job.
func (c *Client) GetJobResult(id string) (*scheduler.JobResult, error) {
	var result scheduler.JobResult
	response, err := c.http.R().SetResult(&result).Get(jobResultPath(id))
	if err != nil {
		return nil, err
	}
	if err := checkResponse(response); err != nil {
		return nil, err
	}
	return &result, nil
}

func jobPath(id string) string {
	return "/jobs/" + url.PathEscape(id)
}

func jobResultPath(id string) string {
	return jobPath(id) + "/results"
}

func checkResponse(response *resty.Response) error {
	if response.IsSuccess() {
		return nil
	}
	message := response.Status()
	if body := response.String(); body != "" {
		message = body
	}
	return &APIError{StatusCode: response.StatusCode(), Message: message}
}
