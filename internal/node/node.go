package node

type Node struct {
	ID          string
	Title       string
	Description string
	Status      string // "active" | "archived"
	CreatedBy   string // "user" | "agent_recap" | "lore_inference"
	CreatedAt   int64
	UpdatedAt   int64
}
