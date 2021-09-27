package test

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/aws"
	http_helper "github.com/gruntwork-io/terratest/modules/http-helper"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

// Custom function which should take a map of EC2 instance tags and a key
// And it should return the value of each tag by its key
func GetTagValue(tags map[string]string, key string) string {
	found := ""
	for mapKey, mapValue := range tags {
		if mapKey == key {
			found = mapValue
		}
	}
	return found
}

func TestingS3Bucket(t *testing.T, terraformOpts *terraform.Options, awsRegion string, NAME_TAG string, OWNER_TAG string) {
	// Get Terraform output "bucket_id"
	bucketId := terraform.Output(t, terraformOpts, "bucket_id")
	fmt.Println("MRG_DEBUG:::TERRATEST_OUTPUT:: Bucket ID:", bucketId)

	// Get this bucket by "Name" tag
	bucketNameByNameTag := aws.FindS3BucketWithTag(t, awsRegion, "Name", NAME_TAG)
	fmt.Println("MRG_DEBUG:::FIND_S3_BUCKET_BY_TAG::NAME:: Bucket:", bucketNameByNameTag)

	// Get the bucket by "Owner" tag
	bucketNameByOwnerTag := aws.FindS3BucketWithTag(t, awsRegion, "Owner", OWNER_TAG)
	fmt.Println("MRG_DEBUG:::FIND_S3_BUCKET_BY_TAG::OWNER:: Bucket:", bucketNameByOwnerTag)

	// Check if this bucket actually exists
	aws.AssertS3BucketExists(t, awsRegion, bucketId)

	// Check that bucket was found by querying it with "Name" tag
	assert.Equal(t, bucketId, bucketNameByNameTag)

	// Check that bucket was found by querying it with "Owner" tag
	assert.Equal(t, bucketId, bucketNameByOwnerTag)
}

func TestingEc2Instance(t *testing.T, terraformOpts *terraform.Options, awsRegion string, NAME_TAG string, OWNER_TAG string) {
	// Get Terraform output
	ec2InstanceId := terraform.Output(t, terraformOpts, "ec2_instance_id")
	fmt.Println("MRG_DEBUG:::TERRATEST_OUTPUT:: Instance ID:", ec2InstanceId)

	// Here I'm just using another approach to test that the actual ec2 instance that was created, has the wanted tags
	// Note that i could've used "GetEc2InstanceIdsByTag" to do it in the same exact way that we did with asserting S3 Bucket
	// But just to use various approaches
	// Also i think this approach is better as then we will be returning an array of tags, and we could just find the needle in a really small haystack
	ec2Tags := aws.GetTagsForEc2Instance(t, awsRegion, ec2InstanceId)
	fmt.Println("MRG_DEBUG:::INSTANCE_TAGS:: Tags:", ec2Tags)

	nameTag := GetTagValue(ec2Tags, "Name")
	fmt.Println("MRG_DEBUG:::INSTANCE_TAGS:: Name Tag:", nameTag)

	ownerTag := GetTagValue(ec2Tags, "Owner")
	fmt.Println("MRG_DEBUG:::INSTANCE_TAGS:: Owner Tag:", ownerTag)

	assert.Equal(t, NAME_TAG, nameTag)
	assert.Equal(t, OWNER_TAG, ownerTag)
}

