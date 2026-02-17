package logger

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// HttpRequestKey holds the http_request field
	HttpRequestKey = "http_request"
	// HttpRequestPayload holds the http_request_payload field
	HttpRequestPayload = "http_request_payload"
	// HTTPRequestMethod holds the http_request_method field
	HTTPRequestMethod = "http_request_method"
	// HTTPRequestURL holds the http_request_URL field
	HTTPRequestURL = "http_request_url"
	// ServerNameKey holds the server_name field
	ServerNameKey = "server_name"
	// ServiceKey holds the service field
	ServiceKey = "service"
	// LatencyKeySecond holds the latency field
	LatencyKeySecond = "latency_sec"
	// LatencyKeyMillisecond holds the latency field
	LatencyKeyMillisecond = "latency_ms"
	// StatusCodeKey holds the status_code field
	StatusCodeKey = "status_code"
	// SchoolIDKey holds the school_id field
	SchoolIDKey = "school_id"
	// UserIDKey holds the user_id field
	UserIDKey = "user_id"
)

// Logger wraps logrus with additional context capabilities
type Logger struct {
	ec        *logrus.Entry // ec: entry clone
	entry     *logrus.Entry
	skipPaths []string
}

// WithError creates an entry from the logger and adds an error to it
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		ec:        l.ec.WithError(err),
		entry:     l.entry,
		skipPaths: l.skipPaths,
	}
}

// WithField creates an entry from the logger and adds a field to it
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return &Logger{
		ec:        l.ec.WithField(key, value),
		entry:     l.entry,
		skipPaths: l.skipPaths,
	}
}

// WithFields adds multiple fields to the logger
func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	return &Logger{
		ec:        l.ec.WithFields(fields),
		entry:     l.entry,
		skipPaths: l.skipPaths,
	}
}

// Error logs at error level
func (l *Logger) Error(args ...interface{}) {
	l.ec.Error(args...)
	l.reset()
}

// Warn logs at warning level
func (l *Logger) Warn(args ...interface{}) {
	l.ec.Warn(args...)
	l.reset()
}

// Info logs at info level
func (l *Logger) Info(args ...interface{}) {
	l.ec.Info(args...)
	l.reset()
}

// Debug logs at debug level
func (l *Logger) Debug(args ...interface{}) {
	l.ec.Debug(args...)
	l.reset()
}

// Fatal logs at fatal level and exits
func (l *Logger) Fatal(args ...interface{}) {
	l.ec.Fatal(args...)
}

// Errorf logs at error level with format
func (l *Logger) Errorf(format string, args ...interface{}) {
	l.ec.Errorf(format, args...)
	l.reset()
}

// Warnf logs at warning level with format
func (l *Logger) Warnf(format string, args ...interface{}) {
	l.ec.Warnf(format, args...)
	l.reset()
}

// Infof logs at info level with format
func (l *Logger) Infof(format string, args ...interface{}) {
	l.ec.Infof(format, args...)
	l.reset()
}

// Debugf logs at debug level with format
func (l *Logger) Debugf(format string, args ...interface{}) {
	l.ec.Debugf(format, args...)
	l.reset()
}

// Fatalf logs at fatal level with format and exits
func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.ec.Fatalf(format, args...)
}

// Println logs at info level (compatibility method)
func (l *Logger) Println(args ...interface{}) {
	l.ec.Info(args...)
	l.reset()
}

// Printf logs at info level with format (compatibility method)
func (l *Logger) Printf(format string, args ...interface{}) {
	l.ec.Infof(format, args...)
	l.reset()
}

// reset resets the clone entry to the base entry
func (l *Logger) reset() {
	l.ec = l.entry.Dup()
}

// WithRequest adds HTTP request context to the log entry
func (l *Logger) WithRequest(r *http.Request) *Logger {
	if r == nil {
		return l
	}

	reqURI := r.RequestURI
	if reqURI == "" {
		reqURI = r.URL.RequestURI()
	}

	newLogger := l.WithField(HttpRequestKey, fmt.Sprintf("%s %s", r.Method, reqURI))

	// Skip body dump for multipart or file uploads
	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "multipart/form-data") {
		return newLogger
	}

	// Skip body dump for configured paths
	for _, path := range l.skipPaths {
		if strings.Contains(r.URL.Path, path) {
			return newLogger
		}
	}

	// Dump request body
	if r.Body != nil && r.Body != http.NoBody {
		var buf bytes.Buffer
		tee := io.TeeReader(r.Body, &buf)
		body, _ := io.ReadAll(tee)
		r.Body = io.NopCloser(&buf)

		if len(body) != 0 && len(body) < 10000 { // Limit body size
			newLogger = newLogger.WithField(HttpRequestPayload, string(body))
		}
	}

	return newLogger
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level string) error {
	logrusLevel, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	l.ec.Logger.SetLevel(logrusLevel)
	return nil
}

// RecoverMiddleware returns a recovery middleware for Gin-like frameworks
func (l *Logger) RecoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rval := recover(); rval != nil {
				debug.PrintStack()
				l.WithRequest(r).Errorf("panic recovered: %v", rval)
				w.Header().Set("Content-Type", "application/json")
				http.Error(w, `{"error": "internal server error"}`, 500)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// LatencyMiddleware logs request latency
func (l *Logger) LatencyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		l.WithField(HTTPRequestMethod, r.Method).
			WithField(HTTPRequestURL, r.URL.String()).
			WithField(LatencyKeyMillisecond, duration.Milliseconds()).
			Info("request completed")
	})
}

// New creates a new logger instance
func New(serviceName string) *Logger {
	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(&logrus.TextFormatter{
		DisableQuote:     true,
		DisableSorting:   true,
		QuoteEmptyFields: true,
		FullTimestamp:    true,
		TimestampFormat:  "2006-01-02 15:04:05",
	})

	entry := logrus.NewEntry(log)
	if serviceName != "" {
		entry = entry.WithField(ServiceKey, serviceName)
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		entry = entry.WithField(ServerNameKey, hostname)
	}

	return &Logger{
		entry:     entry,
		ec:        entry.Dup(),
		skipPaths: []string{},
	}
}

// NewJSON creates a new logger instance with JSON formatting (good for production)
func NewJSON(serviceName string) *Logger {
	log := logrus.New()
	log.Out = os.Stdout
	log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	entry := logrus.NewEntry(log)
	if serviceName != "" {
		entry = entry.WithField(ServiceKey, serviceName)
	}

	hostname, _ := os.Hostname()
	if hostname != "" {
		entry = entry.WithField(ServerNameKey, hostname)
	}

	return &Logger{
		entry:     entry,
		ec:        entry.Dup(),
		skipPaths: []string{},
	}
}

// SetSkipPaths sets paths to skip request body logging
func (l *Logger) SetSkipPaths(paths []string) {
	l.skipPaths = paths
}
