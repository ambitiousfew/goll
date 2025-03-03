# Goll

Goll is a lightweight command-line interface (CLI) that streamlines your interaction with a local Ollama Generate API. With Goll, you can:

* Customize your prompts in one place using folder structures.
* Easily iterate over multiple prompts to save time and reduce errors.
* Chain multiple prompts for seamless workflow automation.

By automating these common tasks, Goll aims to make your local Ollama CLI experience more efficient and enjoyable.

Yes, naming is hard. `goll` is unique, short and easy to type.

## Background

 We wanted a *very simple* CLI to enable us to break up more complex prompts into a pipeline of smaller, task oriented prompts using local Ollama. We also wanted to be able to easily adjust the model and it's settings for each prompt in the pipeline.

## Features

* Support iteration of subfolder prompts in a folder.
* Supports simple chaining of multiple prompts.
* Customize the model for each step in chain.
* Print and log each response with metrics such as tokens per second.
* Support for structured/JSON output.

## Prereqs

* Go v1.23 or greater installed.
* Access to an Ollama API.  Currently tested on a local instance and only uses `generate` endpoint.
* Ensure you have pulled whatever Ollama models you reference in your config.  

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
  ./goll -f folder1,folder2,folder3 -p "optional beginning prompt" -v
  ```

* `-f`: Comma-separated list of folder names.  You have to provide at least one folder.
* `-p`: Optional text prompt.  Applied to first folder in chain, or to all subfolders if -r is used.  If not present we expect a prompt.txt in first folder of a chain, or in all subfolders if -r is used.
* `-r`: Optional recurse of folder. If used only one folder can be set with -f flag.  No prompt chaining.  Will iterate over each subfolder in the given folder running each prompt.  Only one level supported.
* `-v`: Optional verbose output. Print results of each step to command line.

### Example: Chaining With Prompt

When you want to pass one prompt in and let the pipeline you create pass the output to the next folder as an input prompt.

```sh
  ./goll -f improve_prompt,basic -p "why is the sky blue"
```

* Starts with improving the prompt provided with the `-p` flag and passes the improved prompt to the basic folder.  If we did not use `-p` flag here we would expect prompt.txt in the improve_prompt folder to read "why is the sky blue".
* Each step will generate an output log with date/time appended.

### Example: Folder Recursion With Verbose output

This is useful for testing combinations of prompts/models/configs in one call.  Or use the `-p` flag to pass the same prompt to each model/config in subfolders.  Set it all up, run it, and go get coffee.

```sh
  ./goll -f improve_prompt_test -r -v
```

* Iterates over the `improve_prompt_test` folder and calls generate on each subfolder.  We expect prompt.txt in each subfolder since we did not use `-p` flag to pass in a prompt.
* No prompt chaining supported.
* `-v` Will cause verbose output to print on command line.
* Each step will generate an output log with date/time appended.

## Configuration

Each folder should contain the following files:

* `config.json`: Configuration file for the model.
* `system.txt`: System prompt.
* `prompt.txt`: User prompt.
* `format.json`: Optional. Contains output format schema per Ollama spec in JSON format.

Note:  If chaining, `prompt.txt` is only required for the first folder in chain if you do not use the `-p` flag.  Each step will write to the next folders `prompt.txt` file.  If calling each folder recursively with -r flag then you can either pass the same prompt to each subfolder with the `-p` flag or you need to provide a `prompt.txt` in each subfolder being called.

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

These are applied in every step in chain.  

`Timeout` is in seconds and is max time each step/request can take to the Ollama API before being closed.

## Known Issues\Limitations

* Some settings might be better in per prompt config.  Settings such as `api_base_url` and `timeout` might be better served in each prompt's config.json.  This would allow for more flexibility.  Current approach works but might be too simple for some.
* `<think></think>` tags and enclosed content are automatically removed when passing prompt.  Might be better to make this an option rather than force removal for each prompt.
* Support for limited model options of `num_ctx`, `repeat_last_n`, `repeat_penalty`, and `temperature` in `config.json` as it appears not all models support all options.

## Notes

* Running tests on MacBook Pro M3 with 12/18 cores and 32GB RAM
* Using Ollama latest
* `llama3.2:latest`: ~50 tokens per second
* `deepseek-r1:8b`: ~24 tokens per second
* `deepseek-r1:14b`: ~12 tokens per second

## More Options

* If you need broader support for more than local Ollama, and great community prompts: Fabric AI <https://github.com/danielmiessler/fabric>
* Chatbox is a pretty cool UI that allows simple model config tweaks and custom system prompts. <https://github.com/Bin-Huang/chatbox>
* Open WebUI is a powerful self hosted UI that can do everything goll does plus much more. <https://github.com/open-webui/open-webui>
