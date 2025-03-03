// Package ollama contains the structs and functions for working with the Ollama API.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// ModelOptions struct contains the Ollama options for the model.
// These are the options that can be set in the config.json file.
// Only a subset of the options are implemented in this struct since not all models support all options.
// Avoid omitempty json flags so we do not omit zero value num types that Ollama needs.
type ModelOptions struct {
	NumCtx        int     `json:"num_ctx"`        // Sets the size of the context window used to generate the next token. (Default: 2048)
	RepeatLastN   int     `json:"repeat_last_n"`  // Sets how far back for the model to look back to prevent repetition. (Default: 64, 0 = disabled, -1 = num_ctx)
	RepeatPenalty float64 `json:"repeat_penalty"` // Sets how strongly to penalize repetitions. (Default: 1.1)
	Temperature   float64 `json:"temperature"`    // The temperature of the model. (Default: 0.8)
}

// NewModelOptions creates a modelOptions struct with default values.
// Used to set defaults and override with values from config.json.
func NewModelOptions() ModelOptions {
	return ModelOptions{
		NumCtx:        2048,
		RepeatLastN:   64,
		RepeatPenalty: 1.1,
		Temperature:   0.8,
	}
}

// OutputFormat struct contains the output format for the model.
type OutputFormat struct {
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Required   []string       `json:"required"`
}

// ModelConfig struct contains the configuration for the model.
type ModelConfig struct {
	Model        string       `json:"model"`
	System       string       `json:"system"`
	Options      ModelOptions `json:"options"`
	OutputFormat OutputFormat `json:"format"` // Optional. If this is set to "json" make sure the prompt instructs the agent to ouput json.
}

// Request struct is passed to ollama api to generate a response.
// It contains configuration for the model as well as the prompt.
type request struct {
	Model   string       `json:"model"`
	Options ModelOptions `json:"options"`
	Prompt  string       `json:"prompt"`
	Stream  bool         `json:"stream"` // Set to false since we are not chatting.
	System  string       `json:"system"`
	Format  any          `json:"format"`
	Raw     bool         `json:"raw"` // Not implemented yet.
}

// Response struct contains the response from ollama.
type Response struct {
	Model              string    `json:"model"`
	CreatedAt          time.Time `json:"created_at"`
	Output             string    `json:"response"`
	Done               bool      `json:"done"`
	TotalDuration      int64     `json:"total_duration"`
	LoadDuration       int       `json:"load_duration"`
	PromptEvalCount    int       `json:"prompt_eval_count"`
	PromptEvalDuration int       `json:"prompt_eval_duration"`
	EvalCount          int       `json:"eval_count"`
	EvalDuration       int64     `json:"eval_duration"`
}

// Generate is a struct that contains the configuration for the Generate struct.
// It also contains the HTTP client, API base URL, and folder base path.
type Generate struct {
	folder      string        // folder name
	Prompt      string        // optional initial prompt text
	ModelConfig ModelConfig   // model configuration
	client      http.Client   // HTTP client
	apiBase     string        // API base URL
	folderBase  string        // folder base path
	timeout     time.Duration // timeout for the request
}

// Option is a function that takes a pointer to a Generate struct and modifies it.
type Option func(*Generate)

// NewGenerate creates a new Generate struct with the given request and options.
func NewGenerate(folder string, options ...Option) (Generate, error) {
	g := Generate{
		folder:      folder,
		Prompt:      "",
		ModelConfig: ModelConfig{},
		client:      http.Client{},
		apiBase:     "",
		folderBase:  "",
		timeout:     300 * time.Second,
	}

	for _, opt := range options {
		opt(&g)
	}

	if g.apiBase == "" {
		return g, fmt.Errorf("API base URL is required")
	}

	// If we have a prompt, use it. Otherwise, read the prompt.txt file.
	if g.Prompt == "" {
		promptFromFile, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "prompt.txt"))
		if err != nil {
			return g, fmt.Errorf("error reading prompt.txt: %w", err)
		}
		g.Prompt = string(promptFromFile)
	}

	// Get the model config.
	config, err := g.config()
	if err != nil {
		return g, fmt.Errorf("error getting model config: %w", err)
	}
	g.ModelConfig = config

	return g, nil
}

