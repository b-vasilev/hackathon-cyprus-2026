# CLI-Anything

**Tagline:** Agent-generated application CLIs, with catalog-based discovery and installation.

CLI-Anything combines coding-agent generation specifications, a local Python CLI Hub, discovery skills and independently installed application CLI harnesses. Agents generate namespace-packaged command-line tools with structured output and interactive interfaces; users and agents can also discover and install existing harnesses. Harnesses use varied local executables, libraries, data files and remote APIs rather than a shared application server. The model records representative workflows and source-backed operations at repository revision 34f519533bc175d2fe287ab8316b0dd99bb9cc43; documented checks and inspected tests are not claimed as executed.

**Product ID:** ZKX3MR2W6BVBNSP4JBH7AX4HEZ

**Passport revision:** IGABBYNHKRNUZ24YEJNX3CTIBW

**Version:** 1

**Generated at:** 2026-10-16T12:38:57.002569Z

**Published at:** 2026-10-16T12:39:35.34138Z

**Generation run:** LAJXGJ2LF7S2QW4ZDFSOSLP47L

**Schema version:** 1

**Document version:** 2

## Product overview

**Chapter ID:** overview

**Question:** What is CLI-Anything, who uses it, what results does it deliver, and where does its responsibility end?

CLI-Anything provides tooling for users and coding agents to generate application command-line harnesses or discover and install existing ones through a local Hub. A harness exposes application tasks through command-line commands, structured JSON output and interactive use.

There are two main outcomes:
- **Generate an installable application CLI:** a coding agent follows a generation specification using a local source path or GitHub URL. The workflow prescribes validation and local installation; the specification alone does not establish successful execution.
- **Use an existing harness:** users or agents discover an application harness through the Hub and install it locally for application tasks.

These are independently installed local tools, not clients of a common application server. Their responsibilities and dependencies vary: they use separately provisioned application engines, libraries, data or remote services. There is no universal backend contract.

Completion also depends on the command. Some commands produce local artifacts. Blender export produces a render script and suggested invocation, not a completed render. A remote task can complete without a local download; download behavior is conditional and distinct from remote output.

The next section explains source ownership and the distribution contracts connecting generation, discovery and installed harnesses.

**Claims:**

- arch-claim-purpose
- arch-claim-generator
- arch-claim-hub
- arch-claim-harness-interface
- arch-claim-backend-diversity
- passport-generation
- passport-blender-script
- passport-anygen-task

## Repositories and connections

**Chapter ID:** repositories

**Question:** Which repository owns CLI-Anything’s source responsibilities, and which package, registration and catalog contracts connect it to consumers?

**HKUDS/CLI-Anything is the only supplied repository**, at revision `34f519533bc175d2fe287ab8316b0dd99bb9cc43`. It contains agent generation specifications and host adapters, Hub discovery/installation code, application harness packages, and discovery skills with mirror validation. The validation source was inspected, not executed.

Three distinct contracts connect this source to consumers:

- **Host registration:** Claude Code and Cursor marketplace declarations point to `./cli-anything-plugin` and `./cursor-plugin`, respectively. These are declarative registrations, not running services. Pi and Codex are additional adapter variants; external agent-host implementations are not supplied.
- **Package distribution:** Agent specifications prescribe Python namespace packages under `cli_anything.<software>`, `cli-anything-<software>` entry points and local editable installation. Application harnesses are independently installed packages, not a shared runtime service.
- **Catalog consumption:** Hub exposes `cli-hub`, retrieves the two published HTTPS JSON catalogs, `registry.json` and `public_registry.json`, and installs selected harnesses. Consumer behavior is evidenced, but the selected model does not establish the catalog publishing pipeline. Discovery skills guide agents through Hub discovery and installation.

Registration, package installation and catalog retrieval are separate contracts; none establishes a verified cross-repository dependency. A complete source-root classification is not supplied, so these responsibilities should not be read as exhaustive source-area coverage.

**Claims:**

- arch-claim-purpose
- arch-claim-generator
- arch-claim-registration
- arch-claim-variants
- arch-claim-skills
- arch-claim-hub
- arch-claim-harness-interface
- passport-skill-mirror-validation

**Table style:** repositories

| ID | Name | Type | Technology | Responsibility | Status |
| --- | --- | --- | --- | --- | --- |
| arch-repository | HKUDS/CLI-Anything | repository | Git repository | Source provenance: https://github.com/HKUDS/CLI-Anything at revision 34f519533bc175d2fe287ab8316b0dd99bb9cc43. | implemented |

**Claims:**

- arch-claim-purpose
- arch-claim-generator
- arch-claim-registration
- arch-claim-variants
- arch-claim-hub
- arch-claim-skills
- arch-claim-harness-interface
- passport-skill-mirror-validation

## Logical architecture

**Chapter ID:** architecture

**Question:** What are CLI-Anything&\#39;s executable and declarative boundaries, how do they connect, and which contracts vary across application harnesses?

CLI-Anything combines **agent-interpreted specifications** with **local executable tools**, rather than a shared application server. Generation specifications tell coding agents how to produce application CLI harnesses; discovery skills guide agents to the Hub’s discovery and installation commands. Neither is a peer runtime alongside the tools it describes.

The executable boundaries are the local **CLI Hub**, which handles discovery and package installation, and **independently installed application CLI harnesses**, which accept application commands. Harnesses share an outward CLI interface but connect to application engines, services or local data through implementation-specific contracts—not a universal backend protocol.

The following structural view shows this runtime portion of the system. Generation and discovery specifications are explained separately because they guide agent behavior rather than execute as services.

**Claims:**

- arch-claim-purpose
- arch-claim-generator
- arch-claim-hub
- arch-claim-skills
- arch-claim-harness-interface
- arch-claim-backend-diversity

**Diagram (see Diagrams):** architecture-part-2-diagram

### Agent specifications produce independently consumable harnesses

The generator is an **agent-interpreted specification, not a running generation service**. A coding-agent host receives a local application source path or GitHub URL and follows instructions to analyze the application and produce a Python namespace package. The prescribed outputs include a Click command interface, interactive REPL, structured JSON output, tests and discovery skills. The resulting harness is independently installable and exposes a `cli-anything-<software>` command.

Discovery skills serve a different purpose: they guide compatible agents to the Hub’s `list`, `search`, `info` and `install` commands to find and consume existing harnesses rather than generate new ones.

| Host adapter | Integration contract |
|---|---|
| Claude Code and Cursor | Declarative marketplace registrations point to their plugin directories. Registration does not establish execution or end-to-end support. |
| Pi | A TypeScript extension injects the harness specification context through `sendUserMessage`. |
| Codex | Documented as **Experimental/Community**, with skill installation that preserves the generated Python harness format. |

The specification prescribes testing and local editable installation as workflow steps; these are **workflow intent, not observed successful runs**.

**Claims:**

