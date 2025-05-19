package domain

type Mount struct {
	Path     string `json:"mount_path"`
	Id       string `json:"unique_id"`
	Metadata string `json:"metadata"`
}