// WithPrompt sets the prompt for the Generate struct.
func WithPrompt(prompt string) Option {
	return func(g *Generate) {
		g.Prompt = prompt
	}
}

// WithClient sets the HTTP client for the Generate struct.
func WithClient(client http.Client) Option {
	return func(g *Generate) {
		g.client = client
	}
}

// WithAPIBase sets the API base URL for the Generate struct.
func WithAPIBase(apiBase string) Option {
	return func(g *Generate) {
		g.apiBase = apiBase
	}
}

// WithFolderBase sets the folder base path for the Generate struct.
func WithFolderBase(folderBase string) Option {
	return func(g *Generate) {
		g.folderBase = folderBase
	}
}

// WithTimeout sets the timeout for the Generate struct.
func WithTimeout(timeout int) Option {
	return func(g *Generate) {
		g.timeout = time.Duration(timeout) * time.Second
	}
}

// config reads the config.json file from the folder and returns a ModelConfig struct or an error.
func (g *Generate) config() (ModelConfig, error) {
	// Read the config.json file from the path and unmarshal it into a modelConfig struct.
	empty := ModelConfig{}
	configDirPath := filepath.Join(g.folderBase, g.folder)
	configContent, err := os.ReadFile(filepath.Join(configDirPath, "config.json"))
	if err != nil {
		return empty, fmt.Errorf("error reading config.json: %w", err)
	}

	config := ModelConfig{Options: NewModelOptions()}
	err = json.Unmarshal(configContent, &config)
	if err != nil {
		return empty, fmt.Errorf("error unmarshalling config.json: %w", err)
	}

	// Read the system.txt file.
	systemPromptFromFile, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "system.txt"))
	if err != nil {
		return empty, fmt.Errorf("error reading system.txt: %w", err)
	}
	config.System = string(systemPromptFromFile)

	// If optional format.json file is present in the folder, use it.
	if g.ModelConfig.OutputFormat.Type == "" {
		formatFromFile, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "format.json"))
		if err == nil {
			var format OutputFormat
			err := json.Unmarshal(formatFromFile, &format)
			if err != nil {
				return empty, fmt.Errorf("error unmarshalling format.json: %w", err)
			}
			// Set the format in the config.
			config.OutputFormat = format
		}
	}

	return config, nil

}

// Post sends a POST request with context to the Ollama API and returns a Response struct or an error.
func (g *Generate) Post(ctx context.Context) (Response, error) {
	empty := Response{}

	var format any
	if g.ModelConfig.OutputFormat.Type != "" {
		format = g.ModelConfig.OutputFormat
	} else {
		format = ""
	}

	// Build the request
	req := request{
		Model:   g.ModelConfig.Model,
		Options: g.ModelConfig.Options,
		Prompt:  g.Prompt,
		Stream:  false,
		System:  g.ModelConfig.System,
		Format:  format,
		Raw:     false,
	}

	// Create a new context with a timeout from parent context.
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(g.timeout)*time.Second)
	defer cancel()

	// Marshal the request into JSON.
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return empty, fmt.Errorf("error marshalling request: %w", err)
	}

	// Create a new request with context and JSON body.
	request, err := http.NewRequestWithContext(requestCtx, "POST", g.apiBase+"/generate", bytes.NewReader(reqJSON))
	if err != nil {
		return empty, fmt.Errorf("error creating POST request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Send the request.
	resp, err := g.client.Do(request)
	if err != nil {
		return empty, fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200.
	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("error response status code: %d", resp.StatusCode)
	}

	// Unmarshal the response body into a Response struct.
	response := Response{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return empty, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return response, nil
}