- arch-claim-generator
- arch-claim-registration
- arch-claim-skills
- arch-claim-harness-interface
- arch-claim-variants
- passport-generation

### The Hub owns discovery and installation state

The CLI Hub is a local discovery and package-management tool, exposed through `cli-hub=cli_hub.cli:main`. Its responsibility is to find available harnesses and manage their installation—not to execute application commands.

The Hub consumes two published JSON catalogs, `registry.json` and `public_registry.json`, over HTTPS. Each has a separate local cache with a one-hour lifetime. These caches hold discovery information; they are distinct from the local installed-package metadata that records successful installations.

For installation, the Hub resolves the selected catalog entry and dispatches package installation. It persists installed-state metadata only after the installer succeeds. This separates catalog availability from installation state: a discoverable entry is not evidence that its package is installed.

Successful installation establishes package-management completion, not successful application execution or backend compatibility. Those remain outside the Hub’s installation contract.

**Claims:**

- arch-claim-hub
- passport-catalog
- passport-install

Application harnesses share an outward CLI contract, not a backend protocol. Each is independently installed as a `cli-anything-<software>` command, with help, structured JSON output and an interactive REPL. Behind that interface, application commands cross different execution and state boundaries.

| Harness | Backend contract and completion boundary |
| --- | --- |
| Draw.io | Invokes an export subprocess with input, output and format arguments. Completion requires both subprocess success and output. |
| ChromaDB | Uses HTTP under `/api/v2`; documented defaults are `localhost:8000`, `default_tenant` and `default_database`. These defaults are not a backend-version guarantee. |
| CloudAnalyzer | Calls `ca.evaluate.evaluate` in process through an external Python library, rather than invoking a separate application service. |
| CC Switch | Accesses SQLite and configuration directly, rather than delegating through a common service API. |
| Blender export | Generates `_render_script.py` and returns script/output paths plus a suggested Blender invocation; it does not execute Blender. |

These are representative harness strategies, not a universal backend abstraction or additional shared components. In particular, Blender export completes at script generation: producing the script is not proof that rendering ran or that a rendered image exists.

**Claims:**

- arch-claim-harness-interface
- arch-claim-backend-diversity
- passport-blender-script

## End-to-end flows

**Chapter ID:** scenarios

**Question:** How do the main outcomes happen, and which material variants change stages, contracts or acceptance?

Outcomes come from agent-directed harness creation, Hub installation, or application-specific command execution. **Completion depends on each workflow’s contract:** an installed package with recorded metadata, a conversion file checked for existence, a generated Blender script rather than a rendered image, a remote task result with optional download, or a written audio file are different outcomes.

The generation workflow prescribes work for an agent; it is not evidence of completed generation or passed tests. The other flows describe inspected implementations, with mocked Hub tests also inspected. No execution verification was performed during research.

**Claims:**

- passport-generation
- passport-install
- passport-install-validation
- passport-calibre-convert
- passport-blender-script
- passport-anygen-task
- passport-audacity-format

These local-artifact workflows share a file-oriented outcome, but their acceptance contracts differ. The matrix describes inspected implementations, not observed successful runs.

| Family and trigger | Ordered stages and material variants | Implemented terminal outcome or acceptance boundary |
|---|---|---|
| **Calibre conversion:** book and requested input/output formats | Export the input to temporary storage → invoke `ebook-convert` with optional extra arguments → check output existence → optionally reimport the converted format into the original book. | Returns the converted file’s path, size and library-update status. Missing output fails the existence check; reimport is optional. |
| **Blender script generation:** scene project and export/frame options | Generate `_render_script.py` beside the requested output → return script/output paths, render metadata and a suggested Blender invocation. | Completion means a generated script and invocation, not a rendered image. This path does not execute Blender. |
| **Audacity audio export:** project, output path and preset/settings | Select tracks, skipping muted or excluded non-solo tracks → render clips and effects → mix volume/pan → clamp samples → write WAV and measure metadata. | Produces WAV data with duration, file size, peak and RMS. A non-WAV preset still writes WAV bytes, so the reported format can disagree with the encoding. |

**Claims:**

- passport-calibre-convert
- passport-blender-script
- passport-audacity-format

**Scenario (see Scenarios):** passport-generate-harness

**Scenario (see Scenarios):** passport-install-harness

**Scenario (see Scenarios):** passport-cloud-generation

## Technologies and tools

**Chapter ID:** technologies

**Question:** Which technologies and tools implement CLI-Anything&\#39;s responsibilities, in which scopes, and what do the available version and usage evidence establish?

CLI-Anything combines agent-readable Markdown specifications, a local Python Hub, independently installed Python harnesses and backend-specific tools or services—not a common application server. Runtime responsibilities are separate from the generated packaging and prescribed pytest checks.

The Hub declares **Python >=3.10, Click >=8.0 and requests >=2.28**. These are requirement bounds, not observed installed versions.

For a scoped example, **cli-anything-calibre 1.0.0** declares Python >=3.10, Click >=8.0.0 and prompt-toolkit >=3.0.0. Its external Calibre executables need separate provisioning on PATH; backend-version negotiation is not demonstrated. These requirements should not be generalized to every harness.

Within the generator scope, Pi uses a TypeScript adapter to inject generation context, while Codex provides an **Experimental/Community** skill route that preserves the generated Python harness format. Marketplace registration alone establishes neither execution nor pinned host versions.

Generated packages include command entry points, tests and discovery skills. Inspected tests and generation instructions are not executed checks.

The complete reference below maps technologies to their supported responsibilities and scopes, including aggregate components and output artifacts rather than treating every entry as a separately deployed tool.

**Claims:**

- arch-claim-generator
- arch-claim-registration
- arch-claim-hub
- arch-claim-harness-interface
- arch-claim-backend-diversity
- arch-claim-variants
- passport-calibre-runtime
- passport-generation
- passport-install-validation

**Table style:** technologies

