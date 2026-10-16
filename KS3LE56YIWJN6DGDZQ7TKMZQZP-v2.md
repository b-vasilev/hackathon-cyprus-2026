# Antares BI analysis and migration

**Tagline:** BI assessment, documentation and migration through durable jobs and artifact-driven conversion.

Antares combines a Go API, web/desktop and command-line clients with a separately packaged .NET/Python BI migration converter. The application persists analysis work in SQLite, runs background workers and stores local artifacts; conversion uses local process/file contracts, deterministic processing and selective LLM enrichment. Representative flows cover analysis acceptance, safe upload, BRD generation, Power BI cache export and CI report comparison. Repository overview coverage is distinguished from verified runtime contracts; test sources were inspected, not executed.

**Product ID:** KS3LE56YIWJN6DGDZQ7TKMZQZP

**Passport revision:** RBOWFJFS4UH4PHSZSAQ3NHS35V

**Version:** 2

**Generated at:** 2026-10-15T22:11:57.461733Z

**Published at:** 2026-10-15T22:12:36.083038Z

**Generation run:** WWRRBUP2K7ZI32NSH5NMUJT3ZA

**Schema version:** 1

**Document version:** 2

## Product overview

**Chapter ID:** overview

**Question:** What is Antares, who uses it, what results does it deliver, and where does its responsibility end?

Antares is a BI assessment, documentation and migration product. Users work through web or desktop clients, while users and scripts can use the CLI to submit analysis inputs, check status and retrieve reports.

Documented input coverage includes Tableau workbooks, Tableau Server and Cloud, Power BI PBIX files or PBIP ZIP archives, and Fabric workspaces. Documented targets include Microsoft Fabric, Databricks AI/BI and beta Sigma. This is documented coverage, not endpoint-by-endpoint verification.

The product supports asynchronous analysis and reports, generated documentation, converted artifacts and target deployment. Report comparison provides supporting validation tooling. Analysis acceptance is not completion: an accepted request creates pending work for background execution rather than proving that results are ready.

The application coordinates work and delegates conversion to separately packaged tooling. Local artifact generation and remote deployment are distinct outcomes. Databricks AI/BI generation produces local dashboard and view bundles; their existence does not establish successful deployment. Fabric deployment requires completed Power BI conversion and separately creates remote semantic-model and report items, with asynchronous operation handling where required.

These contracts describe supported behavior, not observed successful runs. The following chapter identifies source ownership and the connections between the application and conversion tooling.

**Claims:**

- arch-app-purpose
- arch-web-contract
- arch-claim-cli
- passport-claim-analysis-acceptance
- passport-claim-conversion-flow
- passport-claim-local-bundles
- arch-fabric-contract
- passport-claim-validation-flow

## Repositories and connections

**Chapter ID:** repositories

**Question:** Which repositories own which responsibilities, and what contracts connect their outputs?

The supplied source spans two repositories: **antares-app** owns the application, while **bi-migration-converter** owns separately packaged conversion tooling. These are source-ownership boundaries, not a count of services or processes. The repository table below records each repository’s revision and responsibilities.

**Claims:**

- arch-app-purpose
- arch-converter-package

**Table style:** repositories

| ID | Name | Type | Technology | Responsibility | Status |
| --- | --- | --- | --- | --- | --- |
| arch-app-repo | antares-app | repository | Go and web toolchain | https://github.com/Antares-BI/antares-app at f281dccd24b538f8d2a8b314570744c158581295. | implemented |
| arch-converter-repo | bi-migration-converter | repository | .NET; Python | https://github.com/Antares-BI/bi-migration-converter at 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923. | implemented |

**Claims:**

- arch-app-purpose
- arch-api-start
- arch-claim-cli
- arch-web-contract
- arch-desktop-contract
- passport-claim-development
- arch-converter-package
- arch-python-boundary
- arch-claim-profiler
- arch-claim-validation
- passport-claim-validation-flow

**The cross-repository connection is documented intent plus verified producer packaging, not a verified end-to-end integration.** Application documentation says `antares-app` delegates conversion to a converter image. In `bi-migration-converter`, packaging definitions build Linux x64/arm64 NativeAOT outputs and a frozen-uv Python archive. The selected evidence does not establish the application's concrete image selection, launch wiring, or complete request/artifact contract with that package; it therefore does not establish a network API between the repositories.

The verified C#-to-Python child-process/file contract is **internal to `bi-migration-converter`**, not a handoff between repositories. C# invokes local Python modules with scoped context, and the initialize/step/finish adapter atomically persists configuration.

Profiler evidence also describes separate source responsibilities: application build definitions include embedded profiler dependencies, while converter code invokes its own Tableau profiler against a project workbook. These facts do not establish shared profiler producer wiring between the repositories.

**Claims:**

- arch-app-purpose
- arch-converter-package
- arch-python-boundary
- arch-claim-profiler
- passport-claim-development

## Logical architecture

**Chapter ID:** architecture

**Question:** What are the system&\#39;s parts, boundaries and contracts, and how do the API and conversion subsystems decompose?

The Antares application owns HTTP access, analysis coordination and local persistence; the separately packaged BI migration converter owns artifact-driven conversion. Converter-image delegation is documented, but the supplied model does not establish the application's concrete invocation contract. Documented input and target coverage has not been verified endpoint by endpoint.

The Go API initializes configuration, artifact storage, a durable queue and in-process workers. SQLite is a local file-backed store for records and queue state, not a separately addressed database service.

The frontend and standalone CLI are HTTP clients, not analysis engines. The frontend sends credentialed JSON and multipart requests; the CLI supports bearer-authenticated uploads and optional status polling. Frontend configuration selects web or desktop mode—it does not establish that both run together. In desktop mode, the Tauri shell starts and owns an embedded loopback API child, supplies credentials and local paths, and stops the child on exit.

Optional ClamAV scanning crosses an external daemon boundary: configured scan errors reject uploads; without a scanner, antivirus scanning is skipped. Microsoft Fabric is a remote deployment boundary for converted semantic models and reports, using bearer-authenticated REST calls. Item creation handles immediate identity responses or asynchronous operation polling; deployment requires a completed Power BI conversion.

The structural view separates these client, processing, storage and external-system boundaries.

**Claims:**

- arch-app-purpose
- arch-api-start
- arch-web-contract
- arch-claim-cli
- arch-desktop-contract
- arch-sqlite-contract
- arch-converter-package
- arch-claim-av
- arch-fabric-contract

**Diagram (see Diagrams):** architecture-part-2-diagram

The API separates durable analysis work from artifact storage and event delivery. Analysis creation persists a pending record and enqueues its UUID in the SQL-backed queue. **HTTP 201 means accepted, not completed**; enqueue failure marks the analysis failed. The queue returns job IDs to an in-process worker pool, whose registered executors process source inputs and generate reports.

Three storage and delivery contracts remain distinct:

- **SQLite** holds analysis records and durable queue state, shared by the API and workers.
- **The artifact filesystem** holds source-tagged raw workbooks, computed indexes and report artifacts rather than queue state.
- **Job event streaming** maintains a separate JSONL write-ahead log, bounded replay and live subscriptions. The API exposes replay and live events through SSE. WAL write failures are logged without failing event emission, and overflowing live subscriptions may drop events; streaming therefore must not be treated as the authoritative completion record.

The **LLM gateway** is another component inside the API process, attached outside desktop mode—not a separate service. It authenticates gateway keys, resolves configured upstream credentials, applies capacity and rate admission policies, rewrites upstream routing and authentication, and streams provider-native HTTP responses. This gateway contract does not establish that converter enrichment calls pass through it.

**Claims:**

- arch-api-start
- passport-claim-analysis-acceptance
- arch-sqlite-contract
- arch-worker-runtime
- arch-artifact-contract
- arch-event-contract
- arch-gateway-contract

**Diagram (see Diagrams):** architecture-part-4-diagram

The separately packaged converter uses a **local process/file boundary**: its .NET caller launches Python modules with project, run, workbook and step scope. Adapters dispatch `initialize`, `step` and `finish`, persist configuration atomically, and return process status; a nonzero exit fails the caller.

The local Tableau profiler turns workbook metadata, calculations and visual structure into versioned JSON artifacts (schema 0.7). Python adapters combine deterministic preprocessing and output writing with selective, artifact-bound LLM enrichment—not an unrestricted remote conversion engine.

Two artifact contracts clarify ownership:

- **BRD sections:** deterministic preprocessing supplies context for LLM generation where required; postprocessing and deterministic Markdown writing produce section files whose existence is checked.
- **Power BI source cache:** Python exports and verifies Parquet tables and returns technical JSON results. C# validates those results and publishes the authoritative cache manifest; Python does not own manifest publication.

Culture resolution is an in-process adapter responsibility. It combines deterministic signals and returns a decision, confidence and supporting evidence; an undecidable result can remain unset with low confidence rather than forcing a culture choice.

Report validation and experiment tooling provides **local/CI orchestration**, not production runtime middleware. Comparison requires matching successful conversion and deployment stage results before the comparison module is invoked. This prerequisite is an implementation contract, not evidence that validation has been run successfully.

**Claims:**

- arch-converter-package
- arch-python-boundary
- arch-python-work
- arch-claim-profiler
- arch-claim-validation
- arch-source-cache
- arch-culture
- passport-claim-conversion-flow

**Diagram (see Diagrams):** architecture-part-6-diagram

## End-to-end flows

**Chapter ID:** scenarios

**Question:** How do the main outcomes happen, which variants change the flow, and what counts as completion?

