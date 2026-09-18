package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	scheduler "github.com/Antonoir1/scheduler"
	"github.com/Antonoir1/scheduler/client"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	global := flag.NewFlagSet("scheduler-client", flag.ContinueOnError)
	global.SetOutput(os.Stderr)
	server := global.String("server", "http://localhost:8080", "scheduler service URL")
	if err := global.Parse(args); err != nil {
		return 2
	}
	if global.NArg() == 0 {
		printUsage()
		return 2
	}

	api := client.New(*server)
	command := global.Arg(0)
	commandArgs := global.Args()[1:]
	var value any
	var err error

	switch command {
	case "list":
		value, err = api.ListJobs()
	case "create":
		value, err = createJob(api, commandArgs)
	case "get":
		value, err = getJob(api, commandArgs)
	case "update":
		value, err = updateJob(api, commandArgs)
	case "delete":
		err = deleteJob(api, commandArgs)
		if err == nil {
			fmt.Println("job deleted")
			return 0
		}
	case "result":
		value, err = getResult(api, commandArgs)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", command)
		printUsage()
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := printJSON(value); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func createJob(api *client.Client, args []string) (*scheduler.Job, error) {
	request, err := parseJobRequest("create", args)
	if err != nil {
		return nil, err
	}
	return api.CreateJob(request)
}

func updateJob(api *client.Client, args []string) (*scheduler.Job, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("update requires a job ID")
	}
	request, err := parseJobRequest("update", args[1:])
	if err != nil {
		return nil, err
	}
	return api.UpdateJob(args[0], request)
}

func getJob(api *client.Client, args []string) (*scheduler.Job, error) {
	id, err := requireID("get", args)
	if err != nil {
		return nil, err
	}
	return api.GetJob(id)
}

func deleteJob(api *client.Client, args []string) error {
	id, err := requireID("delete", args)
	if err != nil {
		return err
	}
	return api.DeleteJob(id)
}

func getResult(api *client.Client, args []string) (*scheduler.JobResult, error) {
	id, err := requireID("result", args)
	if err != nil {
		return nil, err
	}
	return api.GetJobResult(id)
}

func parseJobRequest(name string, args []string) (scheduler.CreateJobRequest, error) {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	jobName := flags.String("name", "", "job name")
	schedule := flags.String("schedule", "", "five-field cron expression")
	webhookURL := flags.String("webhook-url", "", "webhook URL")
	parameters := flags.String("parameters", "{}", "parameters as a JSON object")
	headers := flags.String("headers", "{}", "webhook headers as a JSON object")
	payload := flags.String("payload", "", "raw webhook request body")
	if err := flags.Parse(args); err != nil {
		return scheduler.CreateJobRequest{}, err
	}
	if *jobName == "" || *schedule == "" || *webhookURL == "" {
		return scheduler.CreateJobRequest{}, fmt.Errorf("%s requires -name, -schedule, and -webhook-url", name)
	}
	parsedParameters, err := parseStringMap(*parameters, "parameters")
	if err != nil {
		return scheduler.CreateJobRequest{}, err
	}
	parsedHeaders, err := parseStringMap(*headers, "headers")
	if err != nil {
		return scheduler.CreateJobRequest{}, err
	}
	return scheduler.CreateJobRequest{Name: *jobName, Schedule: *schedule, WebhookURL: *webhookURL, Parameters: parsedParameters, Headers: parsedHeaders, Payload: *payload}, nil
}

func requireID(command string, args []string) (string, error) {
	if len(args) != 1 || args[0] == "" {
		return "", fmt.Errorf("%s requires exactly one job ID", command)
	}
	return args[0], nil
}

func parseStringMap(value, name string) (map[string]string, error) {
	result := make(map[string]string)
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, fmt.Errorf("invalid %s JSON: %w", name, err)
	}
	return result, nil
}

func printJSON(value any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: scheduler-client [-server URL] <command> [flags]")
	fmt.Fprintln(os.Stderr, "Commands: list, create, get, update, delete, result")
}
