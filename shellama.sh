#!/bin/bash

# This script runs the DeepSeek model using the Ollama API.
# It takes a folder as input and reads the system and user prompts from files.
# Optionally, it can take a prompt text as a command line argument.
# It was the genesis of the goll project

set -eo pipefail

# Define constants
readonly MODEL_NAME="deepseek-r1:latest"
readonly SYSTEM_FILE="system.txt"
readonly PROMPT_FILE="prompt.txt"

# Validate input
if [ "$#" -lt 1 ]; then
  echo "Usage: $0 <folder> [prompt]"
  exit 1
fi
# Folder containing system and optional user prompt
FOLDER="$1"
# Optional prompt text. If not provided it's read from file.
PROMPT="$2"

# Validate folder and files
cd "$FOLDER" || { echo "Error: Could not enter folder '$FOLDER'"; exit 1; }

SYSTEM_PROMPT=$(cat "$SYSTEM_FILE" | jq -sR .)
echo "Using system prompt from: $SYSTEM_FILE"

if [ -z "$PROMPT" ]; then
  echo "Using user prompt from: $PROMPT_FILE"
  USER_PROMPT=$(cat "$PROMPT_FILE" | jq -sR .)
else 
  echo "Using user prompt from command line: $PROMPT"
  USER_PROMPT=$(echo "$PROMPT" | jq -sR .)
fi

# Run Ollama and capture output
echo "Waiting for response from Ollama API..."
# Print a new character every 1 seconds to show progress
spin_chars=("|" "/" "-" "\\")
i=0
while true; do
  printf " [%c] " "${spin_chars[i]}"
  i=$(( (i+1) % 4 ))
  sleep 0.1
  printf "\b\b\b\b\b\b"
done &

# Capture the PID of the background process
DOT_PID=$!

# Run the curl command and capture the response
RESPONSE=$(curl -s http://localhost:11434/api/generate \
  -d "{\"stream\":false,\"system\":$SYSTEM_PROMPT,\"prompt\":$USER_PROMPT,\"model\":\"$MODEL_NAME\"}" \
  -H "Content-Type: application/json" | jq -r .response)

# Kill the background process
kill $DOT_PID

# Print the response
echo
echo "$RESPONSE"
echo

echo "Shellama command completed successfully"