| ID | Name | Type | Technology | Responsibility | Status |
| --- | --- | --- | --- | --- | --- |
| arch-product | CLI-Anything | component | Python CLI packages; coding-agent Markdown specifications | Application CLI generation and consumption ecosystem; distributed local tooling rather than a single hosted runtime. | implemented |
| arch-repository | HKUDS/CLI-Anything | repository | Git repository | Source provenance: https://github.com/HKUDS/CLI-Anything at revision 34f519533bc175d2fe287ab8316b0dd99bb9cc43. | implemented |
| arch-generator | Agent harness generator plugins | component | Markdown agent commands; JSON plugin descriptors | Agent-consumed generation specification with host-specific marketplace registrations, not an autonomous runtime service. | implemented |
| arch-agent-hosts | Coding-agent hosts | external\_system | External coding-agent environments | Claude Code, Cursor and SKILL-compatible agents interpret generator/discovery instructions; host internals are outside this repository. | implemented |
| arch-discovery-skills | Agent discovery skills | component | SKILL.md | Agent-readable guidance for locating and consuming catalog harnesses. | implemented |
| passport-harness-package | Generated harness package and discovery skills | entity | Python, Click, pytest, Markdown, setup.py | Generated Python namespace package, console entry point, tests and canonical/compatibility SKILL.md artifacts prescribed by the agent workflow. | implemented |
| arch-hub | CLI Hub | component | Python &gt;=3.10; Click &gt;=8.0; requests &gt;=2.28 | Local discovery and package-management CLI. | implemented |
| arch-catalog-service | Published Hub catalogs | external\_system | HTTPS; JSON; GitHub Pages | GitHub Pages-hosted JSON catalogs consumed by Hub. | implemented |
| passport-catalog-artifacts | Public CLI catalogs and local caches | entity | HTTPS, JSON | Published JSON catalogs consumed by Hub and cached separately for one hour. | implemented |
| passport-installed-metadata | Hub installed-package metadata | entity | JSON | Local installed-state metadata persisted after successful installation. | implemented |
| arch-harnesses | Application CLI harnesses | component | Python; Click; JSON; REPL | Independently installed application-specific CLI runtimes, not a shared application server. | implemented |
| arch-app-backends | Application engines and services | external\_system | Executable subprocesses; HTTP/JSON; Python library calls | Separately provisioned application dependencies, represented by Draw.io desktop CLI, ChromaDB REST service and CloudAnalyzer Python engine. Harness behavior and compatibility vary by backend; there is no universal backend protocol. | implemented |
| passport-converted-book | Converted book artifact | entity | Calibre ebook-convert | Converted book file and metadata, optionally reimported into the original Calibre library. | implemented |
| passport-render-script | Blender render script artifact | entity | Python, Blender command line | Generated Python script and suggested Blender command; not a completed rendered image. | implemented |
| passport-cloud-task-result | Cloud generation task and optional download | entity | Remote task API | AnyGen task identifier, URL, status and remote output, with conditional local file metadata. | implemented |
| passport-audio-artifact | Audio export and measured metadata | entity | WAV audio | Rendered WAV data with duration, size, peak and RMS; reported non-WAV format may differ from file encoding. | implemented |

**Claims:**

- arch-claim-purpose
- arch-claim-generator
- arch-claim-registration
- arch-claim-skills
- arch-claim-hub
- arch-claim-harness-interface
- arch-claim-backend-diversity
- arch-claim-variants
- passport-generation
- passport-catalog
- passport-install
- passport-install-validation
- passport-calibre-convert
- passport-calibre-runtime
- passport-blender-script
- passport-anygen-task
- passport-audacity-format

## Data and artifacts

**Chapter ID:** data

**Question:** What information and artifacts does CLI-Anything exchange, persist and validate, and what does each output actually establish?

CLI-Anything exchanges generated packages, discovery catalogs and local installation records rather than one shared application data model.

| Information | Origin and consumer | Contract and persistence boundary |
|---|---|---|
| Harness package and skills | The agent-directed specification prescribes a `cli_anything.<software>` namespace package, a `cli-anything-<software>` console entry point, tests and canonical/compatibility `SKILL.md` artifacts for users and agents. | These are prescribed generation outputs, not evidence that generation, installation or tests actually ran. |
| Published catalogs and cached entries | Hub reads published `registry.json` and `public_registry.json` discovery data over HTTPS. | Each catalog has a separate local cache with a one-hour TTL. Published discovery information is distinct from local installed-state records. |
| Installed-state JSON | Hub records installation metadata only after the installer reports success. | This records a successful installation event; it does not establish continuous verification of the environment. The selected evidence does not establish retention, deletion, backup or consistency guarantees. |

Inspected Hub test definitions cover installed-state JSON roundtrip, mocked successful pip installation, unknown names and subprocess failures. They establish what the tests check, not passing results: these tests were not executed during research.

**Claims:**

- passport-generation
- passport-catalog
- passport-install
- passport-install-validation

An output may establish file existence, script generation, remote state or audio encoding—not downstream acceptance. These are inspected implementation contracts, not verified run results.

| Artifact | Origin and consumer | Validation or invariant | Persistence or acceptance boundary |
|---|---|---|---|
| Calibre converted file | Harness exports the source format temporarily and invokes `ebook-convert`; the original library is an optional consumer. | Checks output existence; returns artifact metadata. | Writes the converted file; optional reimport targets the original library. Existence alone does not prove downstream acceptance. |
| Blender `_render_script.py` | Harness generates the script from a scene project; Blender is the intended consumer via a suggested invocation. | Returns script/output paths and a Blender command. | Script creation does not execute Blender, prove script acceptance or establish a rendered image. |
| AnyGen task result | Harness creates and polls a remote task; callers receive task state and remote output. | Download requires an output directory and operation `slide`, `doc` or `smart_draw`. | Remote completion is not local materialization; local file metadata is conditional on download. |
| Audacity audio export | Harness renders tracks/effects, mixes and clamps samples, and writes audio for callers. | Reports duration, file size, peak and RMS. | Written bytes are WAV even when a non-WAV preset is reported: reported format and actual encoding can disagree. |

**Claims:**

- passport-calibre-convert
- passport-blender-script
- passport-anygen-task
- passport-audacity-format

## Runtime and deployment

**Chapter ID:** runtime

**Question:** What runs where, how does it start, and which modes and configuration does it require?

Execution is distributed across coding-agent hosts, a local Hub command and separately installed harness commands—not one hosted application server. Coding-agent hosts interpret the generator specifications. Marketplace entries register plugins for those hosts; the registrations are not running services.

The Hub starts through `cli-hub=cli_hub.cli:main` and requires **Python >=3.10**. Its runtime dependency is the published HTTPS JSON catalogs, `registry.json` and `public_registry.json`.

| Local data | Purpose and lifetime |
|---|---|
| Catalog caches | Separate caches for the two published catalogs, each with a one-hour time to live. These hold discovery data, not installation records. |
| Installed-package metadata | Local installed-state records saved only after package installation succeeds; distinct from cached catalog data. |

These are source and configuration facts, not observed deployment results. The supplied evidence does not establish production placement or service-readiness behavior.

**Claims:**

- arch-claim-generator
- arch-claim-registration
- arch-claim-hub
- passport-catalog
- passport-install

Harnesses run as independently installed `cli-anything-<software>` commands with command-line, structured JSON and interactive REPL interfaces. Runtime dependencies vary by harness; there is no universal backend protocol.

| Execution mode | Representative dependency and boundary |
|---|---|
| Local executable subprocess | Draw.io requires its desktop CLI for export; subprocess success and an output file are required. |
| In-process library or direct data access | CloudAnalyzer calls the external Python engine through `ca.evaluate.evaluate`; CC Switch accesses SQLite/configuration directly. These are not shared application services. |
| External HTTP service | ChromaDB requires a reachable REST service using `/api/v2`; its default endpoint is `localhost:8000`. |
| Artifact-only script generation | Blender export creates `_render_script.py`, paths and a suggested Blender command. Creating the script does not execute Blender or establish that an image was rendered. |

