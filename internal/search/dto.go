package search

type SearchData struct {
	Query  string              `json:"query"`
	Groups []SearchResultGroup `json:"groups"`
}
