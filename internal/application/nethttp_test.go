package application

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func structToJSON(object interface{}) string {
	out, err := json.Marshal(object)
	if err != nil {
		log.Fatal(err.Error())
	}

	return string(out)
}

func TestGeneralHandlers(t *testing.T) {
	handler := Configure()
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	failureTests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
		expectedBody   *FailureResponse
	}{
		{
			name:           "test_failure_not_found",
			method:         http.MethodGet,
			route:          "/api/v1/404",
			expectedStatus: http.StatusNotFound,
			expectedBody:   NewFailureResponse(http.StatusNotFound, "Route '/api/v1/404' with method 'GET' does not exist in this server!"),
		},
		{
			name:           "test_failure_method_not_allowed",
			method:         http.MethodDelete,
			route:          "/api/v1",
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   NewFailureResponse(http.StatusMethodNotAllowed, "Method 'DELETE' is not allowed in this route!"),
		},
	}

	successTests := []struct {
		name           string
		method         string
		route          string
		expectedStatus int
		expectedBody   *SuccessResponse
	}{
		{
			name:           "test_health_check",
			method:         http.MethodGet,
			route:          "/api/v1",
			expectedStatus: http.StatusOK,
			expectedBody:   NewSuccessResponse(http.StatusOK, "Welcome to 'net/http' API!", nil),
		},
	}

	for _, tt := range failureTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			assert.NotNil(t, w.Body)
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}

	for _, tt := range successTests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, nil)
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)

			assert.NotNil(t, w.Body)
			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.JSONEq(t, structToJSON(tt.expectedBody), w.Body.String())
		})
	}
}

func TestDecodeJSONBody(t *testing.T) {
	handler := Configure()
	ts := httptest.NewServer(handler)
	defer ts.Close()

	tests := []struct {
		name         string
		method       string
		route        string
		input        string
		expectedBody *FailureResponse
		withHeader   bool
	}{
		{
			name:         "test_content_type",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede"}`,
			expectedBody: NewFailureResponse(http.StatusUnsupportedMediaType, "The 'Content-Type' header is not 'application/json'!"),
			withHeader:   false,
		},
		{
			name:         "test_json_bad_format",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede",examplebadinput}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains a badly formatted JSON at position 40!"),
			withHeader:   true,
		},
		{
			name:         "test_json_unexpected_eof",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede"`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains a badly-formed JSON!"),
			withHeader:   true,
		},
		{
			name:         "test_json_wrong_data_type",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":1234}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains an invalid value for the \"password\" field at position 35!"),
			withHeader:   true,
		},
		{
			name:         "test_json_unknown_field",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede","unknownAttribute":"1234"}`,
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body contains unknown field '\"unknownAttribute\"'!"),
			withHeader:   true,
		},
		{
			name:         "test_json_empty",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        "",
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body must not be empty!"),
			withHeader:   true,
		},
		{
			name:         "test_json_too_large",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede","password":"kaede"}`,
			expectedBody: NewFailureResponse(http.StatusRequestEntityTooLarge, "Request body must not be larger than 512 bytes!"),
			withHeader:   true,
		},
		{
			name:         "test_double_json",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        "{}{}",
			expectedBody: NewFailureResponse(http.StatusBadRequest, "Request body must only contain a single JSON object!"),
			withHeader:   true,
		},
		{
			name:         "test_success",
			method:       http.MethodPost,
			route:        "/api/v1",
			input:        `{"username":"kaede","password":"kaede"}`,
			expectedBody: nil,
			withHeader:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(tt.method, tt.route, strings.NewReader(tt.input))
			w := httptest.NewRecorder()
			if tt.withHeader {
				r.Header.Set("Content-Type", "application/json")
			}

			failureResponse := decodeJSONBody(w, r, &AuthRequestBody{})
			assert.JSONEq(t, structToJSON(tt.expectedBody), structToJSON(failureResponse))
		})
	}
}
