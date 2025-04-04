package util

import (
	"fmt"
	"github.com/pennsieve/collections-service/internal/shared/logging"
	"io"
	"log/slog"
	"net/http"
)

const ApplicationJSON = "application/json"

var logger = logging.Default.With(slog.String("goPackage", "util"))

func CloseAndWarn(response *http.Response) {
	if err := response.Body.Close(); err != nil {
		logger.Warn("error closing response body",
			slog.String("method", response.Request.Method),
			slog.String("url", response.Request.URL.String()),
			slog.Any("error", err))
	}
}

func Invoke(request *http.Request) (*http.Response, error) {

	res, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("error invoking %s %s: %w", request.Method, request.URL, err)
	}
	if err := checkHTTPStatus(res); err != nil {
		// if there was an error, checkHTTPStatus read the body
		if closeError := res.Body.Close(); closeError != nil {
			logger.Warn("error closing response body from http status error",
				slog.String("method", request.Method),
				slog.String("url", request.URL.String()),
				slog.Any("error", closeError))
		}
		return nil, err
	}
	return res, nil
}

// checkHTTPStatus returns an error if 400 <= response status code < 600. Otherwise, returns nil.
// If an error is being returned, this function will consume response.Body so it should be
// called before the caller has read the body.
func checkHTTPStatus(response *http.Response) error {
	readBody := func() []byte {
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return []byte(fmt.Sprintf("<unable to read body: %s>", err.Error()))
		}
		return body
	}
	if http.StatusBadRequest <= response.StatusCode && response.StatusCode < 600 {
		responseBody := readBody()
		var displayBody string
		if len(responseBody) > 1000 {
			displayBody = fmt.Sprintf("<truncated for logging> %s", string(responseBody[:1000]))
		}
		displayBody = string(responseBody)
		errorType := "client"
		if response.StatusCode >= http.StatusInternalServerError {
			errorType = "server"
		}
		return fmt.Errorf("%s error %s calling %s %s; response body: %s",
			errorType,
			response.Status,
			response.Request.Method,
			response.Request.URL,
			displayBody)
	}
	return nil
}
