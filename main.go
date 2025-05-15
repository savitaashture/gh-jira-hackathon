package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/savitaashture/gh-jira/pkg/summarizer"
	"golang.org/x/oauth2"
)

var (
	githubOwner       = os.Getenv("GH_OWNER")
	githubRepo        = os.Getenv("GH_REPO")
	githubToken       = os.Getenv("GH_TOKEN")
	jiraUsername      = os.Getenv("JIRA_USERNAME")
	jiraAPIToken      = os.Getenv("JIRA_API_TOKEN")
	jiraBaseURL       = os.Getenv("JIRA_BASE_URL")
	jiraProjectKey    = "GT"
	jiraIssueType     = "Task"
	processedIssueIDs = make(map[int64]bool)
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.Printf("Starting application with configuration:")
	log.Printf("GitHub Owner: %s", githubOwner)
	log.Printf("GitHub Repo: %s", githubRepo)
	log.Printf("Jira Base URL: %s", jiraBaseURL)
	log.Printf("Jira Project Key: %s", jiraProjectKey)
	log.Printf("Jira Issue Type: %s", jiraIssueType)
}

func main() {
	log.Printf("Initializing Ollama summarizer with mistral model")
	var err error
	sum, err := summarizer.New(summarizer.Config{
		Model: "mistral", // Using mistral model
	})
	if err != nil {
		log.Fatalf("Failed to initialize summarizer: %v", err)
	}
	log.Printf("Summarizer initialized successfully")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	log.Printf("Starting initial GitHub poll")
	pollGitHub(sum)

	log.Printf("Entering main polling loop")
	for range ticker.C {
		log.Printf("Polling GitHub for new issues")
		pollGitHub(sum)
	}
}

func pollGitHub(sum *summarizer.Summarizer) {
	log.Printf("Creating GitHub client")
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	log.Printf("Fetching open issues from GitHub")
	issues, _, err := client.Issues.ListByRepo(ctx, githubOwner, githubRepo, &github.IssueListByRepoOptions{
		State: "open",
		Sort:  "created",
	})
	if err != nil {
		log.Printf("Error fetching GitHub issues: %v", err)
		return
	}
	log.Printf("Found %d issues", len(issues))

	for _, issue := range issues {
		if issue.IsPullRequest() {
			log.Printf("Skipping PR #%d", *issue.Number)
			break
		}

		log.Printf("Processing issue #%d: %s", *issue.Number, *issue.Title)

		// Create a context with a longer timeout for model generation
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		log.Printf("Starting summary generation for issue #%d", *issue.Number)

		// Generate the summary
		summary, err := sum.SummarizeWithCustomPrompt(ctx, *issue.Body, fmt.Sprintf(`Please analyze this GitHub issue description and create a clear, concise summary with necessary code snippet:

%s

Please format the response as follows:
1. Brief overview (1-2 sentences)
2. Key points (bullet points)
3. Technical details (if any)
4. Impact and dependencies (if mentioned)`, *issue.Body))

		// Cancel the context after we're done with the API call
		cancel()

		if err != nil {
			log.Printf("Failed to generate summary for issue #%d: %v", *issue.Number, err)
			continue
		}
		log.Printf("Successfully generated summary for issue #%d", *issue.Number)

		if !processedIssueIDs[*issue.ID] {
			log.Printf("New GitHub issue detected: #%d - %s", *issue.Number, *issue.Title)
			log.Printf("Creating Jira issue for GitHub issue #%d", *issue.Number)
			err := createJiraIssue(issue, summary)
			if err == nil {
				log.Printf("Successfully created Jira issue for GitHub issue #%d", *issue.Number)
				processedIssueIDs[*issue.ID] = true
			} else {
				log.Printf("Failed to create Jira issue for GitHub issue #%d: %v", *issue.Number, err)
			}
		} else {
			log.Printf("Issue #%d already processed, skipping", *issue.Number)
		}
	}
	log.Printf("Finished processing all issues")
}

func createJiraIssue(issue *github.Issue, summary string) error {
	log.Printf("Preparing Jira issue payload for GitHub issue #%d", *issue.Number)
	jiraURL := fmt.Sprintf("%s/rest/api/2/issue", jiraBaseURL)

	payload := map[string]interface{}{
		"fields": map[string]interface{}{
			"project": map[string]string{
				"key": jiraProjectKey,
			},
			"summary":     fmt.Sprintf("GitHub Issue #%d: %s", *issue.Number, *issue.Title),
			"description": fmt.Sprintf("Imported from GitHub: %s\n\nSummarized Description:\n%s", *issue.HTMLURL, summary),
			"issuetype": map[string]string{
				"name": jiraIssueType,
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Failed to marshal Jira payload for issue #%d: %v", *issue.Number, err)
		return err
	}
	log.Printf("Jira payload prepared for issue #%d", *issue.Number)

	req, err := http.NewRequest("POST", jiraURL, strings.NewReader(string(jsonData)))
	if err != nil {
		log.Printf("Failed to create HTTP request for issue #%d: %v", *issue.Number, err)
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(jiraUsername, jiraAPIToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "jira-client/1.0")

	log.Printf("Sending request to Jira API for issue #%d", *issue.Number)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request failed for issue #%d: %v", *issue.Number, err)
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	log.Printf("Jira API response for issue #%d - Status: %s, Body: %s", *issue.Number, resp.Status, string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("Jira issue created successfully for GitHub issue #%d", *issue.Number)
		return nil
	}

	err = fmt.Errorf("Jira API responded with status %s", resp.Status)
	log.Printf("Failed to create Jira issue for GitHub issue #%d: %v", *issue.Number, err)
	return err
}
