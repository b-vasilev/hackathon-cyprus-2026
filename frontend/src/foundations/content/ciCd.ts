import type { FoundationModule } from '../types';

const NPM_BUILD = '`npm run build`';
const CREATES_DIST = 'creates `dist/`';
const READS_DIST = 'reads `dist/`';
const MACHINE_A = 'Machine A: job `build`';
const MACHINE_B = 'Machine B: job `deploy`';
const REPO_SECRETS = 'Your repository secrets';
const WORKFLOW_RUN = 'Workflow run';

export const ciCd: FoundationModule = {
  id: 'ci-cd',
  title: 'CI/CD and GitHub Actions',
  summary: 'What a pipeline does, how a workflow file is built, and where it runs.',
  outcome: 'Read a workflow file, find out why a check failed, and fix it.',
  minutes: 30,
  lessons: [
    {
      id: 'what-is-ci',
      title: 'What CI and CD mean',
      summary: 'What CI and CD mean, and the steps of a typical pipeline.',
      blocks: [
        {
          type: 'p',
          text: 'Continuous integration (CI) means that people merge small changes often. Every push or pull request (PR) is built and checked automatically. When something breaks, you find out a few minutes after the change. You still remember what you changed, so the fix is easier.',
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
          text: 'A pipeline is the list of automatic steps that runs for each change. Pipelines usually run the quick checks first. A typo then fails in 30 seconds, and you do not wait for a 10-minute test run. A typical order is below. (A linter is a tool that reads your code and warns about likely mistakes.)',
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
            'Deploy, if it is a delivery pipeline',
          ],
        },
        {
          type: 'p',
          text: 'The pipeline stops at the first stage that fails. The stages after it do not run.',
        },
        {
          type: 'scenes',
          title: 'What happens when a stage fails',
          legend: [
            { tone: 'brand', label: 'Passed' },
            { tone: 'danger', label: 'Failed' },
            { tone: 'muted', label: 'Skipped' },
          ],
          scenes: [
            {
              name: 'All green',
              summary: 'Every stage passes, so the PR can be merged.',
              rows: [
                {
                  label: 'Pipeline',
                  blocks: [
                    { text: 'Install', tone: 'brand' },
                    { text: 'Lint', tone: 'brand' },
                    { text: 'Test', tone: 'brand' },
                    { text: 'Build', tone: 'brand' },
                  ],
                },
              ],
            },
            {
              name: 'Lint fails',
              summary: 'The pipeline stops at the lint stage.',
              description: 'Tests and build do not run. Fix the lint error first.',
              rows: [
                {
                  label: 'Pipeline',
                  blocks: [
                    { text: 'Install', tone: 'brand' },
                    { text: 'Lint', tone: 'danger', note: 'unused variable' },
                    { text: 'Test', tone: 'muted', note: 'skipped' },
                    { text: 'Build', tone: 'muted', note: 'skipped' },
                  ],
                },
              ],
            },
            {
              name: 'Test fails',
              summary: 'Lint passes, but one test fails, so the build does not run.',
              description: 'The PR cannot be merged until the test passes.',
              rows: [
                {
                  label: 'Pipeline',
                  blocks: [
                    { text: 'Install', tone: 'brand' },
                    { text: 'Lint', tone: 'brand' },
                    { text: 'Test', tone: 'danger', note: '1 test failed' },
                    { text: 'Build', tone: 'muted', note: 'skipped' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'p',
          text: 'On GitHub, every result is shown as a check on the PR. Branch protection can make checks required. The merge button then stays disabled until the checks pass.',
        },
      ],
    },
    {
      id: 'actions-anatomy',
      title: 'Reading a GitHub Actions workflow',
      summary: 'How events, jobs and steps fit together in a workflow file.',
      blocks: [
        {
          type: 'p',
          text: 'A workflow is a YAML file in `.github/workflows/`. It tells GitHub what to run and when. Here is a small but realistic one. Click a part to see what it does.',
        },
        {
          type: 'anatomy',
          title: '`.github/workflows/ci.yml`',
          segments: [
            { text: 'name: CI', part: 'name' },
            { text: '\n\n' },
            { text: 'on:\n  pull_request:\n  push:\n    branches: [main]', part: 'on' },
            { text: '\n\n' },
            { text: 'permissions:\n  contents: read', part: 'permissions' },
            { text: '\n\n' },
            { text: 'jobs:\n  check:', part: 'job' },
            { text: '\n    ' },
            { text: 'runs-on: ubuntu-latest', part: 'runs-on' },
            { text: '\n    ' },
            { text: 'steps:', part: 'steps' },
            { text: '\n      - ' },
            { text: 'uses: actions/checkout@v5', part: 'uses' },
            {
              text: '\n      - uses: actions/setup-node@v5\n        with:\n          node-version: 24\n          cache: npm\n      - ',
            },
            { text: 'run: npm ci', part: 'run' },
            { text: '\n      - run: npm run lint\n      - run: npm test\n\n  build:\n    ' },
            { text: 'needs: check', part: 'needs' },
            {
              text: '\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n      - run: npm ci && npm run build',
            },
          ],
          parts: [
            {
              id: 'name',
              label: 'Workflow name',
              explanation:
                'The name GitHub shows in the Actions tab and next to the checks on a PR.',
            },
            {
              id: 'on',
              label: 'Events',
              explanation:
                'What starts the workflow. Here it runs on every PR and on every push to `main`. Other events are `schedule` and `workflow_dispatch` (a manual "Run" button).',
            },
            {
              id: 'permissions',
              label: 'Permissions',
              explanation:
                'What the automatic `GITHUB_TOKEN` may do. With `contents: read`, the jobs can read the code but cannot change the repo.',
            },
            {
              id: 'job',
              label: 'Job',
              explanation:
                '`jobs:` lists the jobs, and `check` is a job id that you choose. A job is a set of steps on one machine. Jobs run in parallel unless one `needs` another.',
            },
            {
              id: 'runs-on',
              label: 'Runner',
              explanation:
                'The machine the job runs on. `ubuntu-latest` is a Linux machine that GitHub hosts. Each job gets its own machine.',
            },
            {
              id: 'steps',
              label: 'Steps',
              explanation:
                "The steps of the job. They run in order on the same machine, so they share the job's files.",
            },
            {
              id: 'uses',
              label: 'Action step',
              explanation:
                '`uses:` runs an action, a reusable step that someone published. `actions/checkout` copies your repo onto the machine. `@v5` is the version.',
            },
            {
              id: 'run',
              label: 'Command step',
              explanation:
                '`run:` runs a shell command, like in your terminal. `npm ci` installs exactly what is in the lockfile.',
            },
            {
              id: 'needs',
              label: 'Needs',
              explanation:
                '`build` waits for `check` to pass. If `check` fails, `build` does not run. `needs` only sets the order. It does not move files.',
            },
          ],
        },
        {
          type: 'p',
          text: 'Two more terms you will see in workflow files:',
        },
        {
          type: 'table',
          columns: ['Term', 'What it is'],
          rows: [
            [
              'Matrix',
              'Runs the same job several times with different values, for example Node 20, 22 and 24.',
            ],
            ['Artifact', 'Files a job uploads so a later job, or a person, can download them.'],
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Jobs do not share a disk',
          text: 'Each job runs on its own machine. A `dist/` folder built in one job does not exist in the next job. Pass it with `actions/upload-artifact` and `actions/download-artifact`, or do both steps in the same job.',
        },
        {
          type: 'boundary',
          title: 'Where the `dist/` folder exists',
          legend: [
            { tone: 'neutral', label: 'Step' },
            { tone: 'brand', label: 'Has `dist/`' },
            { tone: 'soft', label: 'Copied as artifact' },
            { tone: 'danger', label: 'Failed' },
          ],
          scenes: [
            {
              name: 'Steps in one job',
              summary: 'All steps run on one machine, so later steps see `dist/`.',
              description: 'Steps in the same job share files.',
              zones: [
                {
                  label: MACHINE_A,
                  items: [
                    { text: 'Checkout' },
                    { text: NPM_BUILD, tone: 'brand', note: CREATES_DIST },
                    { text: 'Deploy', tone: 'brand', note: READS_DIST },
                  ],
                },
              ],
            },
            {
              name: 'Two jobs',
              summary: 'The deploy job runs on a new machine without `dist/`.',
              description: '`needs` only sets the order. It does not move files.',
              zones: [
                {
                  label: MACHINE_A,
                  items: [
                    { text: 'Checkout' },
                    { text: NPM_BUILD, tone: 'brand', note: CREATES_DIST },
                  ],
                },
                {
                  label: MACHINE_B,
                  items: [
                    { text: 'Checkout' },
                    { text: 'Deploy', tone: 'danger', note: '`dist/` missing' },
                  ],
                },
              ],
              links: [{ from: 0, label: '`dist/` does not move', ok: false }],
            },
            {
              name: 'Two jobs + artifact',
              summary: 'The build job uploads `dist/`, and the deploy job downloads it.',
              description: 'Artifacts are the way to pass files between jobs.',
              zones: [
                {
                  label: MACHINE_A,
                  items: [
                    { text: NPM_BUILD, tone: 'brand', note: CREATES_DIST },
                    { text: 'upload-artifact', tone: 'soft', note: 'stores `dist/`' },
                  ],
                },
                {
                  label: MACHINE_B,
                  items: [
                    { text: 'download-artifact', tone: 'soft', note: 'gets `dist/`' },
                    { text: 'Deploy', tone: 'brand', note: READS_DIST },
                  ],
                },
              ],
              links: [{ from: 0, label: 'upload / download artifact', ok: true }],
            },
          ],
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
          text: 'A runner is the machine a job runs on. You choose it with `runs-on`. Use a label such as `ubuntu-latest`, `windows-latest`, `macos-latest`, or `[self-hosted, linux]` for your own machines.',
        },
        {
          type: 'table',
          columns: ['Type', 'Who maintains it', 'Between jobs', 'Use it for'],
          rows: [
            [
              'GitHub-hosted',
              'GitHub',
              'A new virtual machine (VM) for every job',
              'Most work. No maintenance.',
            ],
            [
              'Larger runners',
              'GitHub (paid)',
              'A new VM, with more CPU and RAM',
              'Heavy builds, GPUs, fixed IP ranges.',
            ],
            [
              'Self-hosted',
              'Your team',
              'Keeps files and state unless you set it up not to',
              'Access to a private network, special hardware.',
            ],
          ],
        },
        {
          type: 'p',
          text: 'A hosted runner starts empty, so every job installs what it needs. Caching (like `cache: npm` above) keeps this fast. Sometimes a job works on a self-hosted runner and fails on a hosted one. The usual reason is that it needs a tool that someone installed by hand on the self-hosted machine.',
        },
        {
          type: 'boundary',
          title: 'What the next job finds on the machine',
          legend: [
            { tone: 'brand', label: 'Job work' },
            { tone: 'warning', label: 'Left from earlier' },
            { tone: 'muted', label: 'Removed' },
          ],
          scenes: [
            {
              name: 'GitHub-hosted',
              summary: 'Each job gets a new, empty VM.',
              description: 'Nothing from job 1 is left for job 2. The VM is removed after the job.',
              zones: [
                {
                  label: 'Job 1: new VM',
                  tone: 'soft',
                  items: [
                    { text: 'Install + build', tone: 'brand' },
                    { text: 'VM removed', tone: 'muted' },
                  ],
                },
                {
                  label: 'Job 2: another new, empty VM',
                  tone: 'soft',
                  items: [
                    { text: 'Install + test', tone: 'brand' },
                    { text: 'VM removed', tone: 'muted' },
                  ],
                },
              ],
              links: [{ from: 0, label: 'nothing carried over', ok: false }],
            },
            {
              name: 'Self-hosted',
              summary: 'Both jobs run on the same machine, one after the other.',
              description:
                'Job 2 can see files and tools from job 1. This can hide missing setup steps.',
              zones: [
                {
                  label: 'Office server: job 1, then job 2',
                  items: [
                    { text: 'Job 1: install + build', tone: 'brand', note: 'leaves files' },
                    { text: 'Old files', tone: 'warning', note: 'from job 1' },
                    { text: 'Job 2: install + test', tone: 'brand', note: 'sees the old files' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Self-hosted runners and public repos',
          text: "Anyone can open a PR on a public repo, and workflows can run the code in that PR. On a self-hosted runner, a stranger's code then runs on your machine, inside your network. GitHub recommends self-hosted runners for private repos only.",
        },
      ],
    },
    {
      id: 'secrets-and-safety',
      title: 'Secrets and permissions',
      summary: 'How to give workflows only the access they need.',
      blocks: [
        {
          type: 'p',
          // biome-ignore lint/suspicious/noTemplateCurlyInString: GitHub Actions expression syntax, not a JS template
          text: 'A secret is a stored value such as a password or an API key. Store credentials as secrets (per repo, environment or organization) and read them with `${{ secrets.NAME }}`. GitHub hides secret values in logs, but only exact matches. If a script base64-encodes a secret and prints it, the secret is visible in the log.',
        },
        {
          type: 'list',
          items: [
            'Each workflow gets a `GITHUB_TOKEN` automatically. Limit it with a `permissions:` block. Most CI jobs only need `contents: read`.',
            'PRs from forks do not get your secrets (apart from a read-only `GITHUB_TOKEN`). This is on purpose.',
            '`pull_request_target` runs with secrets, in the context of your repo. Never check out and run PR code in it.',
            'Use environments with required reviewers for deployment credentials.',
            'Pin third-party actions to a version tag. To be strict, pin them to a full commit SHA.',
          ],
        },
        {
          type: 'p',
          text: 'A fork is a copy of a repo under another account. Anyone can fork a public repo and open a PR from the fork.',
        },
        {
          type: 'boundary',
          title: 'Which secrets a PR workflow gets',
          legend: [
            { tone: 'neutral', label: 'Trigger' },
            { tone: 'brand', label: 'Available' },
            { tone: 'warning', label: 'Limited' },
            { tone: 'danger', label: 'Not available' },
          ],
          scenes: [
            {
              name: 'PR from your repo',
              summary: 'The workflow gets repo secrets and a normal `GITHUB_TOKEN`.',
              description: 'The `permissions:` block still limits what the token can do.',
              zones: [
                {
                  label: REPO_SECRETS,
                  tone: 'brand',
                  items: [{ text: 'API keys' }, { text: 'Passwords' }],
                },
                {
                  label: WORKFLOW_RUN,
                  items: [
                    { text: 'PR from a branch', tone: 'neutral' },
                    { text: 'Repo secrets', tone: 'brand' },
                    { text: '`GITHUB_TOKEN`', tone: 'brand', note: 'as set in `permissions:`' },
                  ],
                },
              ],
              links: [{ from: 0, label: 'secrets passed', ok: true }],
            },
            {
              name: 'PR from a fork',
              summary: 'The workflow gets no repo secrets and only a read-only token.',
              description: 'This stops code from strangers from reading your credentials.',
              zones: [
                {
                  label: REPO_SECRETS,
                  tone: 'brand',
                  items: [{ text: 'API keys' }, { text: 'Passwords' }],
                },
                {
                  label: WORKFLOW_RUN,
                  items: [
                    { text: 'PR from a fork', tone: 'neutral' },
                    { text: 'Repo secrets', tone: 'danger', note: 'not given' },
                    { text: '`GITHUB_TOKEN`', tone: 'warning', note: 'read-only' },
                  ],
                },
              ],
              links: [{ from: 0, label: 'no repo secrets', ok: false }],
            },
          ],
        },
      ],
    },
    {
      id: 'failing-checks',
      title: 'When a check fails',
      summary: 'A routine to find out what broke.',
      blocks: [
        {
          type: 'list',
          ordered: true,
          items: [
            'Open the failed check and find the first step that failed. Errors after it are often caused by that first failure.',
            'Read the actual error. Search the log for "error" or for the name of the failing test.',
            'Run the same command on your laptop, for example `npm ci && npm test`.',
            'Is the failure unrelated to your change, and does it pass when you run it again? Then it is a flaky test, a test that sometimes passes and sometimes fails without a code change. Report it. Do not click re-run again and again until it passes.',
            'Push the fix to the same branch. The checks run again automatically.',
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Do not go around the pipeline',
          text: 'Skipping tests, turning off the check or merging with admin rights only moves the problem to the next person. Fix it, or agree an exception with the owner of the check.',
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
        'Complete the workflow. It must run on pull requests, use a GitHub-hosted Linux machine, install exactly what is in the lockfile, and build only after `check` passes.',
      template:
        'on:\n  {{0}}:\n\njobs:\n  check:\n    runs-on: {{1}}\n    steps:\n      - uses: actions/checkout@v5\n      - run: {{2}}\n      - run: npm test\n  build:\n    {{3}}: check\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v5\n      - run: npm ci && npm run build',
      answers: ['pull_request', 'ubuntu-latest', 'npm ci', 'needs'],
      distractors: ['self-hosted', 'npm install', 'after', 'pull'],
      explanation:
        '`pull_request` is the event name. `ubuntu-latest` is a hosted runner label. `npm ci` installs from the lockfile without changing it. `needs` makes `build` wait for `check`.',
    },
    {
      id: 'ci-pipeline-order',
      kind: 'fill',
      prompt: 'Put a typical pipeline in order, with the cheapest checks first.',
      template: '1. {{0}}\n2. {{1}}\n3. {{2}}\n4. {{3}}',
      answers: ['Install dependencies', 'Lint and format check', 'Run tests', 'Build'],
      distractors: [],
      explanation:
        'Nothing runs without dependencies. Then run the cheap checks, so they fail fast before minutes of tests. Build only what passed.',
    },
    {
      id: 'ci-red-check',
      kind: 'branch',
      prompt: 'Work through the situation. Each decision changes what happens next.',
      steps: [
        {
          id: 'start',
          text: 'Thursday, 17:40. Your PR must go out tomorrow morning. The `test` check just failed: `checkout.spec.ts › applies discount` timed out. You did not change the checkout code.',
          choices: [
            {
              label: 'Open the log and find the first failing step',
              good: true,
              outcome:
                'Install and lint passed. Only that one test failed. It waited 5 seconds for a call to the payment sandbox.',
              next: 'local',
            },
            {
              label: 'Click re-run until it passes',
              good: false,
              outcome:
                'It passes on the third try. You still do not know why it failed, and it will fail for the next person too.',
              next: 'local',
            },
            {
              label: 'Ask an admin to merge it anyway',
              good: false,
              outcome: 'They say no and ask what is failing. You have to look anyway.',
              next: 'local',
            },
          ],
        },
        {
          id: 'local',
          text: 'You run `npm test -- checkout` on your laptop. It passes 9 times out of 10. On `main`, the same test failed twice this week.',
          choices: [
            {
              label: "Open an issue with links to the failed runs and tell the test's owner",
              good: true,
              outcome:
                'The owner already suspected the sandbox timeouts. They quarantine the test, which means it still runs but cannot block merges. Your next run passes.',
              next: 'merge',
            },
            {
              label: 'Delete the test so your PR passes',
              good: false,
              outcome:
                'The reviewer sees a deleted test in a PR about something else. They ask you to put it back.',
              next: 'merge',
            },
            {
              label: 'Add `sleep(10000)` before the assertion',
              good: false,
              outcome:
                'Now the test suite is 10 seconds slower and still flaky. The sandbox sometimes takes 12 seconds.',
              next: 'merge',
            },
          ],
        },
        {
          id: 'merge',
          text: 'Your PR passes all checks and is approved. It is 18:30.',
          choices: [
            {
              label: 'Merge, and link the flaky-test issue in the PR',
              good: true,
              outcome:
                "Anyone who looks at this week's failed builds later finds the explanation in one click.",
            },
            {
              label: 'Merge and do not mention it',
              good: false,
              outcome: 'Next week someone else spends an hour finding the same flaky test again.',
            },
          ],
        },
      ],
    },
    {
      id: 'ci-jobs-share',
      kind: 'single',
      scenario:
        'Job `build` creates `dist/`. Job `deploy` has `needs: build` and fails with "dist: no such file or directory".',
      prompt: 'Why does `deploy` fail?',
      options: [
        '`needs` only works on self-hosted runners.',
        'Each job runs on its own machine, so `dist/` is not there. Upload it as an artifact and download it in `deploy`.',
        'The jobs ran in the wrong order.',
        'GitHub deletes `dist/` folders for security reasons.',
      ],
      correct: [1],
      explanation: '`needs` sets the order but does not move files. Artifacts move files.',
    },
    {
      id: 'ci-self-hosted',
      kind: 'single',
      scenario:
        'An open-source project wants faster builds. Someone suggests running all PR checks on a spare server in the office.',
      prompt: 'What is the problem?',
      options: [
        'Self-hosted runners cannot run Linux.',
        "Anyone can open a PR, so strangers' code would run on a machine inside the office network.",
        'Self-hosted runners do not support npm.',
        'There is no problem.',
      ],
      correct: [1],
      explanation:
        'On a public repo you do not control who starts workflows. Use self-hosted runners for private repos only.',
    },
    {
      id: 'ci-secrets',
      kind: 'multiple',
      prompt: 'Which of these are true about secrets in GitHub Actions?',
      options: [
        'Secret values are masked (hidden) in logs.',
        'Workflows started by PRs from forks do not get repository secrets.',
        'A `permissions:` block can limit what `GITHUB_TOKEN` can do.',
        'Masking means a script cannot leak a secret.',
      ],
      correct: [0, 1, 2],
      explanation:
        'Masking only hides the exact value. If a script encodes or splits the secret, it is visible in the log.',
    },
    {
      id: 'ci-patch-target',
      kind: 'patch',
      prompt: 'Review this workflow change. Flag the lines that are a security problem.',
      file: '.github/workflows/preview.yml',
      lines: [
        { type: 'context', text: 'on:' },
        {
          type: 'add',
          text: '  pull_request_target:',
          issue:
            '`pull_request_target` runs with your secrets. Combined with checking out PR code, a fork PR can read them.',
        },
        { type: 'context', text: 'permissions:' },
        { type: 'context', text: '  contents: read' },
        { type: 'context', text: 'jobs:' },
        { type: 'context', text: '  preview:' },
        { type: 'context', text: '    runs-on: ubuntu-latest' },
        { type: 'context', text: '    steps:' },
        { type: 'context', text: '      - uses: actions/checkout@v5' },
        { type: 'context', text: '        with:' },
        {
          type: 'add',
          // biome-ignore lint/suspicious/noTemplateCurlyInString: GitHub Actions expression syntax, not a JS template
          text: '          ref: ${{ github.event.pull_request.head.sha }}',
          issue: 'This checks out the PR code, so code from a stranger runs with your secrets.',
        },
        { type: 'context', text: '      - run: npm ci && npm run build' },
      ],
    },
  ],
};