**Calibre’s runtime contract:** Python >=3.10, Click >=8.0.0 and prompt-toolkit >=3.0.0, plus external Calibre tools installed on `PATH`. `CALIBRE_OVERRIDE_LANG=en` stabilizes output parsing. Missing tools produce installation guidance; backend-version negotiation is not demonstrated.

**AnyGen’s remote boundary:** the harness creates and polls a remote task. Task status and remote output are distinct from a local download, which occurs only when an output directory is supplied for `slide`, `doc` or `smart_draw` operations.

Shutdown/resource-cleanup guarantees and production placement are not established for these representative modes.

**Claims:**

- arch-claim-harness-interface
- arch-claim-backend-diversity
- passport-calibre-runtime
- passport-blender-script
- passport-anygen-task

## Development guide

**Chapter ID:** development

**Question:** Which supported work path should I follow, and how can I check my change?

Choose the generator specification for generation behavior, the Hub implementation for discovery/install changes, or generated package and harness skill sources for application-specific changes.

| Work path and entry point | Prerequisites and smallest supported checks |
|---|---|
| **Generator/plugins:** registered sources `cli-anything-plugin` and `cursor-plugin`. These contain agent instructions and plugin registration, not executable services. | Claude Code documentation prescribes `/reload-plugins` and `/help cli-anything` as **manual load checks**. These are documented checks, not observed successful results. |
| **Hub:** start at `cli_hub.cli:main` for the local discovery/package-management CLI. | Requires Python **>=3.10**. Inspected installation test definitions cover metadata JSON roundtrip, mocked pip success, unknown names and subprocess failures. An exact test invocation and a successful run are **not established** by the selected evidence. |
| **Generated harness/skills:** locate the `cli_anything.<software>` namespace, `setup.py` console entry point and harness skill sources. | The generator specification prescribes creating/running tests and editable installation using `pip install -e .` **from the generated package directory**; installation changes the local Python environment. For skill changes, the inspected mirror validator compares root mirrors with harness skill sources after canonical frontmatter rewriting, detects missing/stale mirrors and root-only edits, and exits nonzero on mismatch. Its invocation is not established here; it was inspected, not executed. |

Prescribed workflow steps and inspected test definitions do not establish successful execution.

**Claims:**

- arch-claim-generator
- arch-claim-registration
- passport-plugin-smoke-check
- arch-claim-hub
- passport-install-validation
- passport-generation
- passport-skill-mirror-validation

## Operations and recovery

**Chapter ID:** operations

**Question:** How can maintainers diagnose important failures, what recovery guidance is supported, and where is repeat-operation safety unknown?

Diagnose the affected plugin, installer or application boundary before repeating an operation. Evidence supports limited recovery guidance, not a general recovery guarantee.

| Boundary/failure | Signal | Action | Safety condition | Result check |
|---|---|---|---|---|
| Plugin loading | Plugin unavailable in the host | Use documented `/reload-plugins` and `/help cli-anything`; documentation also prescribes marketplace re-add/install recovery. | These are prescribed procedures, not observed successes. | Check whether help exposes the command; no successful check was observed. |
| Hub installation | Unknown catalog name versus installer subprocess failure | Distinguish name resolution from installer failure before considering another attempt. | Rollback, partial-install cleanup and safe retry are not established. | Metadata is written only after installer success. Inspected mocked tests were not run. |
| Calibre tools/conversion | Missing tools produce installation guidance; converted output may be absent. | Check required tools on `PATH` separately from converted-output existence. | Optional reimport changes the original library; safe repeated reimport is not established. | Output existence is checked, but alone does not establish library-update success; distinguish it from returned add-to-library status. |
| AnyGen task/download | Remote task status and optional local-file metadata are distinct. | Inspect remote state first; assess absent local files only when an output directory was supplied and the operation is `slide`, `doc` or `smart_draw`. | Recovery and duplicate-task safety are not established; resubmission is not prescribed. | Check remote status/output separately from local download metadata. |

**Claims:**

- passport-plugin-smoke-check
- passport-install
- passport-install-validation
- passport-calibre-runtime
- passport-calibre-convert
- passport-anygen-task

## Objects

### Generated harness package and discovery skills

**ID:** passport-harness-package

**Type:** entity

Generated Python namespace package, console entry point, tests and canonical/compatibility SKILL.md artifacts prescribed by the agent workflow.

**Technology:** Python, Click, pytest, Markdown, setup.py

**Status:** implemented

**Inputs:**

- Application source path or GitHub URL

**Outputs:**

- cli\_anything.&lt;software&gt; package
- cli-anything-&lt;software&gt; command
- SKILL.md

**Contracts:**

- Click command interface, REPL and --json
- Local editable pip installation

**Claims:**

- passport-generation

### Public CLI catalogs and local caches

**ID:** passport-catalog-artifacts

**Type:** entity

Published JSON catalogs consumed by Hub and cached separately for one hour.

**Technology:** HTTPS, JSON

**Status:** implemented

**Inputs:**

- registry.json
- public\_registry.json

**Outputs:**

- Cached catalog entries

**Contracts:**

- https://hkuds.github.io/CLI-Anything/registry.json
- https://hkuds.github.io/CLI-Anything/public\_registry.json

**Claims:**

- passport-catalog

### Hub installed-package metadata

**ID:** passport-installed-metadata

**Type:** entity

Local installed-state metadata persisted after successful installation.

**Technology:** JSON

**Status:** implemented

**Inputs:**

- Successful package installation

**Outputs:**

- Installed-state record

**Contracts:**

- Persist only after installer success

**Claims:**

- passport-install
- passport-install-validation

### CLI-Anything

**ID:** arch-product

**Type:** component

Application CLI generation and consumption ecosystem; distributed local tooling rather than a single hosted runtime.

**Technology:** Python CLI packages; coding-agent Markdown specifications

**Status:** implemented

**Architecture role:** software\_system

**Inputs:**

- Application source path or URL
- User or agent tasks

**Outputs:**

- Installable application CLI harnesses
- Discovered and installed harnesses

**Claims:**

- arch-claim-purpose

### HKUDS/CLI-Anything

**ID:** arch-repository

**Type:** repository

Source provenance: https://github.com/HKUDS/CLI-Anything at revision 34f519533bc175d2fe287ab8316b0dd99bb9cc43.

**Technology:** Git repository

**Status:** implemented

**Claims:**

- arch-claim-purpose

### Agent harness generator plugins

**ID:** arch-generator

**Type:** component

Agent-consumed generation specification with host-specific marketplace registrations, not an autonomous runtime service.

**Technology:** Markdown agent commands; JSON plugin descriptors

