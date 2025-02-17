# Goll: Ollama CLI Tool

ALPHA CODE! Needs more testing but the basics work.

The Ollama CLI tool is a simple command-line interface for generating responses from the Ollama API. The tool reads model configuration files from specified folders, sends requests to the API, and prints the responses. It supports chaining multiple prompts by using the output of one prompt as the input for the next prompt.

## Features

- Reads model configuration files from specified folders for each request.
- Sends requests to the Ollama API to generate responses.
- Supports chaining multiple prompts.
- Specify a timeout to cancel requests if they take too long.
- Handles OS signals to gracefully cancel requests on interrupt.
- Prints responses and metrics such as tokens per second.
- Logs the output file.

## Prereqs

- Go v1.23 or greater installed.
- Access to an Ollama API.  Currently tested on a local instance and only uses `generate` endpoint.

## Installation

1. Clone the repository:
  
  ```sh
   git clone https://github.com/FIX-ME-PLEASE
   cd goll
  ```

2. Build the CLI tool:

  ```sh
  go build -o goll ./cmd/goll
  ```

## Usage

  ```sh
  ./goll -f folder1,folder2,folder3
  ```

- `-f`: Comma-separated list of folder names.  You have to provide at least one folder.

## Configuration

Each folder should contain the following files:

- `config.json`: Configuration file for the model.
- `system.txt`: Sytem prompt.
- `prompt.txt`: User prompt.

Examples for each file can be found in `prompts` folder.  In general, the config.json fields match the Ollama generate API spec.

### Tool Settings

There is a `settings.json` in the root that allows you to set some global defaults for the cli.

```json
{
  "api_base_url": "http://localhost:11434/api",
  "folder_base_path": "prompts",
  "timeout": 300
}

These are applied to every request.
