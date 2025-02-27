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
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/ambitiousfew/goll/internal/ollama"
	"github.com/ambitiousfew/goll/internal/tool"
)

type args struct {
	folders []string
	multi   bool
	prompt  string
	verbose bool
	recurse bool
}

func main() {
	// Define and parse command-line flags
	folder := flag.String("f", "", "One or more comma seperated folder names. Limit one parnet folder if using with -r flag")
	prompt := flag.String("p", "", "Optional.  Initial prompt text to use instead of reading from prompt.txt file.")
	verbose := flag.Bool("v", false, "Optional. Print output to stdout.")
	recurse := flag.Bool("r", false, "Optional. Recurse through subfolders. If set -f can only have one folder.")
	flag.Parse()

	// Ensure at least one folder name is provided
	if *folder == "" {
		fmt.Println("Error: At least one folder is required")
		flag.Usage()
		os.Exit(1)
	}

	// Split the folder flag by comma and clean whitespace
	rawFolders := strings.Split(*folder, ",")
	folders := make([]string, 0, len(rawFolders))
	for _, rawFolder := range rawFolders {
		folder := strings.TrimSpace(rawFolder)
		if folder != "" {
			folders = append(folders, folder)
		}
	}

	// Chekc if -r flag is set and only one folder is provided
	if *recurse && len(folders) > 1 {
		fmt.Println("Error: -r flag can only be used with one folder")
		flag.Usage()
		os.Exit(1)
	}

	// Read in settings from the settings.json file
	settingsContent, err := os.ReadFile("settings.json")
	if err != nil {
		fmt.Printf("Error reading settings.json: %v\n", err)
		os.Exit(1)
	}
	settings := tool.Settings{}
	err = json.Unmarshal(settingsContent, &settings)
	if err != nil {
		fmt.Printf("Error unmarshalling settings.json: %v\n", err)
		os.Exit(1)
	}

	// Ensure each folder exists in the folder_base_path
	for _, folder := range folders {
		folderPath := filepath.Join(settings.FolderBase, folder)
		if _, err := os.Stat(folderPath); os.IsNotExist(err) {
			fmt.Printf("Error: Folder %s does not exist in %s\n", folderPath, settings.FolderBase)
			os.Exit(1)
		}
	}

	// If recurse flag is set, get all subfolders of the parent folder
	if *recurse {
		parentFolder := filepath.Join(settings.FolderBase, folders[0])
		var subfolders []string

		err := filepath.Walk(parentFolder, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && path != parentFolder {
				subfolders = append(subfolders, path)
			}
			return nil
		})
		if err != nil {
			fmt.Printf("Error getting subfolders: %v\n", err)
			os.Exit(1)
		}
		folders = subfolders
	}

	args := args{
		folders: folders,
		prompt:  *prompt,
		verbose: *verbose,
		recurse: *recurse,
	}

	// Run the tool for each folder
	err = run(settings, args)
	if err != nil {
		fmt.Println("Error running goll: ", err)
		os.Exit(1)
	}

}

