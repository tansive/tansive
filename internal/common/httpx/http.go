// Package httpx provides HTTP request/response handling utilities and middleware.
// It includes support for JSON responses, error handling, request parsing,
// and streaming responses. The package requires valid http.ResponseWriter
// implementations for response handling.
package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/tansive/tansive/internal/common/apperrors"
)

// GetRequestData parses JSON request body into the provided data structure.
// Only supports POST and PUT methods. Returns error if request body is empty
// or cannot be parsed.
func GetRequestData(r *http.Request, data any) error {
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		return ErrReqMethodNotSupported()
	}
	if r.Body == nil {
		log.Ctx(r.Context()).Error().Msg("Empty request body")
		return ErrUnableToParseReqData()
	}
	if err := json.NewDecoder(r.Body).Decode(data); err != nil {
		return ErrUnableToParseReqData()
	}
	return nil
}

// WriteChunksFunc defines a function type for writing chunked response data.
type WriteChunksFunc func(w http.ResponseWriter) error

// Response represents an HTTP response with configurable status code,
// content type, and optional chunked transfer encoding.
type Response struct {
	StatusCode  int
	Location    string
	Response    any
	ContentType string
	Chunked     bool
	WriteChunks WriteChunksFunc
}

// RequestHandler defines a function type for handling HTTP requests.
type RequestHandler func(r *http.Request) (*Response, error)

// WrapHttpRsp wraps a RequestHandler to provide standardized HTTP response handling,
// including error handling and content type management.
func WrapHttpRsp(handler RequestHandler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsp, err := handler(r)
		if err != nil {
			if httperror, ok := err.(*Error); ok {
				httperror.Send(w)
			} else if appErr, ok := err.(apperrors.Error); ok {
				statusCode := appErr.StatusCode()
				if statusCode == 0 {
					statusCode = http.StatusInternalServerError
				}
				httperror := &Error{
					StatusCode:  statusCode,
					Description: appErr.ErrorAll(),
				}
				httperror.Send(w)
			} else {
				ErrApplicationError(err.Error()).Send(w)
			}
			return
		}
		if rsp == nil {
			ErrApplicationError().Send(w)
			return
		}
		if rsp.Chunked {
			if rsp.WriteChunks == nil {
				ErrApplicationError("unable to write chunks").Send(w)
				return
			}
			w.Header().Set("Content-Type", rsp.ContentType)
			w.Header().Set("Transfer-Encoding", "chunked")
			w.WriteHeader(rsp.StatusCode)
			if err := rsp.WriteChunks(w); err != nil {
				log.Ctx(r.Context()).Error().Err(err).Msg("Error writing chunk")
				return
			}
			return
		}

		if rsp.ContentType == "" {
			rsp.ContentType = "application/json"
		}
		var location []string
		if rsp.Location != "" {
			location = append(location, rsp.Location)
		}
		switch rsp.ContentType {
		case "application/json":
			SendJsonRsp(r.Context(), w, rsp.StatusCode, rsp.Response, location...)
		case "text/plain":
			w.Header().Set("Content-Type", "text/plain")
			if rsp.StatusCode == http.StatusCreated && len(location) > 0 {
				w.Header().Set("Location", location[0])
			}
			w.WriteHeader(rsp.StatusCode)
			w.Write([]byte(rsp.Response.(string)))
		default:
			ErrApplicationError("unsupported response type").Send(w)
		}
	})
}

// StreamResponse represents a streaming response configuration with
// status code, content type, and chunk writing function.
type StreamResponse struct {
	StatusCode  int
	ContentType string
	WriteChunk  func(w http.ResponseWriter) error
}

// StreamHandler defines a function type for handling streaming HTTP responses.
type StreamHandler func(r *http.Request) (*StreamResponse, error)

// WrapStreamHandler wraps a StreamHandler to provide standardized streaming
// response handling with proper error handling and chunked transfer encoding.
func WrapStreamHandler(handler StreamHandler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rsp, err := handler(r)
		if err != nil {
			if httperror, ok := err.(*Error); ok {
				httperror.Send(w)
			} else if appErr, ok := err.(apperrors.Error); ok {
				statusCode := appErr.StatusCode()
				if statusCode == 0 {
					statusCode = http.StatusInternalServerError
				}
				httperror := &Error{
					StatusCode:  statusCode,
					Description: appErr.ErrorAll(),
				}
				httperror.Send(w)
			} else {
				ErrApplicationError(err.Error()).Send(w)
			}
			return
		}
		if rsp == nil {
			ErrApplicationError().Send(w)
			return
		}

		w.Header().Set("Content-Type", rsp.ContentType)
		w.Header().Set("Transfer-Encoding", "chunked")
		w.WriteHeader(rsp.StatusCode)

		flusher, ok := w.(http.Flusher)
		if !ok {
			ErrApplicationError("streaming not supported").Send(w)
			return
		}

		for {
			if err := rsp.WriteChunk(w); err != nil {
				log.Ctx(r.Context()).Error().Err(err).Msg("Error writing chunk")
				return
			}
			flusher.Flush()
		}
	})
}