Completion depends on the flow. Analysis submission accepts asynchronous work; BRD generation produces a local section artifact; source-cache export ends with metadata published by its caller; report validation proceeds through deployment and comparison.

- **Analysis:** HTTP 201 confirms acceptance, not completed analysis.
- **BRD section:** completion means the section artifact exists. That check does not establish semantic acceptance of generated content.
- **Power BI source cache:** Python returns verified export results, but the C# caller must validate them and publish the cache manifest to complete the artifact contract.
- **Report validation:** conversion, deployment and comparison have separate stage results. Report publication is attempted independently of success-conditioned Power BI cleanup.

The intake matrix distinguishes submission, upload inspection and analysis acceptance. Three representative converter flows then explain BRD generation, cache export and report validation. These are separate contracts, not one verified continuous pipeline.

The supplied research establishes implementation and configured checks, not successful observed end-to-end runs.

**Claims:**

- passport-claim-analysis-acceptance
- passport-claim-conversion-flow
- passport-claim-cache
- passport-claim-validation-flow

These related intake contracts cover submission, input inspection and asynchronous acceptance—not completed analysis. The inspected JSON creation handler is not established as the multipart upload handler.

| Trigger or variant | Stages and contract | Implementation status | Terminal outcome or gap |
|---|---|---|---|
| CLI submits a local file | Stream an authenticated multipart upload to `POST /api/v1/analyses/upload`; parse the returned analysis ID. `--wait` optionally polls status. | Implemented CLI submission and optional polling. | Returns an ID or polled status; polling success criteria are not established here. |
| Uploaded source; scanner configured or absent | Check filename, file/package magic and ZIP safety before optional antivirus scanning. Rewind the stream for scanning when configured. | Implemented checks and conditional scanning. Configured scan errors—including operational failures—reject input. Without a scanner, antivirus scanning is skipped. | Input passes inspection or is rejected; passing inspection does not establish analysis completion. |
| JSON analysis creation | Validate instance context, JSON and source; persist a pending analysis with capabilities; enqueue its UUID at priority 5. | Implemented persistence and asynchronous enqueue contract. Queue failure marks the analysis failed. | HTTP 201 accepts the analysis for asynchronous work; it does not report completed analysis. |

**Claims:**

- passport-claim-cli-upload
- passport-claim-upload-safety
- passport-claim-analysis-acceptance

**Scenario (see Scenarios):** passport-scenario-brd

**Scenario (see Scenarios):** passport-scenario-cache

**Scenario (see Scenarios):** passport-scenario-report-validation

## Technologies and tools

**Chapter ID:** technologies

**Question:** Which technologies and tools implement each responsibility, in which scope, and with what evidence?

Antares combines a Go application with local, file-backed SQLite persistence, TypeScript/React clients, and a Tauri/Rust desktop shell. The separately packaged .NET converter invokes local Python adapters through process and file contracts; those adapters are not a remote service.

Version labels describe different scopes: **.NET 10-channel NativeAOT** and the **Python 3.13 archive** describe converter packaging. **Python >=3.11** is the Python package requirement, not a conflicting observed runtime version.

**mise, just and CGO** belong to development and build tooling. CGO is used for API builds; the standalone CLI is built without it. Python/YAML/GitHub Actions validation tooling supports local and CI workflows, not an independently deployed service.

External integrations are conditional: the LLM gateway routes to configured provider-native endpoints and is attached outside desktop mode. Optional **clamd** scanning is skipped when no scanner is configured; when configured, scan errors reject upload processing. Neither integration should be read as an unconditional runtime dependency.

The reference below maps technologies to responsibilities and scope. Inspected source and configuration establish these contracts and packaging choices, not successful execution or deployment. Actual run results and deployed versions are not established here.

**Claims:**

- arch-web-contract
- arch-desktop-contract
- arch-sqlite-contract
- arch-converter-package
- arch-python-boundary
- arch-claim-validation
- passport-claim-development
- arch-gateway-contract
- arch-claim-av

**Table style:** technologies

| ID | Name | Type | Technology | Responsibility | Status |
| --- | --- | --- | --- | --- | --- |
| arch-app | Antares application | component | Go; web clients | BI analysis and migration application delegating conversion to a separately packaged image. | implemented |
| arch-web | Web and desktop frontend | component | TypeScript; React; Vite; Axios; React Query | React frontend covers authenticated analysis workflows, public guest upload/report previews and administration; shared components and hooks remain in-process UI code. | implemented |
| arch-desktop | Desktop native shell | component | Tauri; Rust | Tauri desktop process launches and owns a local embedded API child, opens a webview after startup and stops the backend on close or exit. | implemented |
| arch-cli | Antares command-line client | component | Go; HTTP; JSON | User/script-facing HTTP client for authentication, analysis upload/status and reports; not an analysis engine. | implemented |
| arch-api | Antares API | component | Go; SQLite | Initializes database, configuration, logging, artifact storage, durable queue and in-process worker pool. | implemented |
| arch-workers | Background analysis workers | component | Go | In-process worker pool executes registered jobs using the shared durable SQL queue, database and artifact storage; analysis includes profiling, normalization and report generation. | implemented |
| arch-sqlite | Local SQLite database | store | SQLite; database/sql; mattn/go-sqlite3 | File-backed persistence shared by API and workers, including the durable job queue; not a separately addressed database service. | implemented |
| arch-artifact-store | Analysis artifact filesystem | store | Filesystem; JSON; gzip | Local filesystem persists source-tagged raw workbooks, computed indexes and report artifacts. | implemented |
| arch-events | Job event streaming | component | Go; SSE; JSONL | Per-job event manager assigns sequence IDs, appends a JSONL WAL and maintains bounded replay and live subscriptions; HTTP adapter exposes SSE. | implemented |
| arch-gateway | LLM gateway | component | Go HTTP; SQL-backed gateway configuration | In-process provider-native HTTP reverse proxy with credential resolution and admission policies; attached outside desktop mode. | implemented |
| arch-llm-provider | Configured LLM upstreams | external\_system | Provider-native HTTP APIs | Configured API or subscription endpoints used by the application&\#39;s gateway. | implemented |
| arch-clamav | ClamAV daemon | external\_system | clamd TCP or Unix socket | Optional external malware scanner for uploaded files; configured scan errors reject uploads, while absent scanner skips antivirus scanning. | implemented |
| arch-fabric | Microsoft Fabric | external\_system | Bearer-authenticated Fabric REST | Remote workspace service for semantic model and report deployment. | implemented |
| arch-converter | BI migration converter | component | .NET 10-channel NativeAOT; Python 3.13 archive with uv; package Python &gt;=3.11 | Artifact-driven conversion CLI packaged separately from application. | implemented |
| arch-python | Python conversion adapters | component | Python; JSON; Arrow; zstd Parquet | Structured preprocessing, LLM enrichment, deterministic output writers, culture resolution and source cache helpers. | implemented |
| arch-profiler | Tableau workbook profiler | component | Python; XML; JSON | Local XML-analysis subsystem for TWB/TWBX metadata, calculations, visual structure, lineage and conversion hints. | implemented |
| arch-validation | Report validation and experiment tooling | component | Python; YAML; GitHub Actions | Local and CI-oriented conversion, deployment, comparison and experiment orchestration; not an independently deployed service. | implemented |
| passport-analysis-record | Analysis record | entity | UUID; JSON; database record | Pending analysis identity, user/source/scope metadata and persisted capabilities handed to asynchronous execution. | implemented |
| passport-brd-artifacts | BRD section artifacts | entity | JSON; Markdown | Structured profiler context and generated content merged into deterministic Markdown sections. | implemented |
| passport-cache-artifacts | Power BI source cache artifacts | entity | Arrow; zstd Parquet; JSON | Verified Parquet tables, technical export result and caller-published source cache manifest. | implemented |
| passport-aibi-bundle | Databricks AI/BI local bundles | entity | dashboard.lvdash.json; databricks.yml | Locally generated dashboard and view bundles prepared for later deployment. | implemented |
| passport-validation-run | Report-validation run artifacts | entity | JSON; YAML; HTML; archives | Project-scoped conversion, deployment and comparison stage records and published reports. | implemented |
| arch-app-repo | antares-app | repository | Go and web toolchain | https://github.com/Antares-BI/antares-app at f281dccd24b538f8d2a8b314570744c158581295. | implemented |
| arch-converter-repo | bi-migration-converter | repository | .NET; Python | https://github.com/Antares-BI/bi-migration-converter at 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923. | implemented |
| passport-development-environment | Local development and release build environment | environment | mise; just; Go; CGO | Documented workspace setup, local service ports and API/CLI build outputs. | implemented |

**Claims:**

- arch-app-purpose
- arch-api-start
- arch-web-contract
- arch-desktop-contract
- arch-claim-cli
- arch-worker-runtime
- arch-sqlite-contract
- arch-artifact-contract
- arch-event-contract
- arch-gateway-contract
- arch-claim-av
- arch-fabric-contract
- arch-converter-package
- arch-python-boundary
- arch-python-work
- arch-source-cache
- arch-culture
- arch-claim-profiler
- arch-claim-validation
- passport-claim-analysis-acceptance
- passport-claim-conversion-flow
- passport-claim-cache
- passport-claim-local-bundles
- passport-claim-validation-flow
- passport-claim-development

## Data and artifacts

**Chapter ID:** data

**Question:** What information does Antares persist and exchange, and which artifact checks and ownership boundaries are established?

