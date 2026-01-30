package retriever

type Skill struct {
	Id        string    `json:"id"`
	SpaceId   string    `json:"space_id"`
	Trigger   string    `json:"trigger"`
	SOP       string    `json:"sop"`
	Embedding []float32 `json:"embedding"`
}
