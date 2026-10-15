set shell := ["bash", "-euo", "pipefail", "-c"]

mod backend
mod frontend
mod proxy 'services/llm-proxy'

default:
    @just --list --list-submodules

# Install the dependencies used by the API, UI and source packager.
setup: backend::deps frontend::deps

# Start both services and stop them together on exit or failure.
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
