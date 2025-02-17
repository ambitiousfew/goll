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

// modelOptions struct contains the Ollama options for the model.
type modelOptions struct {
	Mirostat      int     `json:"mirostat,omitempty"`       // Enable Mirostat sampling for controlling perplexity. (default: 0, 0 = disabled, 1 = Mirostat, 2 = Mirostat 2.0)
	MirostatEta   float64 `json:"mirostat_eta,omitempty"`   // Influences how quickly the algorithm responds to feedback from the generated text. (Default: 0.1)
	MirostatTau   float64 `json:"mirostat_tau,omitempty"`   // Controls the balance between coherence and diversity of the output. (Default: 5.0)
	NumCtx        int     `json:"num_ctx,omitempty"`        // Sets the size of the context window used to generate the next token. (Default: 2048)
	RepeatLastN   int     `json:"repeat_last_n,omitempty"`  // Sets how far back for the model to look back to prevent repetition. (Default: 64, 0 = disabled, -1 = num_ctx)
	RepeatPenalty float64 `json:"repeat_penalty,omitempty"` // Sets how strongly to penalize repetitions. (Default: 1.1)
	Temperature   float64 `json:"temperature,omitempty"`    // The temperature of the model. (Default: 0.8)
	Seed          int     `json:"seed,omitempty"`           // Sets the random number seed to use for generation. (Default: 0)
	Stop          string  `json:"stop,omitempty"`           // Sets the stop sequences to use. (Default: "")
	NumPredict    int     `json:"num_predict,omitempty"`    // Maximum number of tokens to predict when generating text. (Default: -1)
	TopK          int     `json:"top_k,omitempty"`          // Reduces the probability of generating nonsense. (Default: 40)
	TopP          float64 `json:"top_p,omitempty"`          // Works together with top-k. (Default: 0.9)
	MinP          float64 `json:"min_p,omitempty"`          // Alternative to the top_p. (Default: 0.0)
}

// modelConfig struct contains the configuration for the Generate struct.
type modelConfig struct {
	Model        string       `json:"model"`
	Options      modelOptions `json:"options"`
	OutputFormat string       `json:"format"` // Optional. If this is set to "json" make sure the prompt instructs the agent to ouput json.
}

// Request struct is passed to ollama api to generate a response.
// It contains configuration for the model as well as the prompt.
type request struct {
	Model   string       `json:"model"`
	Options modelOptions `json:"options"`
	Prompt  string       `json:"prompt"`
	Stream  bool         `json:"stream"` // set to false since we are not chatting
	System  string       `json:"system"`
	Format  string       `json:"format"` // Not implemented yet
	Raw     bool         `json:"raw"`    // Not implemented yet
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
	folder     string
	client     http.Client
	apiBase    string
	folderBase string
	timeout    time.Duration
}

// Option is a function that takes a pointer to a Generate struct and modifies it
type Option func(*Generate)

// NewGenerate creates a new Generate struct with the given request and options
func NewGenerate(folder string, options ...Option) Generate {
	g := Generate{
		folder:     folder,
		client:     http.Client{},
		apiBase:    "http://localhost:11434/api",
		folderBase: "prompts",
		timeout:    300 * time.Second,
	}

	for _, opt := range options {
		opt(&g)
	}

	return g
}

// WithClient sets the HTTP client for the Generate struct
func WithClient(client http.Client) Option {
	return func(g *Generate) {
		g.client = client
	}
}

// WithAPIBase sets the API base URL for the Generate struct
func WithAPIBase(apiBase string) Option {
	return func(g *Generate) {
		g.apiBase = apiBase
	}
}

// WithFolderBase sets the folder base path for the Generate struct
func WithFolderBase(folderBase string) Option {
	return func(g *Generate) {
		g.folderBase = folderBase
	}
}

// WithTimeout sets the timeout for the Generate struct
func WithTimeout(timeout int) Option {
	return func(g *Generate) {
		g.timeout = time.Duration(timeout) * time.Second
	}
}

// config reads the config.json file and returns a modelConfig struct or an error
func (g *Generate) config() (modelConfig, error) {
	empty := modelConfig{}
	// Read the config.json file and unmarshal it into a modelConfig struct
	configContent, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "config.json"))
	if err != nil {
		return empty, fmt.Errorf("error reading config.json: %w", err)
	}

	config := &modelConfig{}
	err = json.Unmarshal(configContent, config)
	if err != nil {
		return empty, fmt.Errorf("error unmarshalling config.json: %w", err)
	}

	return *config, nil
}

// requestFromFolder reads config.json, system.txt, and prompt.txt files from the folder and returns a request struct or an error
func (g *Generate) requestFromFolder() (request, error) {
	empty := request{}
	// Get the modelConfig from the config.json file
	config, err := g.config()
	if err != nil {
		return empty, fmt.Errorf("error reading config.json: %w", err)
	}

	// Read the system.txt file
	systemContent, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "system.txt"))
	if err != nil {
		return empty, fmt.Errorf("error reading system.txt: %w", err)
	}

	// Read the prompt.txt file
	promptContent, err := os.ReadFile(filepath.Join(g.folderBase, g.folder, "prompt.txt"))
	if err != nil {
		return empty, fmt.Errorf("error reading prompt.txt: %w", err)
	}

	return request{
		Model:   config.Model,
		Options: config.Options,
		Prompt:  string(promptContent),
		Stream:  false,
		System:  string(systemContent),
		Format:  config.OutputFormat,
		Raw:     false,
	}, nil
}

// Post sends a POST request with context to the Ollama API and returns a Response struct or an error
func (g *Generate) Post(ctx context.Context) (Response, error) {
	empty := Response{}
	// Build the request from the folder
	req, err := g.requestFromFolder()
	if err != nil {
		return empty, fmt.Errorf("error getting request info from folder: %w", err)
	}

	// Create a new context with a timeout from parent context
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(g.timeout)*time.Second)
	defer cancel()

	// Marshal the request into JSON
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return empty, fmt.Errorf("error marshalling request: %w", err)
	}

	// Create a new request with context and JSON body
	request, err := http.NewRequestWithContext(requestCtx, "POST", g.apiBase+"/generate", bytes.NewReader(reqJSON))
	if err != nil {
		return empty, fmt.Errorf("error creating POST request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := g.client.Do(request)
	if err != nil {
		return empty, fmt.Errorf("error sending POST request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status code is not 200
	if resp.StatusCode != http.StatusOK {
		return empty, fmt.Errorf("error response status code: %d", resp.StatusCode)
	}

	// Unmarshal the response body into a Response struct
	response := Response{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return empty, fmt.Errorf("error unmarshalling response body: %w", err)
	}

	return response, nil
}
