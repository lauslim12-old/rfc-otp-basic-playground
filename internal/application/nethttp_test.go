package application

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
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
