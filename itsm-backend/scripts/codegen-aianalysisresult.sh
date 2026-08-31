#!/bin/bash
set -e
cd "$(dirname "$0")/.."
export PATH="$PATH:$(go env GOPATH)/bin"

echo "[1/4] Cleanup old aianalysisresult files..."
rm -rf ent/aianalysisresult
rm -f ent/ai_analysis_result*.go
echo "  done"

echo "[2/4] Run ent generate..."
START=$(date +%s)
ent generate ./ent/schema
END=$(date +%s)
echo "  took $((END-START))s, exit $?"

echo "[3/4] Verify output..."
ls ent/aianalysisresult/ 2>/dev/null | head -10 && echo "  subdir exists" || echo "  no subdir"
ls ent/ai_analysis_result*.go 2>/dev/null && echo "  root files exist" || echo "  no root files"

echo "[4/4] Compile check..."
go build ./ent/aianalysisresult/... 2>&1 && echo "✅ compile OK" || {
  echo "❌ compile failed — trying full ent package..."
  go build -p 1 ./ent/... 2>&1 | head -10
}