Analysis identity and results live in two local persistence boundaries: **SQLite holds the analysis record and durable job queue; the artifact filesystem holds raw workbooks and computed results.** The selected evidence does not establish retention, deletion, backups or cross-store consistency guarantees.

The persisted analysis record contains pending status, user/source/scope metadata and capabilities. Its UUID identifies accepted work, not completed results: HTTP 201 acknowledges acceptance. If enqueueing fails, the analysis is marked failed.

| Persisted information | Meaning and boundary |
|---|---|
| Analysis record and queued job identity | Local, file-backed SQLite preserves analysis state and durable queued work. The queue carries the job UUID, not the result artifacts. |
| Source-tagged raw workbooks | Background analysis workers write raw workbook data to the local artifact filesystem as gzip-compressed JSON. |
| Computed workbooks and aggregate index | Workers persist computed artifacts containing scores, status and counts. The aggregate index maps workbook IDs to positions in an array; this is an artifact lookup structure, not evidence of physical database keys or database cardinalities. |

Workers are the artifact writers, using the database and local artifact storage shared with the API. Persisting an analysis identity is therefore distinct from producing its workbook and report artifacts.

**Claims:**

- passport-claim-analysis-acceptance
- arch-sqlite-contract
- arch-artifact-contract
- arch-worker-runtime

| Artifact | Origin and consumer | Invariant or persistence boundary |
|---|---|---|
| Tableau profiler JSON | Local Python profiler analyzes project workbooks; conversion consumes its artifacts. | Artifacts identify schema version **0.7**. BRD consumer evidence does not establish external profiler-producer wiring. |
| BRD section Markdown | Python section processing combines profiler context and generated content, then writes deterministic Markdown; the C# bridge checks process success. | Output existence is checked, not content correctness. Downstream content acceptance is unestablished. |
| Power BI source cache | Python exports Parquet tables and technical result JSON; the C# caller validates the result and publishes the cache manifest. | Export verifies schema and row counts. Python owns technical output, **not manifest publication**; downstream cache acceptance is unestablished. |
| AI/BI dashboard and view bundles | AI/BI generation assembles dashboard JSON and local view/dashboard bundles with target catalog/schema mappings. | Bundles are local deployment inputs. Their creation does not establish remote deployment or acceptance. |
| Validation stage JSON and comparison reports | Validation persists conversion/deployment stage records; comparison consumes matching successful records and source/target scenarios. CI attempts report publication. | Stage state is run/project/target scoped. Report publication is attempted, not proven successful; downstream report acceptance is unestablished. |

**Claims:**

- arch-claim-profiler
- passport-claim-conversion-flow
- passport-claim-cache
- passport-claim-local-bundles
- passport-claim-validation-flow
- arch-claim-validation

## Runtime and deployment

**Chapter ID:** runtime

**Question:** What runs where, how does it start, and which modes and configuration does it require?

Antares supports a separately started API and web clients, a desktop shell owning a local API child, an HTTP CLI, and separately packaged local conversion tooling.

| Execution form | Placement and mode |
|---|---|
| API and in-process workers | Documented local API port **8090**; workers run inside the API process. Local definitions do not establish production hosting, scaling or readiness. |
| Web clients | Documented development ports: main app **3000**, guest Try **3001**. `VITE_APP_MODE` selects web or desktop frontend entry. Production placement is unverified. |
| Desktop shell and embedded API | Shell implementation launches and owns a local API child on **127.0.0.1**, using an ephemeral port, and presents the frontend in a webview. |
| Standalone CLI | User/script HTTP client with a configurable API base URL; it is not a local analysis engine. |
| Converter, Python adapters and profiler | Separate converter packaging definitions target **Linux x64/arm64**, including a Python environment. Adapters use local child processes and files; the workbook profiler runs locally, not as a remote service. |
| Validation tooling | Local/CI-oriented conversion, deployment and report-comparison orchestration; not an independently deployed service. |

Development commands and packaging definitions describe intended execution; inspected implementation does not establish observed runs or successful checks.

**External connections:** optional ClamAV, configured LLM upstreams reached through the in-process gateway, and Microsoft Fabric are external endpoints—not bundled processes. Their hosting is not established.

**Claims:**

- arch-api-start
- arch-worker-runtime
- arch-web-contract
- arch-desktop-contract
- arch-claim-cli
- arch-converter-package
- arch-python-boundary
- arch-claim-profiler
- arch-claim-validation
- passport-claim-development
- arch-gateway-contract
- arch-claim-av
- arch-fabric-contract

**API startup initializes local persistence and in-process execution.** Source definitions initialize the database, persisted configuration, logging, artifact storage, durable queue and worker pool. Database and artifact paths are configurable. SQLite is a local file, not an external database service; the database, durable queue and artifact filesystem hold persisted local state. This evidence does not establish retention or recovery guarantees.

**The desktop shell owns the embedded API lifecycle.** It supplies local paths and credentials for launch and streaming through environment variables, starts the API on loopback, and opens its webview after startup. On window close or application exit, it terminates the API child. This startup sequence does not by itself establish readiness.

**Gateway attachment depends on mode.** API startup attaches the LLM gateway outside desktop mode. Gateway-only mode is restricted to development/test and excluded from desktop.

**Conversion uses packaged Python and local process/file contracts.** The converter package includes a Python environment; the converter invokes local Python modules through an initialize/step/finish contract. Configuration updates are written atomically, and a nonzero child-process exit fails the caller. These are inspected implementation and packaging definitions, not evidence of a successful execution.

**Claims:**

- arch-api-start
- arch-worker-runtime
- arch-sqlite-contract
- arch-artifact-contract
- arch-desktop-contract
- arch-gateway-contract
- arch-python-boundary
- arch-converter-package

## Development guide

**Chapter ID:** development

**Question:** Which supported work path should I follow, and how can I check my change?

In **antares-app**, the documented setup commands are `mise install` and `just setup`. Documented local ports are API **8090**, main web **3000**, and guest Try **3001**—development orientation, not deployment guidance.

Choose the entry point matching your change:

| Work path | Implementation entry point | Setup, build or check reference |
| --- | --- | --- |
| Application API/workers | API startup initializes persistence, artifacts and the queue; registered in-process workers process analysis and generate reports. | Use the application’s documented setup and API build definitions. The API build uses **CGO and embedded profiler dependencies**. |
| Web UI | React bootstrap selects desktop/web mode; the shared HTTP client supports analysis creation and upload. | Use the documented application setup and main/guest development startup guidance. No focused UI test invocation is established here. |
| Standalone CLI | Go command dispatch for authentication, analysis upload/status and reports. | Consult the standalone CLI build definition: it builds **without CGO**, unlike the API. |
| Converter | In **bi-migration-converter**, start at the C# local Python bridge, Python adapters and converter packaging definitions. | Report-validation tooling defines conversion, deployment and comparison checks, including matching successful stage-result requirements; it is not a lightweight unit test or a proven successful run. Local application-to-converter rebuild/consumption order is not established; sibling clones must not be assumed to consume changes automatically. |

Inspection establishes implementation and check definitions, **not passed checks**.

**Claims:**

- passport-claim-development
- arch-api-start
- arch-worker-runtime
- arch-web-contract
- arch-claim-cli
- arch-converter-package
- arch-python-boundary
- arch-claim-validation
- passport-claim-validation-flow

## Operations and recovery

**Chapter ID:** operations

**Question:** How can maintainers diagnose important failures, and what must be established before recovery is safe?

Distinguish accepted work, diagnostic events and confirmed results. HTTP 201 accepts analysis work; it does not confirm completion. SSE heartbeats indicate stream activity, not job success. An unconfirmed Fabric dispatch can leave the remote outcome unknown and must not be treated as safe to retry.

The following table describes implemented failure handling and recovery constraints, not a verified operational runbook. Source inspection does not demonstrate successful recovery.

**Claims:**

- passport-claim-analysis-acceptance
- arch-event-contract
- arch-fabric-contract

Implemented handling and diagnostic guidance—not a verified recovery runbook.

| Boundary/failure | Signal | Action | Safety condition | Result check |
|---|---|---|---|---|
| Analysis enqueue failure/interruption | Enqueue failure marks analysis failed; dequeue removes entry. | Inspect persisted analysis state. | Restart recovery and safe re-enqueue are not established. | HTTP 201 means acceptance; interrupted-execution verification is unestablished. |
| Event delivery loss | WAL errors are logged without failing Emit; live overflow drops events. | Consult replay; do not infer job failure from missing events. | Replay completeness is not guaranteed. | Check job results separately. |
| Fabric dispatch | 201 returns identity; 202 requires operation polling; unconfirmed dispatch means unknown outcome. | Poll known operations; remote reconciliation procedure is unestablished. | Unknown is not safe to retry. | Distinguish item identity, operation result and unresolved outcome. |
| Cache export | Schema/row-count checks and caller validation precede manifest publication. | Caller publishes after validation; failed-export recovery is unestablished. | Files alone do not establish publication. | Check validated result and caller-published manifest. |
| Upload scanning | Configured scanner errors reject processing; absent configuration skips antivirus scanning. | Diagnose scanner errors; disabling scanning is not recovery. | Skipped scanning is not a clean scan. | Confirm scan acceptance; recovery verification is unestablished. |

**Claims:**

- passport-claim-analysis-acceptance
- arch-sqlite-contract
- arch-event-contract
- arch-fabric-contract
- passport-claim-cache
- passport-claim-upload-safety

## Objects

### BRD section artifacts

**ID:** passport-brd-artifacts

