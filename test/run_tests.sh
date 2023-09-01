#!/bin/bash

set -o allexport errexit

DEFAULT_TEST_ENV_FILE="test.env"
DEFAULT_TEST_ASTRA_API_URL="https://api.test.cloud.datastax.com"
DEFAULT_TEST_ASTRA_STREAMING_API_URL="https://api.staging.streaming.datastax.com"

SCRIPT_PATH=$(dirname -- "$0")

setup_env() {
  if [ -z "$TEST_ENV_FILE" ]; then
    TEST_ENV_FILE="${SCRIPT_PATH}/${DEFAULT_TEST_ENV_FILE}"
  fi
  if [ -f "$TEST_ENV_FILE" ]; then
    echo "loading config from file $TEST_ENV_FILE"
    source "$TEST_ENV_FILE"
  else
    echo "file '$TEST_ENV_FILE' not found, some tests may be skipped"
  fi

  if [ -z "$ASTRA_API_TOKEN" ]; then \
      echo "environment variable ASTRA_API_TOKEN must be set for acceptance tests"
      exit 1
  fi

  if [ -z "$ASTRA_API_URL" ]; then
    ASTRA_API_URL="$DEFAULT_TEST_ASTRA_API_URL"
  fi
  if [ -z "$ASTRA_STREAMING_API_URL" ]; then
    ASTRA_STREAMING_API_URL="$DEFAULT_TEST_ASTRA_STREAMING_API_URL"
  fi

  if [ -z "$ASTRA_TEST_TIMEOUT" ]; then
    ASTRA_TEST_TIMEOUT="15m"
  fi
}

run_tests() {
  echo "Running tests..."
  TF_ACC=1 # Environment variable to enable terraform acceptance tests
  go test ./internal/provider -v $TESTARGS -timeout "$ASTRA_TEST_TIMEOUT"
}

# Main execution
setup_env
run_tests