// run function generates a response for each folder in the folders slice.
func run(settings tool.Settings, args args) error {

	// Create a context
	// Signal worker is in charge of cancelling the context
	ctx, cancel := context.WithCancel(context.Background())
	// Set up signal handling to cancel context on interrupt
	// sigChannel is closed prior to any exit in order to signal clean up
	sigChan := make(chan os.Signal, 1)
	defer close(sigChan)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	// Wait for the signal handling goroutine to finish cleaning up
	// Signal handling goroutine
	go func() {
		<-sigChan
		signal.Stop(sigChan)
		fmt.Print("Received exit signal. Cancelling context.\n")
		cancel()
	}()

	// Start a spinner goroutine
	spin := make(chan bool)
	defer close(spin)
	go spinner(ctx, spin)

	// Loop through each folder and generate a response
	for index, folder := range args.folders {
		// If firstPrompt is provided, set the prompt text for first folder
		// Generate will ignore empty prompt text and use prompt.txt file
		var prompt string
		if index == 0 && args.prompt != "" && !args.recurse {
			prompt = args.prompt
		}

		// If we are recursing and prompt is provided, set the prompt text for all folders
		// Generate will ignore empty prompt text and use prompt.txt file
		folderBase := settings.FolderBase
		if args.recurse && args.prompt != "" {
			prompt = args.prompt
			// subfolders will be in the format: folderBase/parentFolder/subfolder
			folderBase = ""
		}

		// Create a new ollama generate instance
		gen, err := ollama.NewGenerate(
			folder,
			ollama.WithPrompt(prompt),
			ollama.WithAPIBase(settings.APIBase),
			ollama.WithFolderBase(folderBase),
			ollama.WithClient(http.Client{}),
			ollama.WithTimeout(settings.Timeout),
		)
		if err != nil {
			return fmt.Errorf("error creating generate instance: %v", err)
		}

		modelConfig := gen.Config()

		// Pretty print modelConfig
		modelConfigJSON, err := json.MarshalIndent(modelConfig, "", "  ")
		if err != nil {
			return fmt.Errorf("error marshalling modelConfig: %v", err)
		}

		if args.verbose {
			fmt.Printf("Generating response using folder: %s\n  With Model Config: %v\n", folder, string(modelConfigJSON))
		}

		// Start the spinner
		select {
		case <-ctx.Done():
			return nil
		case spin <- true:
		}

		// Send the request to the ollama generate API
		resp, err := gen.Post(ctx)
		if err != nil {
			return fmt.Errorf("error generating response: %v", err)
		}

		// Stop the spinner
		select {
		case <-ctx.Done():
			return nil
		case spin <- false:
		}

		// convert evalution time from nanoseconds to seconds as float
		evalTime := float64(resp.EvalDuration) / 1e9
		// Compute tokens per second
		tps := float64(resp.EvalCount) / evalTime

		if args.verbose {
			// Print the response and metrics
			fmt.Printf("\n\nResponse: %s", resp.Output)
			fmt.Printf("\n\nGenerated %d tokens in %.2f seconds", resp.EvalCount, evalTime)
			fmt.Printf("\nTokens per second: %.2f\n", tps)
		}

		// If there is a next folder and we are not recursing, write the response to prompt.txt file in the next folder
		if index < len(args.folders)-1 && !args.recurse {
			nextFolder := args.folders[index+1]
			nextFolderPath := filepath.Join(folderBase, nextFolder)
			nextPromptFilePath := filepath.Join(nextFolderPath, "prompt.txt")

			// Remove content wrapped with <think></think> tags
			re := regexp.MustCompile(`(?s)<think>.*?</think>`)
			cleanedOutput := re.ReplaceAllString(resp.Output, "")

			err = os.WriteFile(nextPromptFilePath, []byte(cleanedOutput), 0644)
			if err != nil {
				return fmt.Errorf("error writing prompt.txt: %v", err)
			}
			if args.verbose {
				fmt.Printf("Response written to %s\n", nextPromptFilePath)
			}
		}

		// Write to output_date_time.log file
		outputLogFileName := fmt.Sprintf("output_%s.log", time.Now().Format("2006-01-02_15-04-05"))
		outputLogPath := filepath.Join(folderBase, folder, outputLogFileName)
		outputLog := fmt.Sprintf(
			"Prompt: %s\n\n"+
				"Response: %s\n\n"+
				"Generated %d tokens in %.2f seconds\n"+
				"Tokens per second: %.2f\n"+
				"Using model config: %s\n",
			gen.Prompt(),
			resp.Output,
			resp.EvalCount,
			evalTime,
			tps,
			modelConfigJSON,
		)
		err = os.WriteFile(outputLogPath, []byte(outputLog), 0644)
		if err != nil {
			return fmt.Errorf("error writing output.log: %v", err)
		}
		fmt.Printf("Output written to %s\n", outputLogPath)

		fmt.Printf("%s completed successfully\n\n", folder)
	}

	return nil
}

// spinner function prints a spinner while waiting for the response.
func spinner(ctx context.Context, spin <-chan bool) {
	tick := time.NewTicker(100 * time.Millisecond)
	tick.Stop()
	spinChar := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return
		case run, ok := <-spin:
			if !ok {
				tick.Stop()
				return
			}
			if !run {
				tick.Stop()
				if i > 0 {
					fmt.Print("\r")
				}
			} else {
				tick.Reset(100 * time.Millisecond)
			}
		case <-tick.C:
			index := i % len(spinChar)
			i++
			fmt.Printf("\r%s", spinChar[index])
		}
	}

}
