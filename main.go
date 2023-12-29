package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type GitHubDispatchEvent struct {
	EventType     string `json:"event_type"`
	ClientPayload struct {
		InstanceID     string `json:"instance_id"`
		AppID          string `json:"app_id"`
		AppdID         string `json:"appd_id"`
		Environment    string `json:"environment"`
		InstanceClass  string `json:"instance_class"`
		SizeStorage    string `json:"size_storage"`
		DbName         string `json:"db_name"`
		PgMajorVersion string `json:"pg_major_version"`
		Collation      string `json:"collation"`
		Encoding       string `json:"encoding"`
	} `json:"client_payload"`
}

func HandleRequest(ctx context.Context, s3Event events.S3Event) (string, error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1")}, // Vervang door je regio
	)
	s3svc := s3.New(sess)

	for _, record := range s3Event.Records {
		s3Entity := record.S3
		bucket := s3Entity.Bucket.Name
		key := s3Entity.Object.Key

		// Lees het JSON-bestand van S3
		result, err := s3svc.GetObject(&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return "", err
		}
		defer result.Body.Close()
		body, err := ioutil.ReadAll(result.Body)
		if err != nil {
			return "", err
		}

		// Parse de JSON-data
		var payloadData struct {
			InstanceID     string `json:"instance_id"`
			AppID          string `json:"app_id"`
			AppdID         string `json:"appd_id"`
			Environment    string `json:"environment"`
			InstanceClass  string `json:"instance_class"`
			SizeStorage    string `json:"size_storage"`
			DbName         string `json:"db_name"`
			PgMajorVersion string `json:"pg_major_version"`
			Collation      string `json:"collation"`
			Encoding       string `json:"encoding"`
		}
		if err := json.Unmarshal(body, &payloadData); err != nil {
			return "", err
		}

		// Stel het dispatch event samen
		dispatchEvent := GitHubDispatchEvent{
			EventType: "trigger-workflow",
			ClientPayload: struct {
				InstanceID     string `json:"instance_id"`
				AppID          string `json:"app_id"`
				AppdID         string `json:"appd_id"`
				Environment    string `json:"environment"`
				InstanceClass  string `json:"instance_class"`
				SizeStorage    string `json:"size_storage"`
				DbName         string `json:"db_name"`
				PgMajorVersion string `json:"pg_major_version"`
				Collation      string `json:"collation"`
				Encoding       string `json:"encoding"`
			}{
				InstanceID:     payloadData.InstanceID,
				AppID:          payloadData.AppID,
				AppdID:         payloadData.AppdID,
				Environment:    payloadData.Environment,
				InstanceClass:  payloadData.InstanceClass,
				SizeStorage:    payloadData.SizeStorage,
				DbName:         payloadData.DbName,
				PgMajorVersion: payloadData.PgMajorVersion,
				Collation:      payloadData.Collation,
				Encoding:       payloadData.Encoding,
			},
		}

		requestBody, err := json.Marshal(dispatchEvent)
		if err != nil {
			return "", err
		}

		// Verstuur het event naar GitHub
		githubToken := os.Getenv("GITHUB_TOKEN")
		repoOwner := os.Getenv("REPO_OWNER")
		repoName := os.Getenv("REPO_NAME")
		url := "https://api.github.com/repos/" + repoOwner + "/" + repoName + "/dispatches"

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(requestBody))
		if err != nil {
			return "", err
		}

		req.Header.Add("Authorization", "token "+githubToken)
		req.Header.Add("Accept", "application/vnd.github.v3+json")
		req.Header.Add("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		responseBody, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode >= 400 {
			return "", fmt.Errorf("GitHub API responded with status code: %d, body: %s", resp.StatusCode, string(responseBody))
		}

		// Verplaats het bestand naar de 'archive'-map
		archiveKey := strings.Replace(key, "process/", "archive/", 1)
		_, err = s3svc.CopyObject(&s3.CopyObjectInput{
			Bucket:     aws.String(bucket),
			CopySource: aws.String(bucket + "/" + key),
			Key:        aws.String(archiveKey),
		})
		if err != nil {
			return "", err
		}

		// Verwijder het originele bestand
		_, err = s3svc.DeleteObject(&s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		})
		if err != nil {
			return "", err
		}
	}

	return "Verwerking voltooid", nil
}

func main() {
	lambda.Start(HandleRequest)
}
