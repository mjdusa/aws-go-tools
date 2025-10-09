package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudformation"
	cfTypes "github.com/aws/aws-sdk-go-v2/service/cloudformation/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// getCallerIdentity retrieves the AWS account ID and user ID.
func getCallerIdentity(ctx context.Context, cfg aws.Config) (*sts.GetCallerIdentityOutput, error) {
	stsClient := sts.NewFromConfig(cfg)

	input := sts.GetCallerIdentityInput{}

	output, err := stsClient.GetCallerIdentity(ctx, &input)
	if err != nil {
		log.Fatalf("Unable to get caller identity: %v", err)
		return nil, fmt.Errorf("Unable to get caller identity: %w", err)
	}

	return output, nil
}

// getAWSRegions retrieves a list of all AWS regions.
func getAWSRegions(ctx context.Context, cfg aws.Config, allRegions bool) (*[]ec2Types.Region, error) {
	ec2Client := ec2.NewFromConfig(cfg)

	input := &ec2.DescribeRegionsInput{
		AllRegions: aws.Bool(allRegions),
	}

	output, err := ec2Client.DescribeRegions(ctx, input)
	if err != nil {
		log.Fatalf("Unable to DescribeRegions: %v", err)
		return nil, fmt.Errorf("failed to describe regions: %w", err)
	}

	return &output.Regions, nil
}

