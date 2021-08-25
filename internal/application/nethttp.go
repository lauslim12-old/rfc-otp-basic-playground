package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

// SuccessResponse is used to handle successful requests.
type SuccessResponse struct {
	Status  string      `json:"status"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewSuccessResponse is used to create a default, new success response.
func NewSuccessResponse(code int, message string, data interface{}) *SuccessResponse {
	return &SuccessResponse{
		Status:  "success",
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// FailureResponse is used to handle failed requests.
type FailureResponse struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// NewFailureResponse is used to create a default, new failure response.
func NewFailureResponse(code int, message string) *FailureResponse {
	return &FailureResponse{
		Status:  "fail",
		Code:    code,
		Message: message,
	}
}

// Utility function to send succesful response.
func sendSuccessResponse(w http.ResponseWriter, successResponse *SuccessResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(successResponse.Code)
	json.NewEncoder(w).Encode(successResponse)
}

// Utility function to send failure response.
func sendFailureResponse(w http.ResponseWriter, failureResponse *FailureResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(failureResponse.Code)
	json.NewEncoder(w).Encode(failureResponse)
}

// Utility function to decode a JSON request body.
func decodeJSONBody(w http.ResponseWriter, r *http.Request, dst interface{}) *FailureResponse {
	// Check if Header is 'Content-Type: application/json'.
	if r.Header.Get("Content-Type") != "application/json" {
		return NewFailureResponse(http.StatusUnsupportedMediaType, "The 'Content-Type' header is not 'application/json'!")
	}

	// Parse body, and set max bytes reader (512 bytes).
	r.Body = http.MaxBytesReader(w, r.Body, 512)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError

		switch {
		// Handle syntax errors.
		case errors.As(err, &syntaxError):
			errorMessage := fmt.Sprintf("Request body contains a badly formatted JSON at position %d!", syntaxError.Offset)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle unexpected EOFs.
		case errors.Is(err, io.ErrUnexpectedEOF):
			errorMessage := "Request body contains a badly-formed JSON!"
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle wrong data-type in request body.
		case errors.As(err, &unmarshalTypeError):
			errorMessage := fmt.Sprintf("Request body contains an invalid value for the %q field at position %d!", unmarshalTypeError.Field, unmarshalTypeError.Offset)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle unknown fields.
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			errorMessage := fmt.Sprintf("Request body contains unknown field '%s'!", fieldName)
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle empty request body.
		case errors.Is(err, io.EOF):
			errorMessage := "Request body must not be empty!"
			return NewFailureResponse(http.StatusBadRequest, errorMessage)

		// Handle too large body.
		case err.Error() == "http: request body too large":
			errorMessage := "Request body must not be larger than 512 bytes!"
			return NewFailureResponse(http.StatusRequestEntityTooLarge, errorMessage)

		// Handle other errors.
		default:
			return NewFailureResponse(http.StatusInternalServerError, err.Error())
		}
	}
	defer r.Body.Close()

	// Handle if client tries to send more than one JSON object.
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		errorMessage := "Request body must only contain a single JSON object!"
		return NewFailureResponse(http.StatusBadRequest, errorMessage)
	}

	// If everything goes well, don't return anything.
	return nil
}

// Configure is used to configure the application (server is initialized in 'main').
func Configure() http.Handler {
	// Create a Chi instance.
	r := chi.NewRouter()

	// Set up Chi's natural middlewares.
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Set up custom middleware.
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Application-Name", "Fullstack OTP")
			w.Header().Add("Server", "net/http")
			next.ServeHTTP(w, r)
		})
	})

	// Group routes.
	r.Route("/api/v1", func(r chi.Router) {
		// Sample GET request.
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			res := NewSuccessResponse(http.StatusOK, "Welcome to 'net/http' API!", nil)
			sendSuccessResponse(w, res)
		})

		// Declare method not allowed as a fallback.
		r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
			errorMessage := fmt.Sprintf("Method '%s' is not allowed in this route!", r.Method)
			res := NewFailureResponse(http.StatusMethodNotAllowed, errorMessage)
			sendFailureResponse(w, res)
		})

		// Declare 404 every time a request reaches here.
		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			errorMessage := fmt.Sprintf("Route '%s' with method '%s' does not exist in this server!", r.RequestURI, r.Method)
			res := NewFailureResponse(http.StatusNotFound, errorMessage)
			sendFailureResponse(w, res)
		})
	})

	// Return our configured infrastructure.
	return r
}
