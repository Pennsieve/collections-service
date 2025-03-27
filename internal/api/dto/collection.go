package dto

type CreateCollectionRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	DOIs        []string `json:"dois"`
}

type CollectionResponse struct {
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Size        int    `json:"size"`
}