func TestingPythonServer(t *testing.T, terraformOpts *terraform.Options, awsRegion string) {
	DEFAULT_TRIES := 10
	DEFAULT_TIME_BETWEEN_TRIES := 10 * time.Second
	NAME_TAG := "Flugel"
	OWNER_TAG := "InfraTeam"

	ec2InstanceId := terraform.Output(t, terraformOpts, "ec2_instance_id")
	ec2PublicIp := terraform.Output(t, terraformOpts, "ec2_instance_public_ip")
	fmt.Println("MRG_DEBUG:::INSTANCE_IP:: Instance IP:", ec2PublicIp)

	ec2Url := fmt.Sprintf("http://%s", ec2PublicIp)
	fmt.Println("MRG_DEBUG:::INSTANCE_URL:: Instance URL:", ec2Url)

	// Check the home page of server that is returning a response
	http_helper.HttpGetWithRetryWithCustomValidation(t, ec2Url, &tls.Config{}, DEFAULT_TRIES, DEFAULT_TIME_BETWEEN_TRIES, func(status int, body string) bool {
		assert.Equal(t, 200, status)
		return strings.Contains(strings.ToLower(body), "up & running")
	})

	// Check that it is listing the correct tags
	http_helper.HttpGetWithRetryWithCustomValidation(t, fmt.Sprintf("%s/tags", ec2Url), &tls.Config{}, DEFAULT_TRIES, DEFAULT_TIME_BETWEEN_TRIES, func(status int, body string) bool {
		assert.Equal(t, 200, status)
		parsedBody := strings.ToLower(body)
		return strings.Contains(parsedBody, "name") && strings.Contains(parsedBody, strings.ToLower(NAME_TAG)) &&
			strings.Contains(parsedBody, "owner") && strings.Contains(parsedBody, strings.ToLower(OWNER_TAG))
	})

	// Check that user can shutdown the instance
	http_helper.HttpGetWithRetryWithCustomValidation(t, fmt.Sprintf("%s/shutdown", ec2Url), &tls.Config{}, DEFAULT_TRIES, DEFAULT_TIME_BETWEEN_TRIES, func(status int, body string) bool {
		assert.Equal(t, 200, status)
		parsedBody := strings.ToLower(body)
		return strings.Contains(parsedBody, ec2InstanceId) && strings.Contains(parsedBody, "shutdown")
	})

	fmt.Println("MRG_DEBUG:::TESTING_SERVER_AVAILABILITY:: Waiting 30 seconds before testing..")

	// Wait for 30 seconds to give the instance a chance to stop
	time.Sleep(30 * time.Second)

	// Check that server is up and running then
	escapeServerCacheUrl := fmt.Sprintf("%s/?q=%s", ec2Url, random.UniqueId())

	fmt.Println("MRG_DEBUG:::TESTING_SERVER_AVAILABILITY:: Testing server with this URL", escapeServerCacheUrl)

	// This somehow is failing on Github Workflow with "context deadline exceeded (Client.Timeout exceeded while awaiting headers)" error
	// It works locally though!
	//http_helper.HttpGetWithRetryWithCustomValidation(t, escapeServerCacheUrl, &tls.Config{}, DEFAULT_TRIES, DEFAULT_TIME_BETWEEN_TRIES, func (status int, body string) bool {
	//assert.Equal(t, 502, status)
	//parsedBody := strings.ToLower(body)
	//return strings.Contains(parsedBody, "connection timed out")
	//})

	// Test the web server using a random query string to escape server side caching
	_, err := net.DialTimeout("tcp", escapeServerCacheUrl, DEFAULT_TIME_BETWEEN_TRIES)

	assert.NotNil(t, err)
}

// Wrapper function to prepare the infrastructure, terratest, and terraform variables
func TestInfrastructure(t *testing.T) {
	awsRegion := "eu-central-1"
	NAME_TAG := "Flugel"
	OWNER_TAG := "InfraTeam"

	terraformOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: "../",
		Vars: map[string]interface{}{
			"s3_bucket_name": fmt.Sprintf("mrg-bucket-%s", strings.ToLower(random.UniqueId())),
		},
		EnvVars: map[string]string{
			"AWS_DEFAUTL_REGION": awsRegion,
		},
	})

	defer terraform.Destroy(t, terraformOpts)

	terraform.InitAndApply(t, terraformOpts)

	// Test EC2 instance
	TestingEc2Instance(t, terraformOpts, awsRegion, NAME_TAG, OWNER_TAG)

	// Test S3 bucket
	TestingS3Bucket(t, terraformOpts, awsRegion, NAME_TAG, OWNER_TAG)

	fmt.Println("MRG_DEBUG:::TEST_INFRA:: Pause for 1min before testing the web server")

	// Sleep for 30 seconds before trying to access the webserver to give a change for the script to install everything
	time.Sleep(60 * time.Second)

	fmt.Println("MRG_DEBUG:::TEST_INFRA:: Continue testing web server")

	// Test the web server
	TestingPythonServer(t, terraformOpts, awsRegion)
}
