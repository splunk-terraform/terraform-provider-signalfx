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

func TestGetNameFromChartColorsByIndex(t *testing.T) {
	name, err := getNameFromChartColorsByIndex(4)
	assert.Equal(t, "dark_orange", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromChartColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestGetHexFromChartColorsByName(t *testing.T) {
	hex, err := getHexFromChartColorsByName("cerise")
	assert.Equal(t, "#e9008a", hex, "Expected color hex")
	assert.NoError(t, err, "Expected no error for known color")

	hex, err = getHexFromChartColorsByName("fart")
	assert.Equal(t, "", hex, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestGetNameFromChartColorsByHex(t *testing.T) {
	name, err := getNameFromChartColorsByHex("#bd468d")
	assert.Equal(t, "magenta", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known hex")

	name, err = getHexFromChartColorsByName("#f00f00")
	assert.Equal(t, "", name, "Expected empty string for missing hex")
	assert.Error(t, err, "Expected error for missing color hex")
}

func TestGetNameFromPaletteColorsByIndex(t *testing.T) {
	name, err := getNameFromPaletteColorsByIndex(2)
	assert.Equal(t, "azure", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromPaletteColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestGetNameFromFullPaletteColorsByIndex(t *testing.T) {
	name, err := getNameFromFullPaletteColorsByIndex(16)
	assert.Equal(t, "red", name, "Expected color name")
	assert.NoError(t, err, "Expected no error for known color")

	name, err = getNameFromPaletteColorsByIndex(44)
	assert.Equal(t, "", name, "Expected empty string for missing index")
	assert.Error(t, err, "Expected error for missing color index")
}

func TestSendRequestFail(t *testing.T) {
	// Client will fail to send due to invalid URL
	status_code, body, err := sendRequest("GET", "", "token", nil)
	assert.Equal(t, -1, status_code)
	assert.Nil(t, body)
	assert.Contains(t, err.Error(), "Failed sending GET request")
}

func TestValidateSignalfxRelativeTimeMinutes(t *testing.T) {
	_, errors := validateSignalfxRelativeTime("-5m", "time_range")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSignalfxRelativeTimeHours(t *testing.T) {
	_, errors := validateSignalfxRelativeTime("-5h", "time_range")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSignalfxRelativeTimeDays(t *testing.T) {
	_, errors := validateSignalfxRelativeTime("-5d", "time_range")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSignalfxRelativeTimeWeeks(t *testing.T) {
	_, errors := validateSignalfxRelativeTime("-5w", "time_range")
	assert.Equal(t, 0, len(errors))
}

func TestValidateSignalfxRelativeTimeNotAllowed(t *testing.T) {
	_, errors := validateSignalfxRelativeTime("-5M", "time_range")
	assert.Equal(t, 1, len(errors))
}

func TestConversionSignalfxRelativeTimeIntoMs(t *testing.T) {
	ms, err := fromRangeToMilliSeconds("-15m")
	assert.Equal(t, 900000, ms)
	assert.Nil(t, err)
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

func TestFlattenStringSliceToSet(t *testing.T) {
	set := flattenStringSliceToSet([]string{"a", "b"})
	assert.Equal(t, 2, set.Len(), "Set missing arguments")

	setWithEmptyStrings := flattenStringSliceToSet([]string{"a", "", "b"})
	assert.Equal(t, 2, setWithEmptyStrings.Len(), "Set missing arguments")
}
