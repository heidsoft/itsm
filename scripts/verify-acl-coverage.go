// verify-acl-coverage.go
//
// Go static analysis tool to verify that all non-public routes in router.go
// and *_routes.go files have RequirePermission middleware declarations.
//
// Usage:
//
//	go run scripts/verify-acl-coverage.go
//
// Exit codes:
//	0 - all routes protected or explicitly documented as public
//	1 - one or more routes missing RequirePermission
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Routes that are intentionally public (no auth required)
var knownPublic = map[string]bool{
	"/api/v1/health":        true,
	"/api/v1/healthz":       true,
	"/api/v1/readyz":        true,
	"/api/v1/version":       true,
	"/api/v1/auth/login":    true,
	"/api/v1/refresh-token":  true,
	"/api/v1/csrf-token":    true,
	"/api/v1/readiness/ga":  true,
}

// RouteInfo records a single route found during parsing
type RouteInfo struct {
	Method    string
	Path      string
	Line      int
	File      string
	Protected bool
}

func main() {
	routerDir := "itsm-backend/router"
	if len(os.Args) > 1 {
		routerDir = os.Args[1]
	}

	entries, err := os.ReadDir(routerDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "verify-acl-coverage: cannot read %s: %v\n", routerDir, err)
		os.Exit(1)
	}

	var unprotected []RouteInfo

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
			continue
		}
		path := filepath.Join(routerDir, e.Name())
		routes := parseFile(path)
		for _, r := range routes {
			if knownPublic[r.Path] {
				r.Protected = true // explicitly public
			}
			if !r.Protected {
				unprotected = append(unprotected, r)
			}
		}
	}

	if len(unprotected) > 0 {
		fmt.Fprintf(os.Stderr, "verify-acl-coverage: %d non-public routes missing RequirePermission:\n", len(unprotected))
		for _, r := range unprotected {
			fmt.Fprintf(os.Stderr, "  %s %s  (%s:%d)\n", r.Method, r.Path, r.File, r.Line)
		}
		os.Exit(1)
	}

	fmt.Println("verify-acl-coverage: all routes protected (100% coverage)")
	os.Exit(0)
}

// ---------------------------------------------------------------------------
// Parser
// ---------------------------------------------------------------------------

var routeRE = regexp.MustCompile(`^\s*(?:\w+\s*\.)?(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\s*\(\s*"([^"]+)"`)

var groupDeclRE = regexp.MustCompile(`^\s*(\w+)\s*:=\s*(?:r|auth|tenant|msp|public)\s*\.\s*Group\s*\(\s*"([^"]+)"`)

var permRE = regexp.MustCompile(`RequirePermission\s*\(\s*"[^"]+"\s*,\s*"[^"]+"\s*\)`)

func parseFile(path string) []RouteInfo {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	lines := strings.Split(string(data), "\n")
	var routes []RouteInfo
	groupStack := []string{"/api/v1"}

	for i, raw := range lines {
		l := strings.TrimSpace(raw)
		lineNo := i + 1

		if l == "" || strings.HasPrefix(l, "//") || strings.HasPrefix(l, "/*") {
			continue
		}

		// Track brace depth for group scope
		for _, c := range l {
			if c == '{' {
			} else if c == '}' {
			}
		}

		// Named group declaration: auth := r.Group("/api/v1")
		if m := groupDeclRE.FindStringSubmatch(l); len(m) > 0 {
			groupStack = append(groupStack, m[2])
			continue
		}

		// Closing brace at start of line → pop group scope
		if strings.HasPrefix(l, "}") && len(groupStack) > 1 {
			groupStack = groupStack[:len(groupStack)-1]
			continue
		}

		// Route registration
		m := routeRE.FindStringSubmatch(l)
		if m == nil {
			continue
		}
		method := m[1]
		routePath := m[2]

		// Build full path
		prefix := groupStack[len(groupStack)-1]
		if !strings.HasPrefix(routePath, "/") {
			routePath = "/" + routePath
		}
		fullPath := prefix + routePath

		// Find rest of line after path to look for RequirePermission
		permIdx := strings.Index(l, `"`+routePath+`"`)
		rest := ""
		if permIdx > 0 {
			rest = l[permIdx+len(routePath)+2:]
		}
		hasPerm := permRE.MatchString(rest)

		routes = append(routes, RouteInfo{
			Method:    method,
			Path:      fullPath,
			Line:      lineNo,
			File:      filepath.Base(path),
			Protected: hasPerm,
		})
	}

	return routes
}