**Type:** entity

Structured profiler context and generated content merged into deterministic Markdown sections.

**Technology:** JSON; Markdown

**Status:** implemented

**Inputs:**

- Profiler context
- LLM-generated section content

**Outputs:**

- Section Markdown

**Contracts:**

- Artifact-bound section generation; output existence checked

**Claims:**

- passport-claim-conversion-flow

### Report-validation run artifacts

**ID:** passport-validation-run

**Type:** entity

Project-scoped conversion, deployment and comparison stage records and published reports.

**Technology:** JSON; YAML; HTML; archives

**Status:** implemented

**Inputs:**

- conversion\_configuration.json projectName
- Target scenario
- Converter exit and target output directory

**Outputs:**

- report-validation/runs/&lt;run-id&gt;/&lt;stage&gt;.json
- HTML/archive reports

**Contracts:**

- projectName resolves to repository artifacts/&lt;projectName&gt;
- Targets select tableau2pbi.yaml or tableau2aibi.yaml

**Claims:**

- passport-claim-validation-flow

### Power BI source cache artifacts

**ID:** passport-cache-artifacts

**Type:** entity

Verified Parquet tables, technical export result and caller-published source cache manifest.

**Technology:** Arrow; zstd Parquet; JSON

**Status:** implemented

**Inputs:**

- PBIX file
- Source-owned assignment and table inventory

**Outputs:**

- Parquet physical tables
- Technical result with reader version
- Cache manifest

**Contracts:**

- Exporter verifies schema and row counts; manifest publication belongs to C\# caller

**Claims:**

- passport-claim-cache

### antares-app

**ID:** arch-app-repo

**Type:** repository

https://github.com/Antares-BI/antares-app at f281dccd24b538f8d2a8b314570744c158581295.

**Technology:** Go and web toolchain

**Status:** implemented

**Claims:**

- arch-app-purpose

### bi-migration-converter

**ID:** arch-converter-repo

**Type:** repository

https://github.com/Antares-BI/bi-migration-converter at 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923.

**Technology:** .NET; Python

**Status:** implemented

**Claims:**

- arch-converter-package

### Antares application

**ID:** arch-app

**Type:** component

BI analysis and migration application delegating conversion to a separately packaged image.

**Technology:** Go; web clients

**Status:** implemented

**Architecture role:** software\_system

**Inputs:**

- Tableau workbook/server/cloud
- PBIX/PBIP ZIP
- Fabric workspace

**Outputs:**

- Analysis and deployment results

**Contracts:**

- Documented targets include Fabric, Databricks AI/BI and beta Sigma; overview claims, not endpoint verification.

**Claims:**

- arch-app-purpose

### Antares API

**ID:** arch-api

**Type:** component

Initializes database, configuration, logging, artifact storage, durable queue and in-process worker pool.

**Technology:** Go; SQLite

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-app

**Inputs:**

- HTTP requests
- Environment/CLI configuration

**Outputs:**

- Queued work
- Persisted results

**Contracts:**

- Default port 8090; configured environment required; SQLite/artifact paths configurable.
- Gateway-only mode restricted to development/test and excluded from desktop.

**Claims:**

- arch-api-start

### BI migration converter

**ID:** arch-converter

**Type:** component

Artifact-driven conversion CLI packaged separately from application.

**Technology:** .NET 10-channel NativeAOT; Python 3.13 archive with uv; package Python &gt;=3.11

**Status:** implemented

**Architecture role:** software\_system

**Inputs:**

- Project configuration
- Profiler artifacts

**Outputs:**

- Converted BI artifacts
- Documentation

**Contracts:**

- Linux x64/arm64 packaging; local Python processes, not a remote service.

**Claims:**

- arch-converter-package
- arch-python-boundary

### Python conversion adapters

**ID:** arch-python

**Type:** component

Structured preprocessing, LLM enrichment, deterministic output writers, culture resolution and source cache helpers.

**Technology:** Python; JSON; Arrow; zstd Parquet

**Status:** implemented

**Architecture role:** component

**Parent:** arch-converter

**Inputs:**

- Configuration/project/run/workbook/step scope
- Profiler JSON
- PBIX assignment

**Outputs:**

- Updated configuration
- Markdown
- Technical cache results

**Contracts:**

- initialize/step/finish; atomic configuration writes; nonzero process exit fails caller.
- Fresh empty cache output; C\# publishes manifest.
- Culture Verdict is in-process: decision/confidence/evidence/source; None/low when undecidable.

**Claims:**

- arch-python-boundary
- arch-python-work
- arch-source-cache
- arch-culture

### Analysis record

**ID:** passport-analysis-record

**Type:** entity

Pending analysis identity, user/source/scope metadata and persisted capabilities handed to asynchronous execution.

**Technology:** UUID; JSON; database record

**Status:** implemented

**Inputs:**

- Validated source and instance context

**Outputs:**

- Analysis response
- Queue UUID

**Contracts:**

- HTTP 201 means accepted, not completed
- Queue failure marks analysis failed

**Claims:**

- passport-claim-analysis-acceptance

### Databricks AI/BI local bundles

**ID:** passport-aibi-bundle

**Type:** entity

Locally generated dashboard and view bundles prepared for later deployment.

**Technology:** dashboard.lvdash.json; databricks.yml

**Status:** implemented

**Inputs:**

- AI/BI generated outputs
- Table metadata
- Deployment mappings

**Outputs:**

- aibi-output/bundles views and dashboard directories

**Contracts:**

- Generation completion is not remote deployment completion

**Claims:**

- passport-claim-local-bundles

### Local development and release build environment

**ID:** passport-development-environment

**Type:** environment

Documented workspace setup, local service ports and API/CLI build outputs.

**Technology:** mise; just; Go; CGO

**Status:** implemented

**Inputs:**

- mise install
- just setup

**Outputs:**

- API :8090
- Main app :3000
- Guest Try :3001
- bin/antares-api
- bin/antares

**Contracts:**

- API includes embedded profiler dependencies
- CLI release ZIPs target macOS, Linux and Windows

**Claims:**

- passport-claim-development

### Tableau workbook profiler

**ID:** arch-profiler

**Type:** component

Local XML-analysis subsystem for TWB/TWBX metadata, calculations, visual structure, lineage and conversion hints.

**Technology:** Python; XML; JSON

**Status:** implemented

**Architecture role:** component

**Parent:** arch-converter

**Inputs:**

- TWB or TWBX workbook

**Outputs:**

- Version 0.7 modular profiler artifacts

**Contracts:**

- data/json\_profiles and pages output directories

**Claims:**

- arch-claim-profiler

### Report validation and experiment tooling

**ID:** arch-validation

**Type:** component

Local and CI-oriented conversion, deployment, comparison and experiment orchestration; not an independently deployed service.

**Technology:** Python; YAML; GitHub Actions

**Status:** implemented

**Architecture role:** component

**Parent:** arch-converter

**Inputs:**

- Scenario suite
- Conversion and deployment stage results

**Outputs:**

- Comparison results and reports

**Contracts:**

- Schema-v1 scenarios require a source and at least one target; stage state is run/project/target scoped

**Claims:**

- arch-claim-validation

### ClamAV daemon

**ID:** arch-clamav

**Type:** external\_system

Optional external malware scanner for uploaded files; configured scan errors reject uploads, while absent scanner skips antivirus scanning.

**Technology:** clamd TCP or Unix socket

**Status:** implemented

**Inputs:**

- Seekable upload stream

**Outputs:**

- Clean result or scan error

**Contracts:**

- zINSTREAM command with network-order chunk lengths and zero terminator

**Claims:**

- arch-claim-av

### Antares command-line client

**ID:** arch-cli

**Type:** component

User/script-facing HTTP client for authentication, analysis upload/status and reports; not an analysis engine.

**Technology:** Go; HTTP; JSON

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-app

**Inputs:**

- Commands, local input files and bearer credentials

**Outputs:**

- JSON or human-readable responses

**Contracts:**

- Configurable API base URL; multipart POST /api/v1/analyses/upload; optional status polling

**Claims:**

- arch-claim-cli

### Local SQLite database

**ID:** arch-sqlite

**Type:** store

File-backed persistence shared by API and workers, including the durable job queue; not a separately addressed database service.

**Technology:** SQLite; database/sql; mattn/go-sqlite3

**Status:** implemented

**Inputs:**

- Analysis and job records
- Prioritized queue entries

**Outputs:**

- Persisted records
- Dequeued job IDs

**Contracts:**

- WAL, foreign keys, 10-second busy timeout, immediate transactions
- Queue dequeues highest priority then oldest sequence atomically

**Claims:**

- arch-sqlite-contract

### Background analysis workers

**ID:** arch-workers

**Type:** component

In-process worker pool executes registered jobs using the shared durable SQL queue, database and artifact storage; analysis includes profiling, normalization and report generation.

**Technology:** Go

**Status:** implemented

**Architecture role:** component

**Parent:** arch-api

**Inputs:**

- Queued job IDs
- Analysis inputs

**Outputs:**

- Analysis status
- Reports and computed artifacts

**Contracts:**

- Executor(context, models.Job, integer progress callback); operation-string registry

**Claims:**

- arch-worker-runtime

### Analysis artifact filesystem

**ID:** arch-artifact-store

**Type:** store

Local filesystem persists source-tagged raw workbooks, computed indexes and report artifacts.

**Technology:** Filesystem; JSON; gzip

**Status:** implemented

**Inputs:**

- Extracted and computed analysis data

**Outputs:**

