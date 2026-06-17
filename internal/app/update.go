package app

// UpdateResult holds the outcome of a yeast update check or update operation.
type UpdateResult struct {
	CurrentVersion   string `json:"current_version"`
	TargetVersion    string `json:"target_version"`
	UpdateAvailable  bool   `json:"update_available"`
	AlreadyLatest    bool   `json:"already_latest"`
	CheckOnly        bool   `json:"check_only"`
	ChecksumVerified bool   `json:"checksum_verified,omitempty"`
	BinaryPath       string `json:"binary_path,omitempty"`
	Success          bool   `json:"success"`
}
