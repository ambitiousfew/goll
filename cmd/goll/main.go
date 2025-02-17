// package main implements a simple CLI tool to generate responses from the ollama API.
// The tool reads the config.json, system.txt, and prompt.txt files from the specified folder path.
// It sends a request to the ollama generate API endpoint and prints the response.
// If more than one folder name is provided in the -f flag, the response is written to the prompt.txt file in the next folder.
// The tool is designed to be used in a chain of prompts where the output of one prompt is used as the input for the next prompt.
// The tool uses a context with a timeout to cancel the request if it takes too long.
// The tool also sets up signal handling to cancel the context on interrupt.
// The tool prints a spinner while waiting for each response.
// The tool prints each response and metrics such as tokens per second.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ambitiousfew/goll/internal/ollama"
	"github.com/ambitiousfew/goll/internal/toolconfig"
)

func main() {
	// Define and parse command-line flags
	folder := flag.String("f", "", "One or more comma seperated folder names.")
	flag.Parse()

	// Ensure at least one folder name is provided
	if *folder == "" {
		fmt.Println("Error: At least one folder is required")
		flag.Usage()
		os.Exit(1)
	}

	// Split the folder flag by comma
	folders := strings.Split(*folder, ",")

	// Read in settings from the settings.json file
	settingsContent, err := os.ReadFile("settings.json")
	if err != nil {
		fmt.Printf("Error reading settings.json: %v\n", err)
		os.Exit(1)
	}
	settings := toolconfig.Settings{}
	err = json.Unmarshal(settingsContent, &settings)
	if err != nil {
		fmt.Printf("Error unmarshalling settings.json: %v\n", err)
		os.Exit(1)
	}

	// Ensure each folder exists in the folder_base_path
	for _, folder := range folders {
		folderPath := filepath.Join(settings.FolderBase, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			fmt.Printf("Error: Folder %s does not exist in %s\n", folder, settings.FolderBase)
			os.Exit(1)
		}
	}

	// Run the tool for each folder
	err = run(settings, folders)
	if err != nil {
		fmt.Println("Error running goll: ", err)
		os.Exit(1)
	}

}

// run function generates a response for each folder in the folders slice.
func run(settings toolconfig.Settings, folders []string) error {

	// Loop through each folder and generate a response
	for index, folder := range folders {
		// Create a context
		// Signal worker is in charge of cancelling the context
		ctx, cancel := context.WithCancel(context.Background())
		// Set up signal handling to cancel context on interrupt
		// sigChannel is closed prior to any exit in order to signal clean up
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		// Wait for the signal handling goroutine to finish cleaning up
		var wg sync.WaitGroup
		wg.Add(1)
		// Signal handling goroutine
		go func() {
			defer wg.Done()
			for sig := range sigChan {
				fmt.Printf("Received exit signal: %v. Cancelling context.\n", sig)
				cancel()
			}
		}()

		// Create a new ollama generate instance
		gen := ollama.NewGenerate(
			folder,
			ollama.WithAPIBase(settings.APIBase),
			ollama.WithFolderBase(settings.FolderBase),
			ollama.WithClient(http.Client{}),
			ollama.WithTimeout(settings.Timeout),
		)

		fmt.Printf("Generating response using folder: %s\n", folder)
		// Print a spinner while waiting for the response
		go spinner(ctx)

		// Send the request to the ollama generate API
		resp, err := gen.Post(ctx)
		if err != nil {
			close(sigChan)
			wg.Wait()
			return fmt.Errorf("error generating response: %v", err)
		}

		// convert evalution time from nanoseconds to seconds as float
		evalTime := float64(resp.EvalDuration) / 1e9
		// Compute tokens per second
		tps := float64(resp.EvalCount) / evalTime
		// Print the response and metrics
		fmt.Printf("\n\nResponse: %s", resp.Output)
		fmt.Printf("\n\nGenerated %d tokens in %.2f seconds", resp.EvalCount, evalTime)
		fmt.Printf("\nTokens per second: %.2f\n", tps)

		// If there is a next folder, write the response to prompt.txt file in the next folder
		if index < len(folders)-1 {
			nextFolder := folders[index+1]
			nextFolderPath := filepath.Join(settings.FolderBase, nextFolder)
			nextPromptFilePath := filepath.Join(nextFolderPath, "prompt.txt")
			err = os.WriteFile(nextPromptFilePath, []byte(resp.Output), 0644)
			if err != nil {
				close(sigChan)
				wg.Wait()
				return fmt.Errorf("error writing prompt.txt: %v", err)
			}
			fmt.Printf("Response written to %s\n", nextPromptFilePath)
		}

		// Write to output.log file
		outputLogPath := filepath.Join(settings.FolderBase, folder, "output.log")
		outputLog := fmt.Sprintf("Response: %s\n\nTokens per second: %.2f\n", resp.Output, tps)
		err = os.WriteFile(outputLogPath, []byte(outputLog), 0644)
		if err != nil {
			close(sigChan)
			wg.Wait()
			return fmt.Errorf("error writing output.log: %v", err)
		}
		fmt.Printf("Output written to %s\n", outputLogPath)

		// Clean up resources
		close(sigChan)
		wg.Wait()

		fmt.Printf("%s completed successfully\n\n", folder)
	}

	return nil
}

// spinner function prints a spinner while waiting for the response.
func spinner(ctx context.Context) {
	spin := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			index := i % len(spin)
			i++
			fmt.Printf("\r%s", spin[index])
			time.Sleep(100 * time.Millisecond)
		}
	}
}
