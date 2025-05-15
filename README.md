# gh-jira-hackathon

A GitHub to Jira synchronization system that enhances issue tracking by automatically creating and updating Jira issues from GitHub issues, complete with AI-powered summarization.

## Features

- Automatic synchronization of GitHub issues to Jira
- AI-powered summarization of GitHub issue descriptions using Ollama
- Real-time logging of summary generation
- Support for custom prompt templates
- Error handling and recovery mechanisms

## Prerequisites

- Go 1.x or higher
- [Ollama](https://ollama.ai/) installed and running locally
- Access to both GitHub and Jira APIs
- The Mistral model pulled in Ollama (default model)

## Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/waveywaves/gh-jira-hackathon.git
   cd gh-jira-hackathon
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Configure environment variables:
   ```bash
   export GITHUB_TOKEN="your-github-token"
   export JIRA_URL="your-jira-url"
   export JIRA_USERNAME="your-jira-username"
   export JIRA_TOKEN="your-jira-api-token"
   ```

## Usage

The system provides two main summarization methods:

1. Basic Summarization:
   ```go
   summarizer, err := summarizer.New(summarizer.Config{
       Model: "mistral", // Optional, defaults to mistral
   })
   
   summary, err := summarizer.SummarizeChanges(context.Background(), issueDescription)
   ```

2. Custom Prompt Summarization:
   ```go
   summary, err := summarizer.SummarizeWithCustomPrompt(
       context.Background(),
       content,
       customPromptTemplate,
   )
   ```

## Logging

The system provides detailed logging of the summarization process:

- Initialization logs for the summarizer and Ollama client
- Real-time generation logs showing the content being generated
- Error logs for any issues during generation
- Summary completion logs with the final summary length

## Error Handling

The system includes robust error handling for:
- Channel management to prevent "close of closed channel" panics
- Context cancellation for timeouts and interruptions
- API errors from both Ollama and external services
- Invalid configurations or missing dependencies

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

[Add your license information here]