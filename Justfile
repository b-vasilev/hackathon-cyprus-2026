set shell := ["bash", "-euo", "pipefail", "-c"]
set dotenv-load := true

mod backend
mod frontend
mod proxy 'services/llm-proxy'

default:
    @just --list --list-submodules

# Install the dependencies used by the API, UI and source packager.
setup: backend::deps frontend::deps

# Start the backend, wait for readiness, then start the frontend.
dev:
    #!/usr/bin/env bash
    set -euo pipefail
    set -m
    backend_pid=''
    frontend_pid=''
    cleanup() {
        trap '' INT TERM
        for pid in "$backend_pid" "$frontend_pid"; do
            if [[ -n "$pid" ]]; then
                kill -TERM -- "-$pid" 2>/dev/null || true
            fi
        done
        wait || true
    }
    trap cleanup EXIT
    trap 'exit 130' INT
    trap 'exit 143' TERM
    just backend::run &
    backend_pid=$!
    echo 'Waiting for the backend to be ready...'
    until curl --fail --silent --output /dev/null --max-time 1 http://127.0.0.1:8080/readyz; do
        if ! kill -0 "$backend_pid" 2>/dev/null; then
            wait "$backend_pid"
            exit 1
        fi
        sleep 0.2
    done
    just frontend::dev &
    frontend_pid=$!
    while kill -0 "$backend_pid" 2>/dev/null && kill -0 "$frontend_pid" 2>/dev/null; do
        sleep 1
    done
    if ! kill -0 "$backend_pid" 2>/dev/null; then
        wait "$backend_pid"
    else
        wait "$frontend_pid"
    fi

fmt: backend::fmt
lint: backend::lint
check: backend::check
test: backend::test
audit: backend::audit
build: backend::build
ci: check backend::test-race audit build

check-web: frontend::check

generate-api: backend::spec frontend::generate-api
