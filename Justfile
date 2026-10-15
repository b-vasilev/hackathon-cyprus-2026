set shell := ["bash", "-euo", "pipefail", "-c"]

mod backend
mod frontend
mod proxy 'services/llm-proxy'

default:
    @just --list --list-submodules

# Install the dependencies used by the API, UI and source packager.
setup: backend::deps frontend::deps

# Show startup instructions; do not start services implicitly.
dev:
    @echo 'Backend: mise exec -- just backend::run'
    @echo 'Frontend: mise exec -- just frontend::dev'

fmt: backend::fmt
lint: backend::lint
check: backend::check
test: backend::test
audit: backend::audit
build: backend::build
ci: check backend::test-race audit build

check-web: frontend::check

generate-api: backend::spec frontend::generate-api
