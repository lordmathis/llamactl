package server

type OpenAIListInstancesResponse struct {
	Object string           `json:"object"`
	Data   []OpenAIInstance `json:"data"`
}

type OpenAIInstance struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}
