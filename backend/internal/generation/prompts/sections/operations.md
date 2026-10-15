# operations
Goal: explain signals, failure impact and supported recovery safely and concretely.
Question: How can maintainers diagnose and recover important failures, and what remains uncertain?
Inputs: canonical components/scenarios/environments, stateful/external-effect boundaries, health/log/metrics/error/retry/persistence code, configuration, tests and operational docs.
Investigation: check each important boundary's caller, implementation, state/error path, retry/reconciliation and diagnostics. Separate automatic behavior from documented manual procedures; verify recovery preconditions and duplicate/external-effect safety.
Output: healthy behavior -> diagnostic entry points -> supported failures/impact -> recovery -> remaining unknowns. Prefer a cited Markdown table: failure -> signal -> action -> safety condition -> result check. Use a distinct supported state/dependency diagram only when it clarifies; keep full scenario references in scenarios.
Evidence/gaps: a health response is not readiness/end-to-end correctness; timeout does not prove an external effect failed; retry is not resume. Recovery actions need trigger, safety condition and verification or an explicit gap. Distinguish implemented/configured/documented/observed facts.
Exclusions: no invented alerts, dashboards, on-call, SLOs, backups, restore success, idempotency, failover or production guarantees. Setup belongs in development; topology in runtime.
Acceptance: all important state/effect boundaries are accounted for; failed/partial/unknown outcomes differ; no dangerous assumption is disguised as a command. Include only when useful supported operational content exists.
