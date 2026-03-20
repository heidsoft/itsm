package knowledge

import (
	"time"
)

// Article representing a knowledge base article
type Article struct {
	ID          int       `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	AuthorID    int       `json:"author_id"`
	TenantID    int       `json:"tenant_id"`
	IsPublished bool      `json:"is_published"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Category represents a knowledge category
type Category struct {
	Name string `json:"name"`
}