// cloudFormationListStacks retrieves a list of CloudFormation stacks.
func cloudFormationListStacks(ctx context.Context, cfg aws.Config) (*[]cfTypes.StackSummary, error) {
	cfClient := cloudformation.NewFromConfig(cfg)

	var allStacks []cfTypes.StackSummary
	var nextToken *string

	for {
		input := cloudformation.ListStacksInput{
			NextToken: nextToken, // Use the token to fetch the next page
			StackStatusFilter: []cfTypes.StackStatus{
				cfTypes.StackStatusCreateInProgress,
				cfTypes.StackStatusCreateFailed,
				cfTypes.StackStatusCreateComplete,

				cfTypes.StackStatusRollbackInProgress,
				cfTypes.StackStatusRollbackFailed,
				cfTypes.StackStatusRollbackComplete,

				cfTypes.StackStatusDeleteInProgress,
				cfTypes.StackStatusDeleteFailed,
				// cfTypes.StackStatusDeleteComplete,

				cfTypes.StackStatusUpdateInProgress,
				cfTypes.StackStatusUpdateFailed,
				cfTypes.StackStatusUpdateComplete,

				cfTypes.StackStatusUpdateRollbackInProgress,
				cfTypes.StackStatusUpdateRollbackFailed,
				cfTypes.StackStatusUpdateRollbackCompleteCleanupInProgress,
				cfTypes.StackStatusUpdateRollbackComplete,

				cfTypes.StackStatusReviewInProgress,

				cfTypes.StackStatusImportInProgress,
				cfTypes.StackStatusImportComplete,

				cfTypes.StackStatusImportRollbackInProgress,
				cfTypes.StackStatusImportRollbackFailed,
				cfTypes.StackStatusImportRollbackComplete,
			},
		}

		output, err := cfClient.ListStacks(ctx, &input)
		if err != nil {
			log.Fatalf("Unable to ListStacks: %v", err)
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		// Append the current page of stacks to the result
		allStacks = append(allStacks, output.StackSummaries...)

		// Check if there is another page
		if output.NextToken == nil {
			break
		}

		// Set the next token for the next iteration
		nextToken = output.NextToken
	}

	return &allStacks, nil
}

// cloudFormationListStackResources retrieves a list of CloudFormation stack resources.
func cloudFormationListStackResources(ctx context.Context, cfg aws.Config, stackID string) (*[]cfTypes.StackResourceSummary, error) {
	cfClient := cloudformation.NewFromConfig(cfg)

	var allStackResources []cfTypes.StackResourceSummary
	var nextToken *string

	for {
		input := cloudformation.ListStackResourcesInput{
			StackName: aws.String(stackID),
			NextToken: nextToken, // Use the token to fetch the next page
		}

		output, err := cfClient.ListStackResources(ctx, &input)
		if err != nil {
			log.Fatalf("Unable to ListStacks: %v", err)
			return nil, fmt.Errorf("failed to list stacks: %w", err)
		}

		// Append the current page of stacks to the result
		allStackResources = append(allStackResources, output.StackResourceSummaries...)

		// Check if there is another page
		if output.NextToken == nil {
			break
		}

		// Set the next token for the next iteration
		nextToken = output.NextToken
	}

	return &allStackResources, nil
}

func main() {
	verbose := true
	ctx := context.Background()

	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-west-2"
	}

	// Load AWS configuration.
	cfg, cerr := config.LoadDefaultConfig(ctx)
	if cerr != nil {
		log.Fatalf("Unable to load AWS configuration: %v", cerr)
		return
	}

	identity, ierr := getCallerIdentity(ctx, cfg)
	if ierr != nil {
		log.Fatalf("Unable to load AWS Caller Identity: %v", ierr)
		return
	}

	if verbose {
		log.Printf("AWS Account ID: %s\n", *identity.Account)
		log.Printf("AWS User ID: %s\n", *identity.UserId)
		log.Printf("AWS ARN: %s\n", *identity.Arn)
		log.Println()
	}

	regions, rerr := getAWSRegions(ctx, cfg, false)
	if rerr != nil {
		log.Fatalf("Unable to load AWS Regions: %v", rerr)
		return
	}

	allRegionNames := []string{region} // Add more regions if needed

	for _, region := range *regions {
		if region.RegionName != nil {
			if verbose {
				log.Printf("Adding region '%s'\n", *region.RegionName)
			}

			allRegionNames = append(allRegionNames, *region.RegionName)
		}
	}

	log.Printf("All detected AWS Regions: %v\n", allRegionNames)

	log.Println("Checking each region for stacks...")

	for _, regionName := range allRegionNames {
		log.Printf("- Region: %s\n", regionName)

		stacks, serr := cloudFormationListStacks(ctx, cfg)
		if serr != nil {
			log.Printf("Error calling cloudFormationListStacks: %v", serr)
			continue
		}

		for _, stack := range *stacks {
			if verbose {
				log.Println("- Stack:")
				log.Printf("  - Id: %s", NilSafeString(stack.StackId))
				log.Printf("  - Name: %s", NilSafeString(stack.StackName))
				log.Printf("  - Status: %s", stack.StackStatus)
				log.Printf("  - Status Reason: %s", NilSafeString(stack.StackStatusReason))
				log.Printf("  - Parent Id: %s", NilSafeString(stack.ParentId))
				log.Printf("  - Root Id: %s", NilSafeString(stack.RootId))
				log.Printf("  - Creation Time: %s", NilSafeTime(stack.CreationTime, ""))
				log.Printf("  - Last Updated Time: %s", NilSafeTime(stack.LastUpdatedTime, ""))
				log.Printf("  - Deletion Time: %s", NilSafeTime(stack.DeletionTime, ""))
			}

			stackResources, srerr := cloudFormationListStackResources(ctx, cfg, *stack.StackId)
			if srerr != nil {
				log.Printf("Error calling cloudFormationListStackResources: %v", srerr)
				continue
			}

			for _, stackResource := range *stackResources {
				if verbose {
					log.Println("  - Stack Resource:")
					log.Printf("     - Physical Resource Id: %s", NilSafeString(stackResource.PhysicalResourceId))
					log.Printf("     - Logical Resource Id: %s", NilSafeString(stackResource.LogicalResourceId))
					log.Printf("     - Resource Type: %s", NilSafeString(stackResource.ResourceType))
					log.Printf("     - Status: %s", stackResource.ResourceStatus)
					log.Printf("     - Status Reason: %s", NilSafeString(stackResource.ResourceStatusReason))
					log.Printf("     - Last Updated Time: %s", NilSafeTime(stackResource.LastUpdatedTimestamp, ""))
				}
			}
		}

		log.Println("")
	}
}

func NilSafeString(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}

func NilSafeTime(tmp *time.Time, fmt string) string {
	if tmp == nil {
		return "<nil>"
	}

	if fmt == "" {
		fmt = time.RFC3339
	}

	return tmp.Format(fmt)
}
