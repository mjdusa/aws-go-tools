package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

const (
	UnixTimeFactor = 1000
)

func getECSClient(ctx context.Context, region string) (*ecs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %w", err)
	}

	return ecs.NewFromConfig(cfg), nil
}

func getCloudWatchLogsClient(ctx context.Context, region string) (*cloudwatchlogs.Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %w", err)
	}

	return cloudwatchlogs.NewFromConfig(cfg), nil
}

func getTaskLogStreamName(ctx context.Context, ecsClient *ecs.Client, cluster string, taskID string) (string, error) {
	resp, err := ecsClient.DescribeTasks(ctx, &ecs.DescribeTasksInput{
		Cluster: aws.String(cluster),
		Tasks:   []string{taskID},
	})
	if err != nil {
		return "", fmt.Errorf("failed to describe tasks: %w", err)
	}

	if len(resp.Tasks) == 0 {
		return "", fmt.Errorf("task not found")
	}

	task := resp.Tasks[0]
	if len(task.Containers) == 0 {
		return "", fmt.Errorf("no containers found in task")
	}

	container := task.Containers[0]
	return *container.Name, nil
}

func getLogEvents(ctx context.Context, cwLogsClient *cloudwatchlogs.Client, logGroupName string, logStreamName string) error {
	endTime := time.Now()
	startTime := endTime.Add(-1 * time.Hour)

	input := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
		StartTime:     aws.Int64(startTime.Unix() * UnixTimeFactor),
		EndTime:       aws.Int64(endTime.Unix() * UnixTimeFactor),
	}

	paginator := cloudwatchlogs.NewGetLogEventsPaginator(cwLogsClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("failed to get log events: %w", err)
		}

		for _, event := range page.Events {
			fmt.Printf("%s\t%s\n", time.UnixMilli(*event.Timestamp).String(), *event.Message)
		}
	}

	return nil
}

func main() {
	ctx := context.Background()

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-2"
	}

	cluster := os.Getenv("ECS_CLUSTER")
	if cluster == "" {
		panic("ECS_CLUSTER environment variable is required")
	}

	taskID := os.Getenv("ECS_TASK_ID")
	if taskID == "" {
		panic("ECS_TASK_ID environment variable is required")
	}

	logGroupName := os.Getenv("LOG_GROUP_NAME")
	if logGroupName == "" {
		panic("LOG_GROUP_NAME environment variable is required")
	}

	ecsClient, err := getECSClient(ctx, region)
	if err != nil {
		log.Fatalf("failed to create ECS client: %v", err)
	}

	cwLogsClient, err := getCloudWatchLogsClient(ctx, region)
	if err != nil {
		log.Fatalf("failed to create CloudWatch Logs client: %v", err)
	}

	logStreamName, err := getTaskLogStreamName(ctx, ecsClient, cluster, taskID)
	if err != nil {
		log.Fatalf("failed to get log stream name: %v", err)
	}

	fmt.Printf("Log Stream Name: %s\n", logStreamName)

	err = getLogEvents(ctx, cwLogsClient, logGroupName, logStreamName)
	if err != nil {
		log.Fatalf("failed to get log events: %v", err)
	}
}