**Status:** implemented

**Architecture role:** component

**Parent:** arch-product

**Inputs:**

- Local application source path
- GitHub repository URL

**Outputs:**

- Click harness package
- Tests
- SKILL.md

**Contracts:**

- /cli-anything &lt;source&gt;
- cli\_anything.&lt;software&gt; namespace; setup.py console\_scripts; pip install -e .

**Claims:**

- arch-claim-generator
- arch-claim-registration

### CLI Hub

**ID:** arch-hub

**Type:** component

Local discovery and package-management CLI.

**Technology:** Python &gt;=3.10; Click &gt;=8.0; requests &gt;=2.28

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-product

**Inputs:**

- Catalog names and queries

**Outputs:**

- Catalog results
- Installed harnesses
- Installation metadata

**Contracts:**

- cli-hub=cli\_hub.cli:main

**Claims:**

- arch-claim-hub

### Agent discovery skills

**ID:** arch-discovery-skills

**Type:** component

Agent-readable guidance for locating and consuming catalog harnesses.

**Technology:** SKILL.md

**Status:** implemented

**Architecture role:** component

**Parent:** arch-product

**Inputs:**

- Agent task

**Outputs:**

- Hub discovery and installation instructions

**Contracts:**

- cli-hub list/search/install/info

**Claims:**

- arch-claim-skills

### Coding-agent hosts

**ID:** arch-agent-hosts

**Type:** external\_system

Claude Code, Cursor and SKILL-compatible agents interpret generator/discovery instructions; host internals are outside this repository.

**Technology:** External coding-agent environments

**Status:** implemented

**Inputs:**

- Plugin registrations
- Skills
- User commands

**Outputs:**

- Agent-driven generation or consumption

**Contracts:**

- Host plugin loading
- SKILL compatibility

**Claims:**

- arch-claim-registration
- arch-claim-skills

### Published Hub catalogs

**ID:** arch-catalog-service

**Type:** external\_system

GitHub Pages-hosted JSON catalogs consumed by Hub.

**Technology:** HTTPS; JSON; GitHub Pages

**Status:** implemented

**Outputs:**

- registry.json
- public\_registry.json

**Contracts:**

- https://hkuds.github.io/CLI-Anything/registry.json
- https://hkuds.github.io/CLI-Anything/public\_registry.json

**Claims:**

- arch-claim-hub

### Converted book artifact

**ID:** passport-converted-book

**Type:** entity

Converted book file and metadata, optionally reimported into the original Calibre library.

**Technology:** Calibre ebook-convert

**Status:** implemented

**Inputs:**

- Book ID
- Input/output formats
- Extra conversion arguments

**Outputs:**

- Output file path and size
- Add-to-library status

**Contracts:**

- Output file existence checked

**Claims:**

- passport-calibre-convert

### Blender render script artifact

**ID:** passport-render-script

**Type:** entity

Generated Python script and suggested Blender command; not a completed rendered image.

**Technology:** Python, Blender command line

**Status:** implemented

**Inputs:**

- JSON-style scene project
- Output/frame options

**Outputs:**

- \_render\_script.py
- Absolute paths and render metadata

**Contracts:**

- Artifact generation does not execute Blender

**Claims:**

- passport-blender-script

### Cloud generation task and optional download

**ID:** passport-cloud-task-result

**Type:** entity

AnyGen task identifier, URL, status and remote output, with conditional local file metadata.

**Technology:** Remote task API

**Status:** implemented

**Inputs:**

- Prompt
- Operation
- Optional output directory

**Outputs:**

- Task result
- Optional downloaded artifact

**Contracts:**

- Download limited to slide/doc/smart\_draw with output directory

**Claims:**

- passport-anygen-task

### Audio export and measured metadata

**ID:** passport-audio-artifact

**Type:** entity

Rendered WAV data with duration, size, peak and RMS; reported non-WAV format may differ from file encoding.

**Technology:** WAV audio

**Status:** implemented

**Inputs:**

- Project dictionary
- Output path
- Preset/settings

**Outputs:**

- WAV bytes
- Duration, file size, peak and RMS

**Contracts:**

- Non-WAV preset still writes WAV bytes

**Claims:**

- passport-audacity-format

### Application CLI harnesses

**ID:** arch-harnesses

**Type:** component

Independently installed application-specific CLI runtimes, not a shared application server.

**Technology:** Python; Click; JSON; REPL

**Status:** implemented

**Architecture role:** runtime\_unit

**Parent:** arch-product

**Inputs:**

- Application commands

**Outputs:**

- Structured results
- Application artifacts

**Contracts:**

- cli-anything-&lt;software&gt;
- --help
- --json
- Interactive REPL

**Claims:**

- arch-claim-harness-interface
- arch-claim-backend-diversity

### Application engines and services

**ID:** arch-app-backends

**Type:** external\_system

Separately provisioned application dependencies, represented by Draw.io desktop CLI, ChromaDB REST service and CloudAnalyzer Python engine. Harness behavior and compatibility vary by backend; there is no universal backend protocol.

**Technology:** Executable subprocesses; HTTP/JSON; Python library calls

**Status:** implemented

**Inputs:**

- Backend-specific arguments or requests

**Outputs:**

- Application results and artifacts

**Contracts:**

- Draw.io: --export INPUT --output OUTPUT --format FORMAT; subprocess success and output required
- ChromaDB: /api/v2; default localhost:8000; default\_tenant/default\_database
- CloudAnalyzer: ca.evaluate.evaluate

**Claims:**

- arch-claim-backend-diversity

## Connections

### arch-rel-repo

**From:** arch-repository

**To:** arch-product

**Label:** Source provenance

**Type:** association

**Interaction:** artifact

**Claims:**

- arch-claim-purpose

### arch-rel-agent-generator

**From:** arch-agent-hosts

**To:** arch-generator

**Label:** Loads and interprets generation specification

**Type:** dependency

**Interaction:** artifact

**Claims:**

- arch-claim-generator
- arch-claim-registration

### arch-rel-hub-catalog

**From:** arch-hub

**To:** arch-catalog-service

**Label:** Retrieves catalogs over HTTPS

**Type:** call

**Interaction:** sync

**Claims:**

- arch-claim-hub

### arch-rel-skills-hub

**From:** arch-discovery-skills

**To:** arch-hub

**Label:** Documents discovery/install commands for agent execution

**Type:** dependency

**Interaction:** artifact

**Claims:**

- arch-claim-skills

### arch-rel-generator-harness

**From:** arch-generator

**To:** arch-harnesses

**Label:** Agent workflow generates installable harness packages

**Type:** data

**Interaction:** artifact

**Claims:**

- arch-claim-generator
- arch-claim-harness-interface

### arch-rel-hub-harness

**From:** arch-hub

**To:** arch-harnesses

**Label:** Installs selected catalog harness

**Type:** dependency

**Interaction:** build

**Claims:**

