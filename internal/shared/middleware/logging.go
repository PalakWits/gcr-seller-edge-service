package middleware

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"adapter/internal/shared/log"
)

type LoggingConfig struct {
	MaxBodyLogSize  int
	SkipPaths       []string
	LogRequestBody  bool
	LogResponseBody bool
}

func DefaultLoggingConfig() LoggingConfig {
	return LoggingConfig{
		MaxBodyLogSize:  1024,
		SkipPaths:       []string{"/health", "/metrics"},
		LogRequestBody:  true,
		LogResponseBody: false,
	}
}

func convertFastHTTPRequest(c *fiber.Ctx) *http.Request {
	req := &http.Request{
		Method: c.Method(),
		URL: &url.URL{
			Scheme:   c.Protocol(),
			Host:     c.Hostname(),
			Path:     c.Path(),
			RawQuery: string(c.Request().URI().QueryString()),
		},
		Header:     make(http.Header),
		RemoteAddr: c.IP(),
		Host:       c.Hostname(),
	}

	c.Request().Header.VisitAll(func(key, value []byte) {
		req.Header.Set(string(key), string(value))
	})

	return req
}

func LoggingMiddleware(config ...LoggingConfig) fiber.Handler {
	cfg := DefaultLoggingConfig()
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *fiber.Ctx) error {
		path := c.Path()
		for _, skipPath := range cfg.SkipPaths {
			if path == skipPath {
				return c.Next()
			}
		}

		requestID := uuid.New().String()
		c.Locals("request_id", requestID)

		ctx := context.WithValue(c.UserContext(), "request_id", requestID)
		c.SetUserContext(ctx)

		httpReq := convertFastHTTPRequest(c)

		start := time.Now()

		var requestBody []byte
		if cfg.LogRequestBody && c.Body() != nil {
			requestBody = c.Body()
		}

		log.RequestStart(ctx, httpReq, requestBody)

		var responseStatusCode int

		err := c.Next()

		responseTime := time.Since(start)
		responseStatusCode = c.Response().StatusCode()
		responseSize := len(c.Response().Body())

		if err != nil {
			log.ErrorWithStack(ctx, err, "Request handler error")

			if fiberErr, ok := err.(*fiber.Error); ok {
				responseStatusCode = fiberErr.Code
			} else {
				responseStatusCode = fiber.StatusInternalServerError
			}
		}

		log.RequestEnd(ctx, httpReq, responseStatusCode, responseTime, responseSize)

		if cfg.LogResponseBody && responseSize > 0 && responseSize <= cfg.MaxBodyLogSize {
			log.Debugf(ctx, "Response body: %s", string(c.Response().Body()))
		}

		return err
	}
}

func RecoveryMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		defer func() {
			if r := recover(); r != nil {
				ctx := c.UserContext()
				httpReq := convertFastHTTPRequest(c)
				log.PanicLog(ctx, httpReq, r)
				c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
					"error":      "Internal server error",
					"request_id": c.Locals("request_id"),
				})
			}
		}()

		return c.Next()
	}
}

func RequestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Locals("request_id") == nil {
			requestID := uuid.New().String()
			c.Locals("request_id", requestID)

			ctx := context.WithValue(c.UserContext(), "request_id", requestID)
			c.SetUserContext(ctx)
		}

		if requestID := c.Locals("request_id"); requestID != nil {
			c.Set("X-Request-ID", requestID.(string))
		}

		return c.Next()
	}
}

type responseRecorder struct {
	io.Writer
	statusCode int
	body       *bytes.Buffer
}

func (r *responseRecorder) WriteHeader(statusCode int) {
	r.statusCode = statusCode
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.Writer.Write(b)
}
