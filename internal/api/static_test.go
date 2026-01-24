package api

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestStaticFileServing tests that static files are served correctly
func TestStaticFileServing(t *testing.T) {
	// Create handler with mock service
	mockService := &MockSessionService{}
	handler := NewHandler(mockService)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantContent string
	}{
		{
			name:        "index.html is served",
			path:        "/",
			wantStatus:  http.StatusOK,
			wantContent: "<div id=\"root\">",
		},
		{
			name:        "JavaScript file is served",
			path:        "/assets/index-",
			wantStatus:  http.StatusOK,
			wantContent: "", // Just check it returns OK
		},
		{
			name:        "SVG file is served",
			path:        "/vite.svg",
			wantStatus:  http.StatusOK,
			wantContent: "<svg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			if tt.wantContent != "" {
				body := rec.Body.String()
				if !strings.Contains(body, tt.wantContent) {
					t.Errorf("response body does not contain %q", tt.wantContent)
				}
			}
		})
	}
}

// TestSPARouting tests that SPA routing works correctly
func TestSPARouting(t *testing.T) {
	mockService := &MockSessionService{}
	handler := NewHandler(mockService)

	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantContent string
	}{
		{
			name:        "root path returns index.html",
			path:        "/",
			wantStatus:  http.StatusOK,
			wantContent: "<div id=\"root\">",
		},
		{
			name:        "projects path returns index.html",
			path:        "/projects/my-project",
			wantStatus:  http.StatusOK,
			wantContent: "<div id=\"root\">",
		},
		{
			name:        "sessions path returns index.html",
			path:        "/projects/my-project/sessions/abc123",
			wantStatus:  http.StatusOK,
			wantContent: "<div id=\"root\">",
		},
		{
			name:        "random path returns index.html",
			path:        "/random/path/that/does/not/exist",
			wantStatus:  http.StatusOK,
			wantContent: "<div id=\"root\">",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.wantContent) {
				t.Errorf("response body does not contain %q", tt.wantContent)
			}
		})
	}
}

// TestAPIEndpointsPriority tests that API endpoints take priority over static files
func TestAPIEndpointsPriority(t *testing.T) {
	mockService := &MockSessionService{
		projects: []ProjectResponse{
			{Name: "test-project", DecodedPath: "/path/to/test", SessionCount: 5},
		},
	}
	handler := NewHandler(mockService)

	tests := []struct {
		name            string
		path            string
		wantStatus      int
		wantContentType string
		wantContent     string
	}{
		{
			name:            "health endpoint returns JSON",
			path:            "/api/health",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantContent:     `"status":"ok"`,
		},
		{
			name:            "projects endpoint returns JSON",
			path:            "/api/projects",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantContent:     "test-project",
		},
		{
			name:            "sessions endpoint returns JSON",
			path:            "/api/sessions",
			wantStatus:      http.StatusOK,
			wantContentType: "application/json",
			wantContent:     `"sessions"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.wantContentType) {
				t.Errorf("got Content-Type %q, want %q", contentType, tt.wantContentType)
			}

			body := rec.Body.String()
			if !strings.Contains(body, tt.wantContent) {
				t.Errorf("response body does not contain %q, got: %s", tt.wantContent, body)
			}
		})
	}
}

// TestStaticFileContentTypes tests that correct MIME types are set
func TestStaticFileContentTypes(t *testing.T) {
	mockService := &MockSessionService{}
	handler := NewHandler(mockService)

	tests := []struct {
		name            string
		path            string
		wantContentType string
	}{
		{
			name:            "HTML has correct content type",
			path:            "/",
			wantContentType: "text/html",
		},
		{
			name:            "SVG has correct content type",
			path:            "/vite.svg",
			wantContentType: "image/svg+xml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			rec := httptest.NewRecorder()

			handler.Routes().ServeHTTP(rec, req)

			// Read the body to ensure headers are set
			io.ReadAll(rec.Body)

			contentType := rec.Header().Get("Content-Type")
			if !strings.Contains(contentType, tt.wantContentType) {
				t.Errorf("got Content-Type %q, want to contain %q", contentType, tt.wantContentType)
			}
		})
	}
}
