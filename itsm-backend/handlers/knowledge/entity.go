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
	AuthorID    int       `json:"authorId"`
	TenantID    int       `json:"tenantId"`
	IsPublished bool      `json:"isPublished"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Category represents a knowledge category
type Category struct {
	Name string `json:"name"`
}