- arch-claim-hub

### arch-rel-harness-backend

**From:** arch-harnesses

**To:** arch-app-backends

**Label:** Delegates using harness-specific subprocess, HTTP or library boundaries (not a uniform protocol)

**Type:** call

**Interaction:** unknown

**Claims:**

- arch-claim-backend-diversity

### arch-rel-generator-package

**From:** arch-generator

**To:** passport-harness-package

**Label:** Agent generation workflow produces package, tests and discovery artifacts

**Type:** data

**Interaction:** artifact

**Claims:**

- passport-generation
- arch-claim-generator

### arch-rel-hub-cache

**From:** arch-hub

**To:** passport-catalog-artifacts

**Label:** Retrieves and caches published catalog data

**Type:** data

**Interaction:** artifact

**Claims:**

- passport-catalog

### arch-rel-hub-installed-state

**From:** arch-hub

**To:** passport-installed-metadata

**Label:** Records package metadata following successful installation

**Type:** data

**Interaction:** artifact

**Claims:**

- passport-install

## Scenarios

### Generate and locally install an application harness

**ID:** passport-generate-harness

A coding agent follows the plugin task specification; prescribed test execution and installation are not observed execution results.

**Inputs:**

- Local application source path or GitHub repository URL

**Outputs:**

- Generated harness package
- Tests and discovery skills
- Locally installed console command

#### 1. Acquire and analyze application

**Step ID:** passport-generation-source

Resolve local source or clone a URL and analyze engine/data model.

**Objects:**

- arch-agent-hosts
- arch-generator

**Claims:**

- passport-generation

#### 2. Design and generate harness

**Step ID:** passport-generation-build

Design state/commands and produce Click CLI, REPL and structured JSON interface.

**Objects:**

- arch-generator
- passport-harness-package

**Claims:**

- passport-generation

#### 3. Validate and package

**Step ID:** passport-generation-deliver

Specification directs agent to create/run tests, generate skills and perform editable local installation.

**Objects:**

- arch-generator
- passport-harness-package
- arch-harnesses

**Claims:**

- passport-generation

### Resolve and install a catalog harness

**ID:** passport-install-harness

Hub resolves a catalog name and records installation only after success. Validation is based on inspected tests, not executed checks.

**Inputs:**

- Catalog package name
- Published or cached catalog

**Outputs:**

- Installed CLI
- Installed-state metadata

**Failure modes:**

- Unknown catalog name
- Subprocess installation failure

#### 1. Resolve catalog entry

**Step ID:** passport-install-resolve

Use HTTPS catalogs with separate one-hour local caches.

**Objects:**

- arch-hub
- arch-catalog-service
- passport-catalog-artifacts

**Claims:**

- passport-catalog

#### 2. Install and persist success

**Step ID:** passport-install-execute

Dispatch installation and save metadata only after successful completion.

**Objects:**

- arch-hub
- arch-harnesses
- passport-installed-metadata

**Claims:**

- passport-install
- passport-install-validation

### Convert a library book and optionally reimport

**ID:** passport-convert-book

Representative Calibre local backend workflow verifies an output file and optionally updates the same book.

**Inputs:**

- Book and requested formats
- Optional extra arguments
- Add-to-library choice

**Outputs:**

- Converted file
- Conversion metadata

**Failure modes:**

- No output artifact fails the existence check.

#### 1. Export, convert and check

**Step ID:** passport-convert-book-run

Export input to temporary storage, invoke ebook-convert and verify output exists.

**Objects:**

- arch-harnesses
- arch-app-backends
- passport-converted-book

**Claims:**

- passport-calibre-convert

#### 2. Optionally update library

**Step ID:** passport-convert-book-return

Optionally add converted format to the original book and return status and artifact metadata.

**Objects:**

- arch-harnesses
- arch-app-backends
- passport-converted-book

**Claims:**

- passport-calibre-convert

### Generate a Blender render script

**ID:** passport-author-render

Representative Blender harness authoring-to-export completion creates script artifacts, not a proven rendered image. Generated script alone is not evidence of successful rendering.

**Inputs:**

- Scene project
- Output/frame options

**Outputs:**

- Render script
- Suggested Blender invocation

#### 1. Generate export artifact

**Step ID:** passport-author-render-export

Produce _render_script.py beside requested output and return paths, metadata and background Blender command.

**Objects:**

- arch-harnesses
- passport-render-script

**Claims:**

- passport-blender-script

### Generate remote content and conditionally download

**ID:** passport-cloud-generation

Representative AnyGen harness workflow: task completion and local artifact materialization are separate results.

**Inputs:**

- Prompt and operation
- Optional output directory

**Outputs:**

- Remote task result
- Optional local file metadata

**Failure modes:**

- Without output directory, or outside slide/doc/smart\_draw, no download is performed.

#### 1. Create and poll task

**Step ID:** passport-cloud-generation-task

Create remote task and poll for completion.

**Objects:**

- arch-harnesses
- arch-app-backends
- passport-cloud-task-result

**Claims:**

- passport-anygen-task

#### 2. Conditionally download

**Step ID:** passport-cloud-generation-download

Download only supported operation outputs when a destination directory is supplied.

**Objects:**

- arch-harnesses
- passport-cloud-task-result

**Claims:**

- passport-anygen-task

### Render project audio to a local file

**ID:** passport-audio-export

Representative Audacity harness workflow renders clips and track effects, mixes active tracks, clamps samples and writes audio with measured metadata.

**Inputs:**

- Audio project
- Output path and preset

**Outputs:**

- WAV data
- Audio statistics

**Failure modes:**

- Non-WAV preset reporting can disagree with actual WAV encoding.

#### 1. Render and write audio

**Step ID:** passport-audio-export-render

Skip muted/non-solo tracks, render effects, mix volumes/pans, clamp and write WAV with statistics.

**Objects:**

- arch-harnesses
- passport-audio-artifact

**Claims:**

- passport-audacity-format

## Diagrams

### Runtime boundaries and external contracts

**ID:** architecture-part-2-diagram

**Type:** containers

Within CLI-Anything, the local CLI Hub retrieves published JSON catalogs over HTTPS and installs selected harnesses. Installation is a package-management relationship, not an application request path. Application CLI harnesses form a family of independently installed runtimes, not a shared service. They delegate to external application engines and services through harness-specific subprocess, HTTP or library contracts, not a uniform backend protocol.

**Question:** What are CLI-Anything&\#39;s runtime boundaries and their external contracts?

**Scope:** arch-product

**Nodes (see Objects):**

- arch-hub
- arch-harnesses
- arch-catalog-service
- arch-app-backends

**Connections (see Connections):**

- arch-rel-hub-catalog
- arch-rel-hub-harness
- arch-rel-harness-backend

## Claims

### passport-generation

