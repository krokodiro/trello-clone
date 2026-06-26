package openapi

import (
	"strings"
	"testing"
)

func TestOpenAPISpecEmbedded(t *testing.T) {
	if len(spec) == 0 {
		t.Fatal("expected embedded openapi spec")
	}
	s := string(spec)
	for _, needle := range []string{"openapi:", "/auth/register", "/workspaces", "bearerAuth"} {
		if !strings.Contains(s, needle) {
			t.Fatalf("spec missing %q", needle)
		}
	}
}

func TestSwaggerUIHTML(t *testing.T) {
	if !strings.Contains(swaggerUIHTML, "swagger-ui") {
		t.Fatal("expected swagger ui markup")
	}
}
