package toolconfig

// Settings struct contains the global settings for goll.
// Set from the settings.json file.
type Settings struct {
	APIBase    string `json:"api_base_url"`
	FolderBase string `json:"folder_base_path"`
	Timeout    int    `json:"timeout"`
}