- Compressed raw JSON
- Workbook indexes
- Report JSON

**Contracts:**

- Raw workbooks indexed by ID; aggregate index includes ID-to-array-position map

**Claims:**

- arch-artifact-contract

### Job event streaming

**ID:** arch-events

**Type:** component

Per-job event manager assigns sequence IDs, appends a JSONL WAL and maintains bounded replay and live subscriptions; HTTP adapter exposes SSE.

**Technology:** Go; SSE; JSONL

**Status:** implemented

**Architecture role:** component

**Parent:** arch-api

**Inputs:**

- Structured log and progress events

**Outputs:**

- SSE log/progress events
- Replay log

**Contracts:**

- 15-second SSE heartbeats and replay deduplication
- WAL failures are logged without failing Emit; live subscriber overflow may drop events

**Claims:**

- arch-event-contract

### Web and desktop frontend

**ID:** arch-web

**Type:** component

React frontend covers authenticated analysis workflows, public guest upload/report previews and administration; shared components and hooks remain in-process UI code.

**Technology:** TypeScript; React; Vite; Axios; React Query

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-app

**Inputs:**

- User actions
- Versioned HTTP JSON and analysis responses

**Outputs:**

- Analysis creation requests
- Report views
- Administrative mutations

**Contracts:**

- POST /api/v1/analyses and multipart /api/v1/analyses/upload
- Credentialed Axios requests; desktop token header when applicable
- VITE\_APP\_MODE selects desktop versus web entry

**Claims:**

- arch-web-contract

### Desktop native shell

**ID:** arch-desktop

**Type:** component

Tauri desktop process launches and owns a local embedded API child, opens a webview after startup and stops the backend on close or exit.

**Technology:** Tauri; Rust

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-app

**Inputs:**

- Desktop application launch

**Outputs:**

- Webview
- Local API child process

**Contracts:**

- antares-api --embedded --host 127.0.0.1 --port &lt;ephemeral&gt;
- Environment passes launch/stream tokens, local database/storage paths and credential secret

**Claims:**

- arch-desktop-contract

### LLM gateway

**ID:** arch-gateway

**Type:** component

In-process provider-native HTTP reverse proxy with credential resolution and admission policies; attached outside desktop mode.

**Technology:** Go HTTP; SQL-backed gateway configuration

**Status:** implemented

**Architecture role:** component

**Parent:** arch-api

**Inputs:**

- Provider-native HTTP requests with gateway key

**Outputs:**

- Streamed upstream HTTP responses

**Contracts:**

- Bearer or X-Api-Key gateway authentication; conflicting keys rejected; rewrites upstream authentication and strips client cookies/forwarding credentials.

**Claims:**

- arch-gateway-contract

### Configured LLM upstreams

**ID:** arch-llm-provider

**Type:** external\_system

Configured API or subscription endpoints used by the application's gateway.

**Technology:** Provider-native HTTP APIs

**Status:** implemented

**Inputs:**

- Rewritten authenticated model requests

**Outputs:**

- Model responses

**Contracts:**

- API target base URL/authentication configuration or subscription service URL/shared bearer key.

**Claims:**

- arch-gateway-contract

### Microsoft Fabric

**ID:** arch-fabric

**Type:** external\_system

Remote workspace service for semantic model and report deployment.

**Technology:** Bearer-authenticated Fabric REST

**Status:** implemented

**Inputs:**

- TMDL semantic model and PBIR report definitions

**Outputs:**

- Remote item IDs and operation results

**Contracts:**

- POST /workspaces/{workspaceID}/items; 201 returns item identity; 202 requires operation ID and polling.

**Claims:**

- arch-fabric-contract

## Connections

### arch-rel-delegates

**From:** arch-app

**To:** arch-converter

**Label:** Delegates conversion to converter image

**Type:** dependency

**Interaction:** unknown

**Claims:**

- arch-app-purpose

### arch-rel-python

**From:** arch-converter

**To:** arch-python

**Label:** Local child process/file adapter

**Type:** call

**Interaction:** sync

**Claims:**

- arch-python-boundary

### arch-cli-api

**From:** arch-cli

**To:** arch-api

**Label:** Uploads files and polls analysis through HTTP

**Type:** call

**Interaction:** sync

**Claims:**

- arch-claim-cli

### arch-api-clamav

**From:** arch-api

**To:** arch-clamav

**Label:** Optionally scans uploaded streams

**Type:** call

**Interaction:** sync

**Claims:**

- arch-claim-av

### arch-converter-profiler

**From:** arch-converter

**To:** arch-profiler

**Label:** Profiles project workbook into JSON artifacts

**Type:** call

**Interaction:** artifact

**Claims:**

- arch-claim-profiler

### arch-api-workers

**From:** arch-api

**To:** arch-workers

**Label:** Constructs and configures analysis worker pool

**Type:** call

**Interaction:** in\_process

**Claims:**

- arch-worker-runtime

### arch-api-db

**From:** arch-api

**To:** arch-sqlite

**Label:** Shares local persistence and durable queue

**Type:** data

**Interaction:** in\_process

**Claims:**

- arch-sqlite-contract
- arch-worker-runtime

### arch-workers-storage

**From:** arch-workers

**To:** arch-artifact-store

**Label:** Persists analysis inputs and computed artifacts

**Type:** data

**Interaction:** artifact

**Claims:**

- arch-artifact-contract

### arch-web-api

**From:** arch-web

**To:** arch-api

**Label:** Credentialed versioned HTTP JSON and multipart upload

**Type:** call

**Interaction:** sync

**Claims:**

- arch-web-contract

### arch-desktop-api

**From:** arch-desktop

**To:** arch-api

**Label:** Owns local embedded API child-process lifecycle

**Type:** dependency

**Interaction:** unknown

**Claims:**

- arch-desktop-contract

### arch-api-events

**From:** arch-api

**To:** arch-events

**Label:** Adapts job replay and live events to SSE

**Type:** call

**Interaction:** in\_process

**Claims:**

- arch-event-contract

### arch-gateway-upstream

**From:** arch-gateway

**To:** arch-llm-provider

**Label:** Proxies authenticated provider-native HTTP with streaming responses

**Type:** call

**Interaction:** sync

**Claims:**

- arch-gateway-contract

### arch-api-fabric

**From:** arch-api

**To:** arch-fabric

**Label:** Starts checkpointed semantic model/report deployment with operation polling

**Type:** call

**Interaction:** sync

**Claims:**

- arch-fabric-contract

### arch-provenance-app

**From:** arch-app-repo

**To:** arch-app

**Label:** Source provenance for application

**Type:** association

**Interaction:** artifact

**Claims:**

- arch-app-purpose

### arch-provenance-converter

**From:** arch-converter-repo

**To:** arch-converter

**Label:** Source provenance for converter

**Type:** association

**Interaction:** artifact

**Claims:**

- arch-converter-package

## Scenarios

### Generate a BRD section

**ID:** passport-scenario-brd

Local C#/Python processing combines deterministic context with selectively generated text.

**Inputs:**

- Profiler artifacts
- Effective configuration and project/run/workbook/step identity

**Outputs:**

- Section Markdown

**Failure modes:**

- Nonzero Python process exit fails the bridge
- Section output existence is checked

#### 1. Preprocess, enrich and render

**Step ID:** passport-step-brd-process

Extract structured context, invoke artifact-bound prompt when required, merge output and deterministically write Markdown.

**Objects:**

- arch-converter
- arch-python
- passport-brd-artifacts

**Claims:**

- passport-claim-conversion-flow

### Validate a converted report

**ID:** passport-scenario-report-validation

CI records conversion, deploys target reports and compares them with source reports.

**Inputs:**

- Artifact project
- Local conversion or recorded successful external converter result

**Outputs:**

- Stage JSON
- HTML/archive comparison report
- CI summary

**Failure modes:**

- Comparison is configured to fail on high-severity differences
- Power BI cleanup is success-conditioned; publishing is always attempted

#### 1. Record conversion and compare deployment

**Step ID:** passport-step-validation

Resolve artifact project and target scenario, record conversion, deploy, compare with fail-on-diff and publish reports.

**Objects:**

- arch-validation
- passport-validation-run

**Claims:**

- passport-claim-validation-flow

### Export a Power BI source cache

**ID:** passport-scenario-cache

Separate technical export from authoritative manifest publication.

**Inputs:**

- PBIX
- JSON export assignment

**Outputs:**

- Verified Parquet tables
- Technical JSON result
- Published manifest

**Failure modes:**

- Caller validates returned result before manifest publication

#### 1. Export, verify and publish

**Step ID:** passport-step-cache-export

Export bounded batches, read back schema and row counts, return technical result and let the C# caller publish the manifest.

**Objects:**

- arch-converter
- arch-python
- passport-cache-artifacts

**Claims:**

- passport-claim-cache

### Submit an analysis from the CLI

**ID:** passport-scenario-cli-upload

A user or script uploads a local file and optionally waits for completion.

**Inputs:**

- Local file
- Bearer authentication
- API base URL

**Outputs:**

- Analysis ID as JSON or human-readable confirmation
- Optional polled status

#### 1. Stream upload and optionally poll

**Step ID:** passport-step-cli-upload

Stream multipart file through POST /api/v1/analyses/upload, parse returned ID and optionally poll with --wait.

**Objects:**

- arch-cli
- arch-api
- passport-analysis-record

**Claims:**

- passport-claim-cli-upload

### Accept an analysis for asynchronous execution

**ID:** passport-scenario-analysis

