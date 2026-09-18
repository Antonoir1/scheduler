## Scheduler service

The service creates cron-backed webhook jobs, stores the latest execution result in MongoDB, and publishes a NATS notification after each stored execution.

### Run dependencies

```sh
docker compose up -d
go run ./cmd/server
```

Environment variables:

- `HTTP_ADDR` (default `:8080`)
- `MONGO_URI` (default `mongodb://localhost:27017`)
- `MONGO_DATABASE` (default `scheduler`)
- `NATS_URL` (default `nats://localhost:4222`)
- `LOG_LEVEL` (default `info`; supported values include `trace`, `debug`, `info`, `warn`, `error`, `fatal`, and `panic`)

Logs are written to stdout in a human-readable console format. For example, use `LOG_LEVEL=debug go run ./cmd/server` to enable debug-level logging.

### API

Create a job with a standard five-field cron expression:

```sh
curl -X POST http://localhost:8080/jobs -H "Content-Type: application/json" -d '{
	"name": "nightly sync",
	"schedule": "0 2 * * *",
	"webhook_url": "https://example.test/hooks/sync",
	"parameters": {"tenant": "acme"},
	"payload": {"full": true}
}'
```

Endpoints:

- `GET /jobs` lists every job.
- `POST /jobs` creates a job.
- `GET /jobs/:id` gets one job without its result.
- `PUT /jobs/:id` replaces a job's schedule and webhook configuration.
- `DELETE /jobs/:id` deletes a job and removes its cron schedule.
- `GET /jobs/:id/results` gets only the latest result. It returns `404` when the job has not executed yet.

Each webhook receives `{ "parameters": {}, "payload": {} }` as JSON. After the result is saved, the service publishes the result JSON on `scheduler.job.executed`.

### Go client

The `client` package provides a typed Resty client for the scheduler API:

```go
schedulerClient := client.New("http://localhost:8080")
job, err := schedulerClient.CreateJob(scheduler.CreateJobRequest{
	Name: "nightly sync",
	Schedule: "0 2 * * *",
	WebhookURL: "https://example.test/hooks/sync",
})
if err != nil {
	// Handle the request or API error.
}

result, err := schedulerClient.GetJobResult(job.ID)
```

### CLI

Build the command-line client with `go build -o scheduler-client ./cmd/client`, then use the service URL before the command:

```sh
scheduler-client -server http://localhost:8080 list
scheduler-client -server http://localhost:8080 create -name "nightly sync" -schedule "0 2 * * *" -webhook-url "https://example.test/hooks/sync" -parameters '{"tenant":"acme"}' -payload '{"full":true}'
scheduler-client -server http://localhost:8080 get JOB_ID
scheduler-client -server http://localhost:8080 update JOB_ID -name "updated sync" -schedule "0 3 * * *" -webhook-url "https://example.test/hooks/sync"
scheduler-client -server http://localhost:8080 result JOB_ID
scheduler-client -server http://localhost:8080 delete JOB_ID
```

The CLI prints successful responses as indented JSON. `create` and `update` accept `-parameters` and `-payload` as JSON objects.
