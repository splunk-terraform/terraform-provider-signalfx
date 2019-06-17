package signalfx

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSendRequestSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `Test Response`)
	}))
	defer server.Close()

	status_code, body, err := sendRequest("GET", server.URL, "token", nil)
	assert.Equal(t, 200, status_code)
	assert.Equal(t, "Test Response\n", string(body))
	assert.Nil(t, err)
}

func TestSendRequestResponseNotFound(t *testing.T) {
	// Handler returns 404 page not found
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	status_code, body, err := sendRequest("POST", server.URL, "token", nil)
	assert.Equal(t, 404, status_code)
	assert.Contains(t, string(body), "page not found")
	assert.Nil(t, err)
}

func TestSendRequestFail(t *testing.T) {
	// Client will fail to send due to invalid URL
	status_code, body, err := sendRequest("GET", "", "token", nil)
	assert.Equal(t, -1, status_code)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "Failed sending GET request")
}

func TestValidateSortByAscending(t *testing.T) {
	_, errors := validateSortBy("+foo", "sort_by")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSortByDescending(t *testing.T) {
	_, errors := validateSortBy("-foo", "sort_by")
	assert.Equal(t, 0, len(errors))
}

func TestValidateFullPaletteColors(t *testing.T) {
	_, errors := validateFullPaletteColors("chartreuse", "color_theme")
	assert.Equal(t, 0, len(errors))
}

func TestValidateFullPaletteColorsFail(t *testing.T) {
	_, errors := validateFullPaletteColors("fart", "color_theme")
	assert.Equal(t, 1, len(errors))
}

func TestValidateSortByNoDirection(t *testing.T) {
	_, errors := validateSortBy("foo", "sort_by")
	assert.Equal(t, 1, len(errors))
}

func TestBuildURL(t *testing.T) {
	u, error := buildURL("https://www.example.com", "/v2/chart", map[string]string{})
	assert.NoError(t, error)
	assert.Equal(t, "https://www.example.com/v2/chart", u)
}

func TestBuildURLWithParams(t *testing.T) {
	u, error := buildURL("https://www.example.com", "/v2/chart", map[string]string{"foo": "bar"})
	assert.NoError(t, error)
	assert.Equal(t, "https://www.example.com/v2/chart?foo=bar", u)
}

func TestBuildAppURL(t *testing.T) {
	u, error := buildAppURL("https://www.example.com", "/chart/abc123")
	assert.NoError(t, error)
	assert.Equal(t, "https://www.example.com/#/chart/abc123", u)
}