Persist analysis metadata and enqueue work rather than synchronously performing analysis.

**Inputs:**

- Authenticated instance context
- Valid JSON/source

**Outputs:**

- HTTP 201 analysis response
- Pending analysis and queued UUID

**Failure modes:**

- Invalid context or source is rejected
- Queue failure marks analysis failed

#### 1. Validate, persist and enqueue

**Step ID:** passport-step-analysis-accept

Construct pending analysis with UUID/user/source/scope metadata, persist capabilities and enqueue at priority 5.

**Objects:**

- arch-api
- arch-sqlite
- passport-analysis-record

**Claims:**

- passport-claim-analysis-acceptance

### Validate an uploaded source before processing

**ID:** passport-scenario-upload-safety

Apply file/package and ZIP-safety checks, then optional antivirus scanning.

**Inputs:**

- Multipart uploaded source file
- Optional configured scanner

**Outputs:**

- Accepted input or validation/scan error

**Failure modes:**

- Configured scanning rejects all scan errors, including operational failure
- No configured scanner means antivirus scanning is skipped

#### 1. Inspect file and optionally scan

**Step ID:** passport-step-upload-checks

Validate filename, magic/package and ZIP safety, rewind and scan when configured.

**Objects:**

- arch-api
- arch-clamav

**Claims:**

- passport-claim-upload-safety

## Diagrams

### Whole-system runtime structure

**ID:** architecture-part-2-diagram

**Type:** containers

The application contains distinct API, frontend, standalone CLI and desktop-shell runtime units. The frontend selects web or desktop mode; these are variants, not evidence of simultaneous operation. Frontend requests use credentialed HTTP JSON and multipart upload; the CLI uses bearer-authenticated multipart upload and status polling. The desktop shell owns a local embedded API child. SQLite is a local file-backed persistence and durable-queue store, not a database service. Outside the application boundary, converter-image delegation is documented, but its concrete invocation contract is unestablished; no API-to-converter call is asserted. The API optionally scans uploads through ClamAV and deploys semantic models and reports through bearer-authenticated Fabric REST, including operation polling.

**Question:** How do application runtime units and local persistence connect to the separately packaged converter and external services?

**Scope:** arch-app

**Nodes (see Objects):**

- arch-app
- arch-api
- arch-web
- arch-cli
- arch-desktop
- arch-sqlite
- arch-converter
- arch-clamav
- arch-fabric

**Connections (see Connections):**

- arch-web-api
- arch-cli-api
- arch-desktop-api
- arch-api-db
- arch-rel-delegates
- arch-api-clamav
- arch-api-fabric

### API components, stores and upstream boundary

**ID:** architecture-part-4-diagram

**Type:** components

Within the parent Antares application context, the API constructs an in-process worker pool sharing its SQLite-backed persistence and durable queue. Workers write analysis inputs and computed artifacts to a local filesystem outside the API component boundary. Job event streaming is another API component: its JSONL event WAL and replay/live delivery are distinct from SQLite queue state and analysis artifacts; the API adapts events to SSE. The gateway belongs inside the API process and is attached only outside desktop mode. It authenticates gateway keys, applies admission policy, rewrites upstream credentials and proxies streaming HTTP responses from configured LLM upstreams outside the application. SQLite and the artifact filesystem are local stores, not independently deployed services.

**Question:** How does the API organize in-process work, local state, event delivery and upstream model access?

**Scope:** arch-api

**Nodes (see Objects):**

- arch-app
- arch-api
- arch-workers
- arch-events
- arch-gateway
- arch-sqlite
- arch-artifact-store
- arch-llm-provider

**Connections (see Connections):**

- arch-api-workers
- arch-api-db
- arch-workers-storage
- arch-api-events
- arch-gateway-upstream

### Converter components and local contracts

**ID:** architecture-part-6-diagram

**Type:** components

The converter contains three local components, not remote services. Its C# caller invokes Python adapters through a synchronous child-process/file contract with project/run/workbook/step scope and initialize/step/finish dispatch; adapters persist configuration atomically. BRD section processing combines deterministic preprocessing and output writing with selective, artifact-bound LLM enrichment. For source-cache export, Python owns verified Parquet and technical JSON results; C# validates them and publishes the authoritative cache manifest. Culture resolution stays in-process and can return an undecidable result. The converter separately invokes the Tableau profiler against the project workbook to produce version 0.7 modular JSON artifacts. Report-validation and experiment tooling is contained local/CI orchestration, not runtime middleware: comparison requires matching successful conversion and deployment stage results. No invocation edge to that tooling is asserted.

**Question:** How does the converter divide local conversion, profiling and validation responsibilities?

**Scope:** arch-converter

**Nodes (see Objects):**

- arch-converter
- arch-python
- arch-profiler
- arch-validation

**Connections (see Connections):**

- arch-rel-python
- arch-converter-profiler

## Claims

### passport-claim-conversion-flow

BRD section processing combines deterministic Python preprocessing, artifact-bound LLM generation where required, postprocessing and deterministic Markdown writing with output existence checks; the C# bridge treats a nonzero Python exit as failure.

**Basis:** source\_backed

**Evidence:**

- passport-ev-brd-flow
- passport-ev-python-bridge

### passport-claim-validation-flow

Report validation accepts local conversion or an externally completed conversion, persists stage JSON under a run ID, then CI deploys and compares with high-severity fail-on-diff policy, attempts report publication, and cleans Power BI resources on success.

**Basis:** source\_backed

**Evidence:**

- passport-ev-validation-convert
- passport-ev-validation-contract
- passport-ev-validation-ci

### passport-claim-cache

Power BI source caching exports bounded Arrow batches from PBIX to zstd Parquet, verifies schema and row counts, and returns a technical result; the C# caller validates that result and publishes the manifest.

**Basis:** source\_backed

**Evidence:**

- passport-ev-cache-export
- passport-ev-cache-caller

### passport-claim-cli-upload

The CLI streams a local file as authenticated multipart upload to POST /api/v1/analyses/upload, returns its analysis ID, and optionally polls status with --wait.

**Basis:** source\_backed

**Evidence:**

- passport-ev-cli-upload

### arch-app-purpose

Application overview documents BI inputs, deployment targets and converter-image delegation; developer commands start API and main/guest web clients.

**Basis:** source\_backed

**Evidence:**

- arch-ev-app-overview
- arch-ev-app-dev

### arch-api-start

API initializes database, persisted configuration, logging, artifacts, durable queue and workers; startup flags constrain gateway mode.

**Basis:** source\_backed

**Evidence:**

- arch-ev-api

### arch-converter-package

Converter packaging builds NativeAOT linux x64/arm64 and a frozen-uv Python archive; Python package exposes deployment commands.

**Basis:** source\_backed

**Evidence:**

- arch-ev-package
- arch-ev-pyproject

### arch-python-boundary

C# invokes local Python modules with scoped context; initialize/step/finish dispatch atomically persists configuration.

**Basis:** source\_backed

**Evidence:**

- arch-ev-bridge
- arch-ev-adapter

### arch-python-work

BRD section execution combines deterministic preprocessing, artifact-bound LLM prompts, postprocessing and Markdown writing.

**Basis:** source\_backed

**Evidence:**

- arch-ev-section

### arch-source-cache

Python exports verified Parquet and technical JSON; C# validates and publishes cache manifest.

**Basis:** source\_backed

**Evidence:**

- arch-ev-cache
- arch-ev-cache-caller

### arch-culture

Local deterministic culture resolution combines sniff, sampling and weighted signals; Power BI preprocessing consumes decisions and warnings and clears sample rows.

**Basis:** source\_backed

**Evidence:**

- arch-ev-culture
- arch-ev-culture-caller
- arch-ev-verdict

### passport-claim-analysis-acceptance

Analysis creation validates instance context, JSON and source, persists a pending analysis with capabilities, enqueues its UUID at priority 5 and returns HTTP 201; queue failure marks it failed, so acceptance is not analysis completion.

**Basis:** source\_backed

**Evidence:**

- passport-ev-analysis-create

### passport-claim-upload-safety

Upload validation checks filename, file/package magic and ZIP safety before optional antivirus scanning. A configured scanner rejects malware and operational errors; an absent scanner skips antivirus scanning.

**Basis:** source\_backed

**Evidence:**

- passport-ev-upload-safety

### passport-claim-local-bundles

AI/BI generation assembles dashboard JSON and builds local views/dashboard bundles with target catalog/schema mappings; these outputs do not establish successful remote deployment.

**Basis:** source\_backed

**Evidence:**

- passport-ev-bundles
- passport-ev-bundle-caller

### passport-claim-development

Documented setup uses mise install and just setup; separate development commands start API on 8090, main app on 3000 and guest Try on 3001. API builds use CGO and embedded profiler tags, while the standalone CLI is built without CGO and packaged for multiple platforms.

**Basis:** source\_backed

**Evidence:**

- passport-ev-dev-entry
- passport-ev-backend-build

### passport-claim-json-validation

The shared JSON validator supports whole-document JSON or per-line JSONL validation and returns 0 for valid, 1 for invalid, 2 for explicit input/schema setup failures.

**Basis:** source\_backed

**Evidence:**

- passport-ev-validator

### arch-claim-profiler

The local Python Tableau profiler parses workbook metadata and structure and is invoked by conversion against the project TWB; artifacts identify schema version 0.7.

**Basis:** source\_backed

**Evidence:**

- arch-ev-profiler
- arch-ev-profiler-call
- arch-ev-profiler-schema

### arch-claim-validation