Harness generation is an agent-directed Markdown workflow: acquire source, analyze application, design commands/state, generate Click/REPL/JSON harness, create and run tests, generate discovery skills, and package/install locally. The specification is not proof these actions were executed.

**Basis:** source\_backed

**Evidence:**

- passport-ev-generation

### passport-install

Hub installation resolves a catalog entry, dispatches installation, and persists installed metadata only after success.

**Basis:** source\_backed

**Evidence:**

- passport-ev-install

### passport-catalog

Hub consumes registry.json and public_registry.json over HTTPS from hkuds.github.io/CLI-Anything with separate local caches and one-hour TTL.

**Basis:** source\_backed

**Evidence:**

- passport-ev-catalog

### passport-install-validation

Inspected Hub tests cover installed-state JSON roundtrip, mocked successful pip installation, unknown names and subprocess failures; these tests were not executed during research.

**Basis:** source\_backed

**Evidence:**

- passport-ev-install-test

### arch-claim-purpose

CLI-Anything supports consuming existing application harnesses through a Hub and generating new harnesses through coding-agent plugins/skills.

**Basis:** source\_backed

**Evidence:**

- arch-ev-routes
- arch-ev-generator

### arch-claim-generator

The generator is an agent task specification accepting a local source path or GitHub URL and prescribing source analysis, Click/REPL/JSON harness generation, tests, skills and editable package installation; it is not a standalone generator server.

**Basis:** source\_backed

**Evidence:**

- arch-ev-generator

### arch-claim-hub

Python >=3.10 Hub exposes cli-hub via cli_hub.cli:main, uses Click and requests, retrieves HTTPS JSON catalogs with separate one-hour caches, and persists installed metadata after successful installation.

**Basis:** source\_backed

**Evidence:**

- arch-ev-hub-package
- arch-ev-hub-install
- arch-ev-catalog

### arch-claim-registration

Declarative Claude Code and Cursor marketplaces register cli-anything from ./cli-anything-plugin and ./cursor-plugin respectively; these registrations are not executing services and do not pin versions.

**Basis:** source\_backed

**Evidence:**

- arch-ev-claude
- arch-ev-cursor

### arch-claim-skills

Agent-readable discovery skills guide compatible agents to discover and install application CLIs through the Hub.

**Basis:** source\_backed

**Evidence:**

- arch-ev-meta
- arch-ev-routes

### passport-calibre-convert

Calibre conversion exports the requested book format to a temporary directory, invokes ebook-convert, checks output existence, optionally adds the result back to the library, and returns artifact metadata.

**Basis:** source\_backed

**Evidence:**

- passport-ev-calibre

### passport-blender-script

Blender export generates _render_script.py and returns script/output paths plus a Blender invocation; this does not itself execute Blender or prove a rendered image exists.

**Basis:** source\_backed

**Evidence:**

- passport-ev-blender

### passport-anygen-task

AnyGen creates and polls a remote task; downloads occur only with an output directory and slide/doc/smart_draw operation. Returned task state and remote output are distinct from optional local file metadata.

**Basis:** source\_backed

**Evidence:**

- passport-ev-anygen

### passport-audacity-format

Audacity export renders tracks/effects, mixes and clamps samples, writes WAV bytes and reports audio metadata; the non-WAV preset branch still writes WAV bytes despite reporting the requested format.

**Basis:** source\_backed

**Evidence:**

- passport-ev-audacity

### passport-calibre-runtime

The representative cli-anything-calibre 1.0.0 package exposes cli-anything-calibre through calibre_cli:main and requires Python >=3.10, Click >=8.0.0 and prompt-toolkit >=3.0.0. External Calibre tools must be installed on PATH; missing tools produce installation guidance. CALIBRE_OVERRIDE_LANG=en stabilizes output parsing, without demonstrated backend-version negotiation.

**Basis:** source\_backed

**Evidence:**

- passport-ev-calibre-package
- passport-ev-calibre-backend

### passport-plugin-smoke-check

Claude Code plugin installation documentation prescribes /reload-plugins and /help cli-anything as a manual load smoke check, with marketplace re-add/install recovery. This is a documented procedure, not evidence that the check was executed.

**Basis:** source\_backed

**Evidence:**

- passport-ev-plugin-smoke-check

### passport-skill-mirror-validation

Repository validation compares generated root skills against harness skill sources after canonical frontmatter rewriting, reports missing or stale mirrors and root-only edits, and exits nonzero with repair guidance on mismatch. Research inspected this validation source but did not execute it.

**Basis:** source\_backed

**Evidence:**

- passport-ev-skill-mirror-validation

### arch-claim-harness-interface

Generated harnesses are independently installed cli-anything-<software> commands with help, structured JSON and interactive REPL contracts; Python namespace packaging and skill metadata support agent consumption.

**Basis:** source\_backed

**Evidence:**

- arch-ev-harness-interface
- arch-ev-generator

### arch-claim-backend-diversity

Application harnesses use distinct backend strategies rather than one common application server: local export subprocesses (Draw.io), HTTP services (ChromaDB), in-process external libraries (CloudAnalyzer), direct SQLite/config access (CC Switch), and artifact-only script generation (Blender export).

**Basis:** source\_backed

**Evidence:**

- arch-ev-blender
- arch-ev-drawio
- arch-ev-chroma
- arch-ev-cloudanalyzer
- arch-ev-ccswitch

### arch-claim-variants

Host adapters remain within the generator subsystem: Pi's TypeScript extension injects HARNESS/specification context through sendUserMessage, while Codex offers an Experimental/Community skill installation preserving the generated Python harness format.

**Basis:** source\_backed

**Evidence:**

- arch-ev-pi
- arch-ev-codex

## Evidence

### passport-ev-generation

**Document ID:** RKTFADHEVLDVOZG22I3N7H7V3V

**File:** cli-anything-plugin/commands/cli-anything.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 9-85

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-anything-plugin/commands/cli-anything.md#L9-L85>)

### passport-ev-install

**Document ID:** RWG6OSPFKK7PHUYJPEMYAKVOC6

**File:** cli-hub/cli\_hub/installer.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 339-353

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/cli_hub/installer.py#L339-L353>)

### passport-ev-catalog

**Document ID:** FSYOQLCUTYHYFDOSCMVKUJL3HR

**File:** cli-hub/cli\_hub/registry.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 9-14

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/cli_hub/registry.py#L9-L14>)

### passport-ev-install-test

**Document ID:** 4ZAWIX4XBQGRV3ES73IVPSNZRP

**File:** cli-hub/tests/test\_cli\_hub.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 909-947

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/tests/test_cli_hub.py#L909-L947>)

### arch-ev-generator

**Document ID:** RKTFADHEVLDVOZG22I3N7H7V3V

**File:** cli-anything-plugin/commands/cli-anything.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 9-85

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-anything-plugin/commands/cli-anything.md#L9-L85>)

### arch-ev-routes

**Document ID:** 2HJKE3CAPGVYOS3RQTCQWTTGTK

