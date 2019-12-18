package datalink

type SearchResults struct {
	// Number of objects that match the search query. If you use paging, `count` is either the value of the `limit` request parameter or the number of objects still undelivered after the request reached the `offset` request parameter.<br> *Note:* If you use paging, count is not the number of objects returned in the response.
	Count int32 `json:"count,omitempty"`
	// An array of data link definitions that match the request criteria.
	Results []DataLink `json:"results,omitempty"`
}