Report comparison consumes source/target scenario suites; the pipeline checks matching successful conversion/deployment stage results before invoking the comparison module with artifact and report arguments.

**Basis:** source\_backed

**Evidence:**

- arch-ev-validation-suite
- arch-ev-validation-call

### arch-claim-av

Upload validation optionally streams files to clamd via INSTREAM; configured scanner errors reject processing, while no scanner skips antivirus scanning.

**Basis:** source\_backed

**Evidence:**

- arch-ev-av
- arch-ev-av-call

### arch-claim-cli

The standalone Go CLI dispatches auth, analysis and report commands; analysis upload streams multipart content with bearer authentication and optionally polls completion.

**Basis:** source\_backed

**Evidence:**

- arch-ev-cli
- arch-ev-cli-entry

### arch-sqlite-contract

Persistence is local file-backed SQLite using database/sql and mattn/go-sqlite3, with WAL, foreign keys, busy timeout and immediate transactions. The SQL-backed queue atomically removes the highest-priority oldest job and returns only its ID.

**Basis:** source\_backed

**Evidence:**

- arch-ev-sqlite
- arch-ev-queue

### arch-worker-runtime

API constructs an in-process worker pool around its DB-backed queue and local storage, registers analysis execution, and workers invoke report generation after processing source inputs.

**Basis:** source\_backed

**Evidence:**

- arch-ev-worker
- arch-ev-worker-executor
- arch-ev-worker-report

### arch-artifact-contract

Analysis storage writes source-tagged raw workbooks as gzip JSON, and computed workbook/index artifacts include scores, status, counts and ID-position mapping.

**Basis:** source\_backed

**Evidence:**

- arch-ev-artifacts-raw
- arch-ev-artifacts-index

### arch-event-contract

Job event manager assigns sequence and timestamp, appends WAL, caches replay and broadcasts; WAL failures do not fail Emit and subscriber overflow can drop live events. HTTP SSE replays/deduplicates events and emits 15-second heartbeats.

**Basis:** source\_backed

**Evidence:**

- arch-ev-events
- arch-ev-sse

### arch-web-contract

Frontend bootstrap selects desktop or web React application by VITE_APP_MODE. Central credentialed Axios client attaches desktop authentication and handles expiration; analysis creation is versioned HTTP JSON or multipart upload, not proof of completed processing.

**Basis:** source\_backed

**Evidence:**

- arch-ev-web-bootstrap
- arch-ev-web-client
- arch-ev-web-create

### arch-desktop-contract

Tauri shell starts and owns an embedded loopback API subprocess, supplies launch/stream credentials and local paths through environment, opens its webview after startup and terminates the child on close or application exit.

**Basis:** source\_backed

**Evidence:**

- arch-ev-desktop-lifecycle
- arch-ev-desktop-process

### arch-gateway-contract

API startup attaches the SQL-backed LLM gateway outside desktop mode. The HTTP proxy authenticates a gateway key, resolves configured API/subscription credentials, applies capacity/rate admission, rewrites upstream routing/authentication and streams the response.

**Basis:** source\_backed

**Evidence:**

- arch-ev-gateway
- arch-ev-gateway-start

### arch-fabric-contract

Fabric deployment requires a completed Power BI conversion and is accepted asynchronously with attempt/correlation IDs. Deployment creates a TMDL SemanticModel, patches PBIR binding and creates a Report; remote item creation handles 201/202 and treats unconfirmed dispatch outcomes as unknown rather than safe to retry.

**Basis:** source\_backed

**Evidence:**

- arch-ev-fabric-handler
- arch-ev-fabric-deploy
- arch-ev-fabric-rest

## Evidence

### passport-ev-brd-flow

**Document ID:** VJIHTTYQXRCZZUNQ6BBFHQ7ESA

**File:** src/convert\_pipeline/tableau\_brd\_steps.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 292-345

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/convert_pipeline/tableau_brd_steps.py#L292-L345>)

### passport-ev-python-bridge

**Document ID:** CACGZF64IOQVUDI5ZWVEXVPGTF

**File:** dotnet/BiMigrationConverter.Orchestrator/interop/PythonBridge.cs

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 10-96

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/dotnet/BiMigrationConverter.Orchestrator/interop/PythonBridge.cs#L10-L96>)

### passport-ev-validation-convert

**Document ID:** 7XCZJJ5VAG256OZ22IPIMSNDRA

**File:** src/report\_validation\_pipeline/conversion.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 16-86

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/report_validation_pipeline/conversion.py#L16-L86>)

### passport-ev-validation-contract

**Document ID:** XM5KURGQB77V52UK4TO3TDTTPX

**File:** src/report\_validation\_pipeline/common.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 30-95

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/report_validation_pipeline/common.py#L30-L95>)

### passport-ev-validation-ci

**Document ID:** BOXLNYMO7TRB47TGAFPHAAAYTF

**File:** .github/workflows/report-validation-full-flow.yml

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 469-549

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/.github/workflows/report-validation-full-flow.yml#L469-L549>)

### passport-ev-cache-export

**Document ID:** CHXQMKNHQVV7N5SSYR6NOCYTYA

**File:** src/sources/powerbi/gen/cache\_export.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 92-184

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/sources/powerbi/gen/cache_export.py#L92-L184>)

### passport-ev-cache-caller

**Document ID:** 65VFJAVLHMHTBPXA6NXHK5ZFMF

**File:** dotnet/PowerBi.Source/PowerBiCacheStage.cs

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 8-68

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/dotnet/PowerBi.Source/PowerBiCacheStage.cs#L8-L68>)

### passport-ev-cli-upload

**Document ID:** R2X7DJWMRQLXRGY2NWGADDURPF

**File:** backend/cmd/antares-cli/analysis.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 219-325

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/antares-cli/analysis.go#L219-L325>)

### arch-ev-app-overview

**Document ID:** CPBAEV2JYBJLXX7TA5FKPBS7M2

**File:** README.md

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 3-27

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/README.md#L3-L27>)

### arch-ev-app-dev

**Document ID:** CPBAEV2JYBJLXX7TA5FKPBS7M2

**File:** README.md

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 68-115

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/README.md#L68-L115>)

### arch-ev-api

**Document ID:** NYGDT2PMSCTY5BXYFDS2QKTNHU

**File:** backend/cmd/api/main.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 47-132

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/api/main.go#L47-L132>)

### arch-ev-package

**Document ID:** PCKPLSRQGLANPQJ2U5WDYZLGIB

**File:** Dockerfile

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 4-50

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/Dockerfile#L4-L50>)

### arch-ev-pyproject

**Document ID:** PYVNTXP5ZQGZK6XECQJMWUN52E

**File:** pyproject.toml

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 1-44

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/pyproject.toml#L1-L44>)

### arch-ev-bridge

**Document ID:** CACGZF64IOQVUDI5ZWVEXVPGTF

**File:** dotnet/BiMigrationConverter.Orchestrator/interop/PythonBridge.cs

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 10-96

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/dotnet/BiMigrationConverter.Orchestrator/interop/PythonBridge.cs#L10-L96>)

### arch-ev-adapter

**Document ID:** NIYN6UW5G3RT2RSCUC26Q5EEMM

**File:** src/convert\_pipeline/adapter.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 159-211

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/convert_pipeline/adapter.py#L159-L211>)

### arch-ev-section

**Document ID:** VJIHTTYQXRCZZUNQ6BBFHQ7ESA

**File:** src/convert\_pipeline/tableau\_brd\_steps.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 292-345

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/convert_pipeline/tableau_brd_steps.py#L292-L345>)

### arch-ev-cache

**Document ID:** CHXQMKNHQVV7N5SSYR6NOCYTYA

**File:** src/sources/powerbi/gen/cache\_export.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 92-205

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/sources/powerbi/gen/cache_export.py#L92-L205>)

### arch-ev-cache-caller

**Document ID:** 65VFJAVLHMHTBPXA6NXHK5ZFMF

**File:** dotnet/PowerBi.Source/PowerBiCacheStage.cs

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 8-68

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/dotnet/PowerBi.Source/PowerBiCacheStage.cs#L8-L68>)

### arch-ev-culture

**Document ID:** UYKY7VXNGLSEDBVT4L3RFYS2ZB

**File:** src/data\_sniffer/resolver.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 86-219

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/data_sniffer/resolver.py#L86-L219>)

### arch-ev-culture-caller

**Document ID:** F3TB7YOG3NWRUWU5RAZ5PR6LWY

**File:** src/targets/powerbi/brd/data\_model\_preprocessor.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 1483-1533

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/targets/powerbi/brd/data_model_preprocessor.py#L1483-L1533>)

### arch-ev-verdict

**Document ID:** FBJDRFNWWBQNIEGENKQDQ5SY5H

**File:** src/data\_sniffer/verdict.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 14-25

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/data_sniffer/verdict.py#L14-L25>)

### passport-ev-analysis-create

**Document ID:** XQ4BB2W2UG5M73N7HBBOUATM2Z

**File:** backend/internal/api/handlers\_analyses.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 70-168

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/api/handlers_analyses.go#L70-L168>)

### passport-ev-upload-safety

**Document ID:** XQ4BB2W2UG5M73N7HBBOUATM2Z

**File:** backend/internal/api/handlers\_analyses.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 1695-1751

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/api/handlers_analyses.go#L1695-L1751>)

### passport-ev-bundles

**Document ID:** CU7KWMZXUFC7OSCS4HKORCQ6QN