**File:** README.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 181-245

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/README.md#L181-L245>)

### arch-ev-hub-package

**Document ID:** UWPW64TWA33IFYMJIH5GUVHUPV

**File:** cli-hub/setup.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 77-85

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/setup.py#L77-L85>)

### arch-ev-hub-install

**Document ID:** RWG6OSPFKK7PHUYJPEMYAKVOC6

**File:** cli-hub/cli\_hub/installer.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 339-353

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/cli_hub/installer.py#L339-L353>)

### arch-ev-catalog

**Document ID:** FSYOQLCUTYHYFDOSCMVKUJL3HR

**File:** cli-hub/cli\_hub/registry.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 9-14

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub/cli_hub/registry.py#L9-L14>)

### arch-ev-claude

**Document ID:** XKWEVP6CLXC3LGDQ6FXLOKBCQF

**File:** .claude-plugin/marketplace.json

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 1-20

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/.claude-plugin/marketplace.json#L1-L20>)

### arch-ev-cursor

**Document ID:** GLMDBEVGW2CXDJ5UDIDRSMJXBZ

**File:** .cursor-plugin/marketplace.json

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 1-25

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/.cursor-plugin/marketplace.json#L1-L25>)

### arch-ev-meta

**Document ID:** JRCMKERYQBDMNDKEC4VZOQE476

**File:** cli-hub-meta-skill/SKILL.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 1-29

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cli-hub-meta-skill/SKILL.md#L1-L29>)

### passport-ev-calibre

**Document ID:** 5QWEY7WFSEEDGHC63KF5PY6E5V

**File:** calibre/agent-harness/cli\_anything/calibre/core/formats.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 88-167

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/calibre/agent-harness/cli_anything/calibre/core/formats.py#L88-L167>)

### passport-ev-blender

**Document ID:** SGYJFFRCVPR45HZVXRYI5OY2BN

**File:** blender/agent-harness/cli\_anything/blender/core/render.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 182-239

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/blender/agent-harness/cli_anything/blender/core/render.py#L182-L239>)

### passport-ev-anygen

**Document ID:** 2BHRFAA4KRJ5DEWZPFAPTFEULE

**File:** anygen/agent-harness/cli\_anything/anygen/utils/anygen\_backend.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 414-442

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/anygen/agent-harness/cli_anything/anygen/utils/anygen_backend.py#L414-L442>)

### passport-ev-audacity

**Document ID:** 5DBNYHMKJFQ6TQ6XPZYNB6R6FA

**File:** audacity/agent-harness/cli\_anything/audacity/core/export.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 122-231

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/audacity/agent-harness/cli_anything/audacity/core/export.py#L122-L231>)

### passport-ev-calibre-package

**Document ID:** TQAN6TPUWL5JZ3IA2ZWXNMWGC2

**File:** calibre/agent-harness/setup.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 10-28

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/calibre/agent-harness/setup.py#L10-L28>)

### passport-ev-calibre-backend

**Document ID:** OV4XI7X3LZRH73BTB5L2AIR2BJ

**File:** calibre/agent-harness/cli\_anything/calibre/utils/calibre\_backend.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 16-58

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/calibre/agent-harness/cli_anything/calibre/utils/calibre_backend.py#L16-L58>)

### passport-ev-plugin-smoke-check

**Document ID:** 2HJKE3CAPGVYOS3RQTCQWTTGTK

**File:** README.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 294-307

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/README.md#L294-L307>)

### passport-ev-skill-mirror-validation

**Document ID:** XWIYKHPWG2YKE2G47EV6YQMJL2

**File:** .github/scripts/validate\_root\_skills.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 37-110

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/.github/scripts/validate_root_skills.py#L37-L110>)

### arch-ev-harness-interface

**Document ID:** 2HJKE3CAPGVYOS3RQTCQWTTGTK

**File:** README.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 763-780

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/README.md#L763-L780>)

### arch-ev-blender

**Document ID:** SGYJFFRCVPR45HZVXRYI5OY2BN

**File:** blender/agent-harness/cli\_anything/blender/core/render.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 182-239

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/blender/agent-harness/cli_anything/blender/core/render.py#L182-L239>)

### arch-ev-drawio

**Document ID:** AQ2KIUXEGOPCN74QDZ6A2G7XXY

**File:** drawio/agent-harness/cli\_anything/drawio/utils/drawio\_backend.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 90-145

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/drawio/agent-harness/cli_anything/drawio/utils/drawio_backend.py#L90-L145>)

### arch-ev-chroma

**Document ID:** 2NABQOCEJPNTNU5JTLDV44FP6G

**File:** chromadb/agent-harness/cli\_anything/chromadb/utils/chromadb\_backend.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 15-40

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/chromadb/agent-harness/cli_anything/chromadb/utils/chromadb_backend.py#L15-L40>)

### arch-ev-cloudanalyzer

**Document ID:** CVZD3HODRTTAOEITJ2WJRRGGYW

**File:** cloudanalyzer/agent-harness/cli\_anything/cloudanalyzer/utils/ca\_backend.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 19-46

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cloudanalyzer/agent-harness/cli_anything/cloudanalyzer/utils/ca_backend.py#L19-L46>)

### arch-ev-ccswitch

**Document ID:** SYVAQE5JS7KL7VR5F2ECMXPSZ6

**File:** cc-switch/agent-harness/cli\_anything/ccswitch/utils/db.py

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 10-38

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/cc-switch/agent-harness/cli_anything/ccswitch/utils/db.py#L10-L38>)

### arch-ev-pi

**Document ID:** RUTVITSFUSNAVSOPHWJ3MWQR2M

**File:** .pi-extension/cli-anything/index.ts

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 18-185

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/.pi-extension/cli-anything/index.ts#L18-L185>)

### arch-ev-codex

**Document ID:** 2HJKE3CAPGVYOS3RQTCQWTTGTK

**File:** README.md

**Repository or context:** https://github.com/HKUDS/CLI-Anything

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Lines:** 593-634

[View source](<https://github.com/HKUDS/CLI-Anything/blob/34f519533bc175d2fe287ab8316b0dd99bb9cc43/README.md#L593-L634>)

## Open questions

**Unresolved:**

- The prescribed generator workflow and inspected validation/test sources do not establish successful execution for arbitrary applications or successful rendering of generated Blender scripts; no new execution verification was performed during synthesis.

## Source snapshots

### https://github.com/HKUDS/CLI-Anything

**Snapshot ID:** 5CL6AHMQ6SYTCKYQKO7GDMI2LF

**Commit:** 34f519533bc175d2fe287ab8316b0dd99bb9cc43

**Coverage note:** Repository text indexed at the pinned SHA. Build outputs, dependencies, binary files and secret-like files are excluded. Available files are not automatically read by the model.

**Indexed files:** 1728

**Files read:** 509

**Files cited:** 42


