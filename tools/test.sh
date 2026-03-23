#!/bin/bash

set -e

MODE="$1"
DEFAULT_FLAGS="-covermode=atomic -coverprofile=coverage.out"
COVERAGE_THRESHOLD="${COVERAGE_THRESHOLD:-80}"

case "$MODE" in
  race)
    echo "🔬 Running tests with race detector..."
    go test ./... $DEFAULT_FLAGS -race
    ;;
  verbose)
    echo "🗣️ Running verbose tests..."
    go test ./... $DEFAULT_FLAGS -v
    ;;
  coverage)
    echo "📊 Opening coverage report..."
    go tool cover -html=coverage.out
    ;;
  check)
    echo "📏 Checking coverage threshold (${COVERAGE_THRESHOLD}%)..."
    coverage=$(go tool cover -func=coverage.out | awk '/^total:/ {gsub("%","",$3); print $3}')
    echo "Total coverage: ${coverage}%"
    awk -v cov="$coverage" -v threshold="$COVERAGE_THRESHOLD" 'BEGIN { exit((cov + 0) < (threshold + 0) ? 1 : 0) }'
    ;;
  *)
    echo "🧪 Running standard tests..."
    go test ./... $DEFAULT_FLAGS
    ;;
esac
