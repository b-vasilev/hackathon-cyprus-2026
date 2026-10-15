import type { FoundationModule } from '../types';

export const ciCd: FoundationModule = {
  id: 'ci-cd',
  title: 'CI/CD and GitHub Actions',
  summary: 'What a pipeline does, how a workflow file is put together, and where it runs.',
  outcome: 'Read a workflow file, figure out why a check failed, and fix it.',
  minutes: 30,
  lessons: [
    {
      id: 'what-is-ci',
      title: 'What CI and CD mean',
      summary: 'The terms, and what a typical pipeline runs.',
      blocks: [
        {
          type: 'p',
          text: "Continuous integration (CI) means people merge small changes often, and every push or PR gets built and checked automatically. When something breaks, you hear about it minutes after the change, while it's still fresh in your head.",
        },
        {
          type: 'table',
          columns: ['Term', 'Meaning'],
          rows: [
            ['Continuous integration', 'Each change is built, linted and tested automatically.'],
            [
              'Continuous delivery',
              'Each change that passes could be released. A person decides when.',
            ],
            ['Continuous deployment', 'Each change that passes goes to production automatically.'],
          ],
        },
        {
          type: 'p',
          text: 'Pipelines usually run the quick checks first, so a typo fails in 30 seconds and not after a 10-minute test run:',
        },
        {
          type: 'list',
          ordered: true,
          items: [
            'Install dependencies from the lockfile',
            'Formatting and lint',
            'Type check',
            'Tests',
            'Build (bundle, binary, Docker image)',
            "Deploy, if it's a delivery pipeline",
          ],
        },
        {
          type: 'p',
          text: 'On GitHub, every result shows up as a check on the PR. Branch protection can make checks required, so the merge button stays disabled until they pass.',
        },
      ],
    },
    {
      id: 'actions-anatomy',
      title: 'Reading a GitHub Actions workflow',
      summary: 'Events, jobs, steps, and how they fit together.',
      blocks: [
        {
          type: 'p',
          text: "Workflows are YAML files in `.github/workflows/`. Here's a small but realistic one:",
        },
        {
          type: 'code',
          code: 'name: CI\n\non:\n  pull_request:\n  push:\n    branches: [main]\n\npermissions:\n  contents: read\n\njobs:\n  check:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n      - uses: actions/setup-node@v5\n        with:\n          node-version: 24\n          cache: npm\n      - run: npm ci\n      - run: npm run lint\n      - run: npm test\n\n  build:\n    needs: check\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n      - run: npm ci && npm run build',
          caption: 'Runs on every PR and every push to `main`. `build` waits for `check` to pass.',
        },
        {
          type: 'table',
          columns: ['Piece', 'What it is'],
          rows: [
            [
              '`on:`',
              'What starts the workflow: `push`, `pull_request`, `schedule`, `workflow_dispatch` (a manual "Run" button), and others.',
            ],
            [
              'Job',
              'A set of steps on one machine. Jobs run in parallel unless one `needs` another.',
            ],
            [
              'Step',
              "Either a shell command (`run:`) or a ready-made action (`uses:`). Steps run in order and share the job's files.",
            ],
            ['Action', 'A reusable step someone published, like `actions/checkout`.'],
            [
              'Matrix',
              'Runs the same job several times with different values, say Node 20, 22 and 24.',
            ],
            ['Artifact', 'Files a job uploads so a later job, or a person, can download them.'],
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: "Jobs don't share a disk",
          text: "Each job runs on its own machine. A `dist/` folder built in one job won't exist in the next. Pass it with `actions/upload-artifact` and `actions/download-artifact`, or do both steps in the same job.",
        },
      ],
    },
    {
      id: 'runners',
      title: 'Runners',
      summary: 'The machines that run your jobs, and how to choose one.',
      blocks: [
        {
          type: 'p',
          text: 'A runner is the machine a job runs on. You pick it with `runs-on`, using a label such as `ubuntu-latest`, `windows-latest`, `macos-latest`, or `[self-hosted, linux]` for your own machines.',
        },
        {
          type: 'table',
          columns: ['Type', 'Who looks after it', 'Between jobs', 'Use it for'],
          rows: [
            [
              'GitHub-hosted',
              'GitHub',
              'Brand new VM for every job',
              'Most things. No maintenance.',
            ],
            [
              'Larger runners',
              'GitHub (paid)',
              'Brand new, with more CPU and RAM',
              'Heavy builds, GPUs, fixed IP ranges.',
            ],
            [
              'Self-hosted',
              'Your team',
              'Keeps files and state unless you set it up not to',
              'Private network access, special hardware.',
            ],
          ],
        },
        {
          type: 'p',
          text: "Because a hosted runner starts empty, every job installs what it needs. Caching (like `cache: npm` above) keeps that fast. If something works on a self-hosted runner and fails on a hosted one, it's usually relying on a tool someone installed by hand.",
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Self-hosted runners and public repos',
          text: "Anyone can open a PR on a public repo, and workflows can run the code in that PR. On a self-hosted runner, that means a stranger's code running on your machine, on your network. GitHub recommends self-hosted runners for private repos only.",
        },
      ],
    },
    {
      id: 'secrets-and-safety',
      title: 'Secrets and permissions',
      summary: 'Give workflows only the access they need.',
      blocks: [
        {
          type: 'p',
          // biome-ignore lint/suspicious/noTemplateCurlyInString: GitHub Actions expression syntax, not a JS template
          text: "Store credentials as secrets (per repo, environment or organization) and read them with `${{ secrets.NAME }}`. GitHub hides secret values in logs, but only exact matches. If a script base64-encodes a secret and prints it, it's out.",
        },
        {
          type: 'list',
          items: [
            'Each workflow gets a `GITHUB_TOKEN` automatically. Limit it with a `permissions:` block. Most CI jobs only need `contents: read`.',
            "PRs from forks don't get your secrets (apart from a read-only `GITHUB_TOKEN`). That's on purpose.",
            '`pull_request_target` runs with secrets, in the context of your repo. Never check out and run PR code in it.',
            'Use environments with required reviewers for deployment credentials.',
            'Pin third-party actions to a version tag, or to a full commit SHA if you want to be strict.',
          ],
        },
      ],
    },
    {
      id: 'failing-checks',
      title: 'When a check goes red',
      summary: 'A routine for working out what broke.',
      blocks: [
        {
          type: 'list',
          ordered: true,
          items: [
            'Open the failed check and find the first step that failed. Everything after it is often fallout.',
            'Read the actual error. Search the log for "error" or the name of the failing test.',
            'Run the same command locally, for example `npm ci && npm test`.',
            "Unrelated to your change and passes on re-run? That's a flaky test. Report it. Don't keep clicking re-run until it's green.",
            'Push the fix to the same branch. Checks re-run on their own.',
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: "Don't go around the pipeline",
          text: 'Skipping tests, disabling the check or merging with admin rights just moves the problem to whoever comes next. Fix it, or agree an exception with whoever owns the check.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'ci-fill-workflow',
      kind: 'fill',
      code: true,
      prompt:
        "Complete the workflow: run on pull requests, use a GitHub-hosted Linux machine, install exactly what's in the lockfile, and build only after `check` passes.",
      template:
        'on:\n  {{0}}:\n\njobs:\n  check:\n    runs-on: {{1}}\n    steps:\n      - uses: actions/checkout@v5\n      - run: {{2}}\n      - run: npm test\n  build:\n    {{3}}: check\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n      - run: npm ci && npm run build',
      answers: ['pull_request', 'ubuntu-latest', 'npm ci', 'needs'],
      distractors: ['self-hosted', 'npm install', 'after', 'pull'],
      explanation:
        '`pull_request` is the event name, `ubuntu-latest` is a hosted runner label, `npm ci` installs from the lockfile without changing it, and `needs` makes `build` wait for `check`.',
    },
    {
      id: 'ci-pipeline-order',
      kind: 'fill',
      prompt: 'Put a typical pipeline in order, cheapest checks first.',
      template: '1. {{0}}\n2. {{1}}\n3. {{2}}\n4. {{3}}',
      answers: ['Install dependencies', 'Lint and format check', 'Run tests', 'Build'],
      distractors: [],
      explanation:
        'Nothing runs without dependencies. Then fail fast on cheap checks before spending minutes on tests, and only build what passed.',
    },
    {
      id: 'ci-red-check',
      kind: 'branch',
      prompt: 'Work through it. Each decision changes what happens next.',
      steps: [
        {
          id: 'start',
          text: "Thursday, 17:40. Your PR has to go out tomorrow morning and the `test` check just went red: `checkout.spec.ts › applies discount` timed out. You didn't touch checkout.",
          choices: [
            {
              label: 'Open the log and find the first failing step',
              good: true,
              outcome:
                'Install and lint passed. Only that one test failed, after waiting 5 seconds for a call to the payment sandbox.',
              next: 'local',
            },
            {
              label: 'Click re-run until it goes green',
              good: false,
              outcome:
                "It passes on the third try. You still don't know why it failed, and it'll do the same to the next person.",
              next: 'local',
            },
            {
              label: 'Ask an admin to merge it anyway',
              good: false,
              outcome: "They say no and ask what's failing. You have to look anyway.",
              next: 'local',
            },
          ],
        },
        {
          id: 'local',
          text: 'You run `npm test -- checkout` locally. It passes 9 times out of 10. On `main`, the same test failed twice this week.',
          choices: [
            {
              label: "Open an issue with links to the failed runs and tell the test's owner",
              good: true,
              outcome:
                'The owner already suspected the sandbox timeouts and quarantines the test. Your next run is green.',
              next: 'merge',
            },
            {
              label: 'Delete the test so your PR passes',
              good: false,
              outcome:
                'The reviewer notices a deleted test in a PR about something else and asks you to put it back.',
              next: 'merge',
            },
            {
              label: 'Add `sleep(10000)` before the assertion',
              good: false,
              outcome:
                'Now the suite is 10 seconds slower and still flaky. The sandbox sometimes takes 12.',
              next: 'merge',
            },
          ],
        },
        {
          id: 'merge',
          text: "Your PR is green and approved. It's 18:30.",
          choices: [
            {
              label: 'Merge, and link the flaky-test issue in the PR',
              good: true,
              outcome:
                "Anyone looking back at this week's red builds finds the explanation in one click.",
            },
            {
              label: "Merge and don't mention it",
              good: false,
              outcome: 'Next week someone else spends an hour rediscovering the same flaky test.',
            },
          ],
        },
      ],
    },
    {
      id: 'ci-jobs-share',
      kind: 'single',
      scenario:
        'Job `build` creates `dist/`. Job `deploy` has `needs: build`, and fails with "dist: no such file or directory".',
      prompt: 'Why?',
      options: [
        '`needs` only works on self-hosted runners.',
        "Each job runs on its own machine, so `dist/` isn't there. Upload it as an artifact and download it in `deploy`.",
        'The jobs ran in the wrong order.',
        'GitHub deletes `dist/` folders for security reasons.',
      ],
      correct: [1],
      explanation: "`needs` sets the order but doesn't move files. Artifacts do.",
    },
    {
      id: 'ci-self-hosted',
      kind: 'single',
      scenario:
        'An open-source project wants faster builds, so someone suggests running all PR checks on a spare server in the office.',
      prompt: "What's the problem?",
      options: [
        "Self-hosted runners can't run Linux.",
        "Anyone can open a PR, so strangers' code would run on a machine inside the office network.",
        "Self-hosted runners don't support npm.",
        "There isn't one.",
      ],
      correct: [1],
      explanation:
        "On a public repo you don't control who triggers workflows. Keep self-hosted runners for private repos.",
    },
    {
      id: 'ci-secrets',
      kind: 'multiple',
      prompt: 'Which of these are true about secrets in GitHub Actions?',
      options: [
        'Secret values are masked in logs.',
        "Workflows triggered by PRs from forks don't get repository secrets.",
        'A `permissions:` block can limit what `GITHUB_TOKEN` can do.',
        "Masking means a script can't leak a secret.",
      ],
      correct: [0, 1, 2],
      explanation:
        'Masking only catches the exact value. Encode or split it and it shows up in the log just fine.',
    },
  ],
};
