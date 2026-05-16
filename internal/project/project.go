package project

import "time"

const MetadataSchema = "yeast.project.v1"

type Metadata struct {
	Schema    string    `json:"schema"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewMetadata(id string, createdAt time.Time) Metadata {
	return Metadata{
		Schema:    MetadataSchema,
		ID:        id,
		CreatedAt: createdAt.UTC(),
	}
}

