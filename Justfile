set shell := ["bash", "-euo", "pipefail", "-c"]

mod backend
mod frontend

default:
    @just --list --list-submodules

# Download Go dependencies after mise install. UI dependencies are opt-in.
setup: backend::deps

# Show startup instructions; do not start services implicitly.
dev:
    @echo 'Backend: mise exec -- just backend::run'
    @echo 'React placeholder: mise exec -- just frontend::deps && mise exec -- just frontend::dev'

fmt: backend::fmt
lint: backend::lint
check: backend::check
test: backend::test
audit: backend::audit
build: backend::build
ci: check backend::test-race audit build

# Frontend checks are separate while the UI is only a placeholder.
check-web: frontend::check
