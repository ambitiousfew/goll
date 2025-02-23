# Goll

Goll is a simple command-line interface for chaining multiple prompts when using Ollama API.

*This thing is highly experimental.* Needs more testing but the basics work.  Our testing will focus on chaining prompts using a mix of unstructured and structured prompts.  Also not sure how a local Ollama instance will perform when we potentially switch out relatively large models for each step in a chain on a resource constrained local server.

## Background

 We wanted a very simple CLI to enable us to break up more complex prompts into a pipeline of smaller, task oriented prompts using local Ollama. We also wanted to be able to easily adjust the model and it's settings for each prompt in the pipeline.  There appears to be plenty of UI's to support chat with custom callbacks etc.  Also projects like Fabric AI.  We just wanted something simple to use from command line focused on this use case.

 Yes, naming is hard. `goll` is unique, short and easy to type.

## Features

- Supports simple chaining of multiple prompts.
- Customize the model for each step in chain.
- Print and log each response with metrics such as tokens per second.
- Support for structured/JSON output

## Prereqs

- Go v1.23 or greater installed.
- Access to an Ollama API.  Currently tested on a local instance and only uses `generate` endpoint.
- Ensure you have pulled whatever Ollama models you reference in your config.  

## Installation

1. Clone the repository:
  
  ```sh
   git clone https://github.com/ambitiousfew/goll.git
   cd goll
  ```

2. Build the CLI tool:

  ```sh
  go build -o goll ./cmd/goll
  ```

## Usage

  ```sh
  ./goll -f folder1,folder2,folder3 -p "optional beginning prompt"
  ```

- `-f`: Comma-separated list of folder names.  You have to provide at least one folder.
- `-p`: Optional text prompt.  Applied to first folder in chain.  If not present we expext a prompt.txt in first folder.

### Example

```sh
  ./goll -f improve_prompt,basic -p "why is the sky blue"
```

- Starts with improving the prompt provided with the `-p` flag and passes the improved prompt to the basic folder.  If we did not use `-p` flag here we would expect prompt.txt in the improve_prompt folder to read "why is the sky blue".

## Configuration

Each folder should contain the following files:

- `config.json`: Configuration file for the model.
- `system.txt`: Sytem prompt.
- `prompt.txt`: User prompt.

Note:  `prompt.txt` is only required for the first folder in chain if you do not use the `-p` flag.  Each step will write to next folders `prompt.txt` file.

Examples for each file can be found in `prompts` folder.  In general, the config.json fields match the Ollama generate API spec.

### Tool Settings

There is a `settings.json` in the root that allows you to set some global defaults for the cli.

```json
{
  "api_base_url": "http://localhost:11434/api",
  "folder_base_path": "prompts",
  "timeout": 300
}
```

These are applied as globals and used in every step in chain.  

`Timeout` is in seconds and is max time each step/request can take to the Ollama API before being closed.
