package model

type ShortURLJSON struct {
	URL string `json:"url"`
}

type ShortURLJSONResult struct {
	Result string `json:"result"`
}

type URLStorage struct {
	ID          string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
