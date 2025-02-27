// Package tool contains the Settings struct that is used to store the global settings for goll.
package tool

// Settings struct contains the global settings for goll.
// Set from the settings.json file.
type Settings struct {
	APIBase    string `json:"api_base_url"`
	FolderBase string `json:"folder_base_path"`
	Timeout    int    `json:"timeout"`
}