**File:** src/targets/aibi/deploy/bundle\_builder.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 818-929

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/targets/aibi/deploy/bundle_builder.py#L818-L929>)

### passport-ev-bundle-caller

**Document ID:** Y4QY6K5K2ZE6V2BO7MNUYM6O7S

**File:** src/targets/aibi/pipeline/gen\_steps.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 687-759

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/targets/aibi/pipeline/gen_steps.py#L687-L759>)

### passport-ev-dev-entry

**Document ID:** CPBAEV2JYBJLXX7TA5FKPBS7M2

**File:** README.md

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 68-115

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/README.md#L68-L115>)

### passport-ev-backend-build

**Document ID:** MVY7W7IWPUC3L6NATW4I2EGNYO

**File:** backend/Justfile

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 42-90

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/Justfile#L42-L90>)

### passport-ev-validator

**Document ID:** 6C6TPOCWKOSSAIA7Q72367IRF3

**File:** src/helpers\_scripts/validate\_json.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 162-212

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/helpers_scripts/validate_json.py#L162-L212>)

### arch-ev-profiler

**Document ID:** REF5QSEQ2MFG5HSUTSPWRHXLPX

**File:** src/tableau\_profiler/main.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 980-1102

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/tableau_profiler/main.py#L980-L1102>)

### arch-ev-profiler-call

**Document ID:** HLDILOIOKYOWIJWXN7K5LUQUCO

**File:** src/convert\_pipeline/tableau\_processing.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 236-247

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/convert_pipeline/tableau_processing.py#L236-L247>)

### arch-ev-profiler-schema

**Document ID:** SFRNP5BJCDEEJ77ANKSQ3WE7HG

**File:** src/tableau\_profiler/core/models.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 18-22

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/tableau_profiler/core/models.py#L18-L22>)

### arch-ev-validation-suite

**Document ID:** 3ZPDSFYM6DOG372ETV52KU2FQN

**File:** src/report\_validation/cli/commands.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 212-225

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/report_validation/cli/commands.py#L212-L225>)

### arch-ev-validation-call

**Document ID:** BRZBNIVRMWNMAEN6KLNT4FC4NF

**File:** src/report\_validation\_pipeline/comparison.py

**Repository or context:** https://github.com/Antares-BI/bi-migration-converter

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Lines:** 39-89

[View source](<https://github.com/Antares-BI/bi-migration-converter/blob/7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923/src/report_validation_pipeline/comparison.py#L39-L89>)

### arch-ev-av

**Document ID:** KA5M2VTEHWE3FREKPF54KQDJK6

**File:** backend/internal/avscan/scanner.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 27-124

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/avscan/scanner.go#L27-L124>)

### arch-ev-av-call

**Document ID:** XQ4BB2W2UG5M73N7HBBOUATM2Z

**File:** backend/internal/api/handlers\_analyses.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 1695-1751

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/api/handlers_analyses.go#L1695-L1751>)

### arch-ev-cli

**Document ID:** R2X7DJWMRQLXRGY2NWGADDURPF

**File:** backend/cmd/antares-cli/analysis.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 219-325

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/antares-cli/analysis.go#L219-L325>)

### arch-ev-cli-entry

**Document ID:** HRKUGAA4WSSBVATXY3V6VCO2M4

**File:** backend/cmd/antares-cli/main.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 13-43

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/antares-cli/main.go#L13-L43>)

### arch-ev-sqlite

**Document ID:** E6SNVEGO5NCYGLLYBBTRWM33IT

**File:** backend/internal/db/database.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 10-58

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/db/database.go#L10-L58>)

### arch-ev-queue

**Document ID:** EALEBCCL75BBIXTT5VK5EXY5X4

**File:** backend/internal/queue/queue.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 20-67

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/queue/queue.go#L20-L67>)

### arch-ev-worker

**Document ID:** NYGDT2PMSCTY5BXYFDS2QKTNHU

**File:** backend/cmd/api/main.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 122-155

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/api/main.go#L122-L155>)

### arch-ev-worker-executor

**Document ID:** CY2RS24HLF7UX76P6TFTLLEWVF

**File:** backend/internal/worker/worker.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 25-110

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/worker/worker.go#L25-L110>)

### arch-ev-worker-report

**Document ID:** MHEGWOBPKXJYBMN66VDU7HR555

**File:** backend/internal/worker/analysis\_helpers.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 516-540

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/worker/analysis_helpers.go#L516-L540>)

### arch-ev-artifacts-raw

**Document ID:** 6OPYQAJJAA6RZGUEXDNS6AKE7Q

**File:** backend/internal/storage/analysis\_storage.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 274-334

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/storage/analysis_storage.go#L274-L334>)

### arch-ev-artifacts-index

**Document ID:** 6OPYQAJJAA6RZGUEXDNS6AKE7Q

**File:** backend/internal/storage/analysis\_storage.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 544-594

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/storage/analysis_storage.go#L544-L594>)

### arch-ev-events

**Document ID:** ZJZ53WATL7B5ARXDPWICKWOV3B

**File:** backend/internal/jobevents/manager.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 194-288

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/jobevents/manager.go#L194-L288>)

### arch-ev-sse

**Document ID:** XQ4BB2W2UG5M73N7HBBOUATM2Z

**File:** backend/internal/api/handlers\_analyses.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 638-707

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/api/handlers_analyses.go#L638-L707>)

### arch-ev-web-bootstrap

**Document ID:** STAQ3WPMB2L7ZBDOHU5BS26JG6

**File:** frontend/src/main.tsx

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 1-19

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/frontend/src/main.tsx#L1-L19>)

### arch-ev-web-client

**Document ID:** YMIOREULKMJWURHMQQXNTNFBJT

**File:** frontend/src/services/api.ts

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 275-335

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/frontend/src/services/api.ts#L275-L335>)

### arch-ev-web-create

**Document ID:** YMIOREULKMJWURHMQQXNTNFBJT

**File:** frontend/src/services/api.ts

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 1014-1034

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/frontend/src/services/api.ts#L1014-L1034>)

### arch-ev-desktop-lifecycle

**Document ID:** 2FTY4DP43DEJ4I734ZQZB3LA3J

**File:** desktop/src-tauri/src/lib.rs

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 25-59

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/desktop/src-tauri/src/lib.rs#L25-L59>)

### arch-ev-desktop-process

**Document ID:** 2FTY4DP43DEJ4I734ZQZB3LA3J

**File:** desktop/src-tauri/src/lib.rs

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 83-103

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/desktop/src-tauri/src/lib.rs#L83-L103>)

### arch-ev-gateway

**Document ID:** LIEVHJTIUZ5NSUTJ6KOKIRKWPC

**File:** backend/internal/llmgateway/proxy.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 71-220

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/llmgateway/proxy.go#L71-L220>)

### arch-ev-gateway-start

**Document ID:** NYGDT2PMSCTY5BXYFDS2QKTNHU

**File:** backend/cmd/api/main.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 170-178

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/cmd/api/main.go#L170-L178>)

### arch-ev-fabric-handler

**Document ID:** HEXZIWSKDYV5PVE3IVJD7BYMIT

**File:** backend/internal/api/handlers\_conversions\_fabric\_deploy.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 59-97

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/api/handlers_conversions_fabric_deploy.go#L59-L97>)

### arch-ev-fabric-deploy

**Document ID:** BKDKCQVUUXPMXAV4NWEMUYV2NU

**File:** backend/internal/fabric/service.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 834-908

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/fabric/service.go#L834-L908>)

### arch-ev-fabric-rest

**Document ID:** JEKSP5OFET7B2UWY4KY4TRIIB7

**File:** backend/internal/fabric/client.go

**Repository or context:** https://github.com/Antares-BI/antares-app

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Lines:** 348-427

[View source](<https://github.com/Antares-BI/antares-app/blob/f281dccd24b538f8d2a8b314570744c158581295/backend/internal/fabric/client.go#L348-L427>)

## Open questions

**Unresolved:**

- External Tableau profiler producer wiring is not established by BRD consumer evidence.
- Research inspected test sources but did not execute tests; API integration wiring stubs profiler binaries and does not validate actual analysis runtime.
- Documented input/target product coverage in repository overview was not independently verified endpoint-by-endpoint.

## Source snapshots

### https://github.com/Antares-BI/bi-migration-converter

**Snapshot ID:** 6TY2IQ6ISUPBELSJ7XXMDK5UTL

**Commit:** 7ccf4b0d74f01621d1e79e0e1d01a2266dd2d923

**Coverage note:** Repository text indexed at the pinned SHA. Build outputs, dependencies, binary files and secret-like files are excluded. Available files are not automatically read by the model. Reused the safe source package at the unchanged commit SHA.

**Indexed files:** 7760

**Files read:** 96

**Files cited:** 42

### https://github.com/Antares-BI/antares-app

**Snapshot ID:** BKGS34AUY6EEI77GPWANQKPY3S

**Commit:** f281dccd24b538f8d2a8b314570744c158581295

**Coverage note:** Repository text indexed at the pinned SHA. Build outputs, dependencies, binary files and secret-like files are excluded. Available files are not automatically read by the model. Reused the safe source package at the unchanged commit SHA.

**Indexed files:** 1045

**Files read:** 204

**Files cited:** 40

## Product branding

**Accent color:** \#2563eb

**Background color:** \#fafafa

**Logo URL:** data:image/png;base64,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

**https://getantares.io/brand/agents.md:** https://getantares.io/brand/agents.md

**Product logo:** https://getantares.io/brand/mark/antares-mark-blue-transparent-1024.png


