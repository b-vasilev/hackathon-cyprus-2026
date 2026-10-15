# operations
Goal: explain how to observe, troubleshoot and safely operate the product.
Use: health endpoints, logs, metrics/tracing code, error handling, retry policies, operational docs and recovery code.
Prepare: health signals, common supported failures, diagnostic entry points, restart/recovery behavior and documented backup or maintenance procedures. Identify operational gaps.
Select: operational claims, relevant components and failure/recovery scenarios.
Visual: a failure/recovery or operational dependency view only when the canonical model supports it.
Missing evidence: never invent dashboards, on-call contacts, alerts, SLOs, SLAs, retry guarantees or successful restore tests. A health endpoint is not a reliability guarantee.
