---
name: env-troubleshooting
description: Diagnoses and fixes common development environment issues, specifically Go version mismatches and NPM dependency conflicts. Invoke when startup fails due to environment or dependency errors.
---

# Environment Troubleshooting Skill

This skill provides systematic solutions for resolving common development environment startup failures, focusing on Go and Node.js/NPM ecosystems.

## 1. Go Version Mismatch
**Symptoms**:
- Error: `compile: version "go1.X" does not match go tool version "go1.Y"`
- Error: `go: cannot run *go1.X* (installed in ...) please ensure that go1.X is in your PATH`

**Diagnosis**:
The version specified in `go.mod` differs from the local `go` toolchain, or the toolchain auto-download mechanism is failing. Often caused by `GOROOT` environment variable pointing to an older system installation while `go` binary is newer.

**Solutions**:
1.  **Force Local Toolchain** (Recommended for dev):
    Tell Go to use the locally installed version regardless of `go.mod`.
    ```bash
    go env -w GOTOOLCHAIN=local
    go clean -cache
    ```
2.  **Fix GOROOT Mismatch** (Critical if Solution 1 fails):
    If `go version` shows X but `go env GOROOT` shows Y, explicit export is needed.
    ```bash
    # Check current GOROOT
    go env GOROOT
    # Find correct toolchain path (example for Mac)
    ls -d ~/go/pkg/mod/golang.org/toolchain@*
    # Set GOROOT (replace path with actual one found above)
    export GOROOT=/Users/<user>/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.6.darwin-amd64
    export PATH=$GOROOT/bin:$PATH
    ```
3.  **Match go.mod**:
    Update `go.mod` to match your local version (if permissible).
    ```bash
    go mod edit -go=1.25.6  # Replace with your `go version`
    go mod tidy
    ```

## 2. NPM Dependency Conflicts
**Symptoms**:
- Error: `ERESOLVE unable to resolve dependency tree`
- Error: `Conflicting peer dependency: ...`

**Diagnosis**:
Strict peer dependency checks in npm v7+ are failing because libraries require different versions of the same package (e.g., React or Ant Design).

**Solutions**:
1.  **Legacy Peer Deps** (Quick Fix):
    Bypass strict peer dependency checks. Safe for dev, but check for runtime errors.
    ```bash
    npm install --legacy-peer-deps
    ```
2.  **Force Install** (Aggressive):
    ```bash
    npm install --force
    ```
3.  **Resolution Override** (Best Practice):
    Add a `overrides` (npm) or `resolutions` (yarn) section to `package.json` to force a specific version.
    ```json
    "overrides": {
      "antd": "^5.0.0"
    }
    ```

## 3. Port Conflicts
**Symptoms**:
- Error: `listen tcp :8090: bind: address already in use`

**Solutions**:
1.  **Find and Kill**:
    ```bash
    lsof -i :8090
    kill -9 <PID>
    ```
2.  **Change Port**:
    Update `config.yaml` or `.env` to use a free port.

## Workflow
1.  **Analyze Error Log**: Identify keywords (`compile version`, `ERESOLVE`, `bind`).
2.  **Apply Fix**: Use the corresponding solution above.
3.  **Clean & Rebuild**: Always run `go clean -cache` or `rm -rf node_modules` after major env changes.
4.  **Verify**: Start the service and check health endpoints.
