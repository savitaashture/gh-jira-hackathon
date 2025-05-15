package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/savitaashture/gh-jira/pkg/summarizer"
	"golang.org/x/oauth2"
)

var (
	githubOwner       = "savitaashture"
	githubRepo        = "gh-jira-hackathon"
	githubToken       = ""
	jiraBaseURL       = "https://abhighosh3108.atlassian.net"
	jiraUsername      = "abhi.ghosh3108@gmail.com"
	jiraAPIToken      = ""
	jiraProjectKey    = "GT"
	jiraIssueType     = "Task"
	processedIssueIDs = make(map[int64]bool)
)

func main() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	sum, err := summarizer.New(summarizer.Config{
		Model: "mistral", // Using mistral model
	})
	if err != nil {
		log.Fatalf("Failed to create summarizer: %v", err)
	}

	// Initial fetch on startup
	go pollGitHub(sum)

	for range ticker.C {
		pollGitHub()
	}
}

func pollGitHub() {
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	issues, _, err := client.Issues.ListByRepo(ctx, githubOwner, githubRepo, &github.IssueListByRepoOptions{
		State: "open",
		Sort:  "created",
	})
	if err != nil {
		log.Printf("Error fetching GitHub issues: %v", err)
		return
	}

	for _, issue := range issues {
		if issue.IsPullRequest() {
			break
		}

		// Create a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Generate the summary
		summary, err := sum.SummarizeWithCustomPrompt(ctx, changes, fmt.Sprinf("Summarize the following changes in a release note: %s %", *issue.Body))
		if err != nil {
			log.Fatalf("Failed to generate summary: %v", err)
		}

		if !processedIssueIDs[*issue.ID] {
			log.Printf("New GitHub issue detected: #%d - %s", *issue.Number, *issue.Title)
			err := createJiraIssue(issue, summary)
			if err == nil {
				processedIssueIDs[*issue.ID] = true
			} else {
				log.Printf("Failed to create Jira issue: %v", err)
			}
		}
	}
}

func createJiraIssue(issue *github.Issue, summary string) error {
	jiraURL := fmt.Sprintf("%s/rest/api/2/issue", jiraBaseURL)

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": jiraProjectKey,
			},
			"summary":     fmt.Sprintf("GitHub Issue #%d: %s", *issue.Number, *issue.Title),
			"description": fmt.Sprintf("Imported from GitHub: %s\n\n%s", *issue.HTMLURL, *issue.Body),
			"issuetype": map[string]string{
				"name": jiraIssueType,
			},
		},
	}

	jsonData, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", jiraURL, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(jiraUsername, jiraAPIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "jira-client/1.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Jira issue created successfully for GitHub issue #%d", *issue.Number)
		return nil
	}

	log.Printf("Jira Response Body: %s", string(body))
	return fmt.Errorf("Jira API responded with status %s", resp.Status)
}
