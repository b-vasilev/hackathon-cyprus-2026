import type { FoundationModule } from '../types';

// Repeated diagram labels.
const TYPE_CHECKER = 'Type checker';
const SCANNER = 'Security scanner';
const MISSING_SEMICOLON = 'Missing semicolon';
const PATH = 'Path';
const WRITE_CODE = 'Write code';
const SAVE = 'Save';
const COMMIT = 'Commit';
const PUSH = 'Push';
const PR_CHECKS = 'PR checks';
const MERGE = 'Merge';
const LAPTOP = 'Your laptop';
const GITHUB = 'GitHub';
const ENV = '`.env`';
const ENV_EXAMPLE = '`.env.example`';
const REAL_PASSWORD = 'real password';

export const codeQuality: FoundationModule = {
  id: 'code-quality',
  title: 'Linters, formatters and repo hygiene',
  summary: 'Use tools to fix code style and find mistakes for you. Keep secrets out of Git.',
  outcome:
    'Know which tool shows a warning and why, run checks before you push, and never commit a password or key.',
  minutes: 20,
  lessons: [
    {
      id: 'tools',
      title: 'Formatter, linter, type checker',
      summary: 'Which tool checks which kind of problem in your code.',
      blocks: [
        {
          type: 'p',
          text: 'Your editor and CI run several tools on your code. CI (continuous integration) is a server that runs checks on every change you push. Each tool looks for a different kind of problem. In the editor, they all mark the problem line with an underline and a message. Read the message to see which tool wrote it.',
        },
        {
          type: 'list',
          items: [
            'A formatter is a tool that changes how your code looks. It rewrites spacing, quotes and line breaks for you. It does not change what the code does.',
            'A linter is a tool that reads your code and warns about likely mistakes. Examples: a variable you never use, a missing `await`, an unsafe call. It does not run the code.',
            'A type checker is a tool that checks that each value has the type the code expects. For example, it shows an error when you pass text to a function that needs a number.',
            'A security scanner is a tool that looks for known security problems. It checks your dependencies (code libraries your project uses) against a list of known vulnerabilities. Some scanners also look for passwords and keys in the code.',
          ],
        },
        {
          type: 'code-tools',
          title: 'Try it: run the tools on the same code, in any order',
          tools: [
            {
              name: 'Formatter',
              fixedBy: 'tool',
              formats: true,
              summary:
                'Only the layout changes: spaces, quotes, semicolons, indentation and long lines. The code does exactly the same thing as before.',
            },
            {
              name: 'Linter',
              fixedBy: 'you',
              summary:
                'The linter finds code that is never used. Unused code is often a sign of a mistake, and it makes the file harder to read. Some linters can remove it with a fix option.',
            },
            {
              name: TYPE_CHECKER,
              fixedBy: 'you',
              summary:
                '`res.json()` returns a Promise, a value that is not ready yet. Without `await`, `order` is the Promise and not the data, so `order.total` does not work. The type checker finds this before the code runs.',
            },
            {
              name: SCANNER,
              fixedBy: 'you',
              summary:
                'The scanner finds a live API key written in the code. Anyone who can read the repository can use it. Read it from an environment variable instead, and rotate (replace) the leaked key.',
            },
          ],
          statements: [
            {
              text: "import {readFile} from 'fs'",
              formatted: "import { readFile } from 'fs';",
              findings: [
                { tool: 'Formatter', text: 'Missing spaces and semicolon' },
                { tool: 'Linter', text: "'readFile' is imported but never used" },
              ],
              fixes: [{ tool: 'Linter', remove: true }],
            },
            {
              text: 'const API_KEY="sk_live_51Hx9QeLk2"',
              formatted: "const API_KEY = 'sk_live_51Hx9QeLk2';",
              findings: [
                { tool: 'Formatter', text: 'Double quotes, missing spaces and semicolon' },
                { tool: SCANNER, text: 'Hard-coded secret: this looks like a live Stripe API key' },
              ],
              fixes: [
                {
                  tool: SCANNER,
                  replace: [
                    ['"sk_live_51Hx9QeLk2"', 'process.env.STRIPE_KEY'],
                    ["'sk_live_51Hx9QeLk2'", 'process.env.STRIPE_KEY'],
                  ],
                },
              ],
            },
            {
              text: 'export async function getTotal(orderId:string){',
              formatted: 'export async function getTotal(orderId: string) {',
              findings: [{ tool: 'Formatter', text: 'Missing spaces' }],
            },
            {
              text: '  const unused = 42',
              formatted: '  const unused = 42;',
              findings: [
                { tool: 'Formatter', text: MISSING_SEMICOLON },
                { tool: 'Linter', text: "'unused' is assigned a value but never used" },
              ],
              fixes: [{ tool: 'Linter', remove: true }],
            },
            {
              text: '    const res=await fetch("/api/orders/"+orderId,{headers:{Authorization:API_KEY}})',
              formatted:
                "  const res = await fetch('/api/orders/' + orderId, {\n    headers: { Authorization: API_KEY },\n  });",
              findings: [
                { tool: 'Formatter', text: 'Wrong indentation, double quotes, line too long' },
              ],
            },
            {
              text: '  const order = res.json()',
              formatted: '  const order = res.json();',
              findings: [{ tool: 'Formatter', text: MISSING_SEMICOLON }],
              fixes: [{ tool: TYPE_CHECKER, replace: [['res.json()', 'await res.json()']] }],
            },
            {
              text: '  return order.total',
              formatted: '  return order.total;',
              findings: [
                { tool: 'Formatter', text: MISSING_SEMICOLON },
                {
                  tool: TYPE_CHECKER,
                  text: "Property 'total' does not exist on type 'Promise<any>'. Did you forget to use 'await'?",
                },
              ],
            },
            { text: '}' },
          ],
        },
        {
          type: 'table',
          columns: ['Tool', 'Looks at', 'Examples'],
          rows: [
            [
              'Formatter',
              'How the code looks. Rewrites spacing, quotes and line breaks for you.',
              'Prettier, `gofmt`, Black, `ruff format`',
            ],
            [
              'Linter',
              'Patterns that often cause bugs: unused variables, a missing `await`, unsafe calls.',
              'ESLint, `golangci-lint`, Ruff, Pylint',
            ],
            [
              TYPE_CHECKER,
              'Whether values have the types the code expects.',
              '`tsc`, mypy, the Go compiler',
            ],
            [
              'Security scanner',
              'Dependencies with known vulnerabilities, and passwords or keys in the code.',
              '`npm audit`, `govulncheck`, CodeQL',
            ],
          ],
        },
        {
          type: 'p',
          text: 'Some tools do more than one job. Biome and Ruff both format and lint. Their settings are stored in files in the repo (`eslint.config.js`, `biome.json`, `.golangci.yml`, `pyproject.toml`). Because of this, your laptop and CI check exactly the same rules.',
        },
        {
          type: 'p',
          text: 'A formatter also saves time in code review. The tool decides where spaces and brackets go. So reviewers do not need to discuss layout and can focus on the logic.',
        },
      ],
    },
    {
      id: 'running-checks',
      title: 'When to run them',
      summary: 'The three places where checks run: the editor, before each commit, and in CI.',
      blocks: [
        {
          type: 'list',
          items: [
            'In your editor: install the formatter and linter extensions that the project uses. Turn on "format on save". Then the formatter runs each time you save a file.',
            'Before each commit: a pre-commit hook is a small script that Git runs each time you commit. Tools like pre-commit, Husky and Lefthook set it up. It runs the fast checks, and it stops the commit if a check fails.',
            'In CI: the full set of checks runs on every pull request (PR). This is the check that decides, because nobody can skip it on their own laptop.',
          ],
        },
        {
          type: 'scenes',
          title: 'Where each check runs',
          legend: [
            { tone: 'neutral', label: 'Normal step' },
            { tone: 'brand', label: 'Check runs here' },
            { tone: 'danger', label: 'Stopped if it fails' },
          ],
          scenes: [
            {
              name: 'Editor',
              summary: 'The editor formats and lints the file each time you save.',
              description: 'This is the fastest feedback. It only checks the files you open.',
              rows: [
                {
                  label: PATH,
                  blocks: [
                    { text: WRITE_CODE },
                    { text: SAVE, tone: 'brand', note: 'format and lint' },
                    { text: COMMIT },
                    { text: PUSH },
                    { text: PR_CHECKS },
                    { text: MERGE },
                  ],
                },
              ],
            },
            {
              name: 'Pre-commit hook',
              summary: 'The hook runs fast checks when you commit.',
              description:
                'If a check fails, Git does not create the commit. People can skip the hook, so it is only a first check.',
              rows: [
                {
                  label: PATH,
                  blocks: [
                    { text: WRITE_CODE },
                    { text: SAVE },
                    { text: COMMIT, tone: 'brand', note: 'fast checks run' },
                    { text: PUSH },
                    { text: PR_CHECKS },
                    { text: MERGE },
                  ],
                },
                {
                  label: 'If it fails',
                  blocks: [
                    { text: COMMIT, tone: 'danger', note: 'commit is not created' },
                    { text: 'Fix and retry' },
                  ],
                },
              ],
            },
            {
              name: 'CI',
              summary: 'CI runs the full set of checks on every PR.',
              description: 'Everyone gets the same checks. If one fails, the PR cannot be merged.',
              rows: [
                {
                  label: PATH,
                  blocks: [
                    { text: WRITE_CODE },
                    { text: SAVE },
                    { text: COMMIT },
                    { text: PUSH },
                    { text: PR_CHECKS, tone: 'brand', note: 'full set, every PR' },
                    { text: MERGE },
                  ],
                },
                {
                  label: 'If it fails',
                  blocks: [
                    { text: PR_CHECKS, tone: 'danger', note: 'a check failed' },
                    { text: MERGE, tone: 'muted', note: 'blocked' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'p',
          text: 'Many linters can fix many problems by themselves: `eslint --fix`, `ruff check --fix`. Look at what they changed before you commit it.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Turning a rule off',
          text: '`// eslint-disable-next-line` or `//nolint` switches a rule off for one line. Sometimes this is fine. Use it for one line only, and add a comment that says why. Changing the shared settings is a team decision, so do it in a separate PR.',
        },
        {
          type: 'p',
          text: 'There is also `.editorconfig`. This file sets indentation and line endings in almost every editor, before the other tools run.',
        },
      ],
    },
    {
      id: 'secrets',
      title: 'Keeping secrets out of the repo',
      summary: 'How to keep passwords and keys out of Git, and what to do if one is pushed.',
      blocks: [
        {
          type: 'p',
          text: 'A secret is a value that gives access to something: a password, an API key, a token. Once a secret is pushed, treat it as known to other people.',
        },
        {
          type: 'list',
          items: [
            'Read secrets from environment variables (values set outside the code, on the machine or server) or from a secret manager (a service that stores secrets safely). Do not write them as constants in the code.',
            'Commit a `.env.example` file with placeholder values. Put the real `.env` file in `.gitignore`, the list of files Git must not add.',
            'Look at `git status` and the diff before you commit. `git add .` adds every changed file, also files you did not want to share.',
            'Turn on secret scanning and push protection if your platform has them. Push protection blocks a push that contains a secret.',
          ],
        },
        {
          type: 'scenes',
          title: 'What reaches GitHub',
          legend: [
            { tone: 'neutral', label: 'File on your laptop' },
            { tone: 'brand', label: 'Safe to share' },
            { tone: 'muted', label: 'Stays local' },
            { tone: 'danger', label: 'Secret is exposed' },
          ],
          scenes: [
            {
              name: '.env committed',
              summary: 'The real password is now in the repo for everyone to see.',
              description: 'Every clone and fork now has a copy of it.',
              rows: [
                {
                  label: LAPTOP,
                  blocks: [{ text: ENV, note: REAL_PASSWORD }, { text: 'Source code' }],
                },
                {
                  label: GITHUB,
                  blocks: [
                    { text: ENV, tone: 'danger', note: 'password is public' },
                    { text: 'Source code', tone: 'brand' },
                  ],
                },
              ],
            },
            {
              name: '.env.example + .gitignore',
              summary: 'Only placeholder values reach GitHub.',
              description:
                'Git ignores the real `.env`. Other people copy `.env.example` and add their own values.',
              rows: [
                {
                  label: LAPTOP,
                  blocks: [
                    { text: ENV, note: REAL_PASSWORD },
                    { text: ENV_EXAMPLE, note: 'placeholders' },
                    { text: '`.gitignore`', note: 'lists `.env`' },
                  ],
                },
                {
                  label: GITHUB,
                  blocks: [
                    { text: ENV, tone: 'muted', note: 'not pushed' },
                    { text: ENV_EXAMPLE, tone: 'brand', note: 'placeholders only' },
                    { text: '`.gitignore`', tone: 'brand' },
                  ],
                },
              ],
            },
            {
              name: 'Deleted later',
              summary: 'Deleting the file in a new commit does not remove the secret.',
              description: 'The old commit still has the file. Anyone can open that commit.',
              rows: [
                {
                  label: 'History',
                  blocks: [
                    { text: 'Commit 1', tone: 'danger', note: `adds ${ENV}` },
                    { text: 'Commit 2', note: `deletes ${ENV}` },
                    { text: 'Latest', tone: 'strong', note: 'secret still in history' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Pushed a secret by mistake?',
          text: 'Revoke or rotate it right away. This means you cancel the old secret, or replace it with a new one, so the old value stops working. Then tell the person who owns it. Deleting the file in a new commit does not help: it is still in the history, and in every clone and fork. Rewriting history afterwards only cleans up. It does not fix the leak.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'cq-safe-file',
      kind: 'cards',
      look: 'file',
      prompt: 'You are about to commit. Which of these files is safe to push?',
      cards: [
        {
          title: 'src/config.ts',
          body: "export const stripeKey =\n  'sk_live_51Hx9QeLk2...';",
          flaw: "'sk_live_51Hx9QeLk2...'",
          feedback:
            'A live API key is written directly in the source code. Anyone with access to the repo, or to any clone of it, now has it. Read it from the environment.',
        },
        {
          title: '.env',
          body: 'DATABASE_URL=postgres://admin:Winter2026!@db.internal:5432/app',
          flaw: 'admin:Winter2026!',
          feedback:
            'This is the real `.env` file with a real password in it. It belongs in `.gitignore`.',
        },
        {
          title: '.env.example',
          body: 'DATABASE_URL=postgres://user:password@localhost:5432/app\nSTRIPE_KEY=',
          correct: true,
          feedback:
            'It has placeholders only. It shows people which variables exist, without sharing any real values.',
        },
        {
          title: 'deploy.sh',
          body: 'curl -H "Authorization: Bearer ghp_7rT2kLm9..." \\\n  https://api.github.com/repos/acme/app/dispatches',
          flaw: 'ghp_7rT2kLm9...',
          feedback:
            'A GitHub token is pasted into a script. Pass it in through an environment variable or a CI secret.',
        },
      ],
    },
    {
      id: 'cq-which-tool',
      kind: 'fill',
      prompt: 'Match each problem to the tool that finds it.',
      template:
        'Tabs and spaces mixed in one file → {{0}}\nA variable declared and never used → {{1}}\nPassing a string where a number is expected → {{2}}\nA dependency with a known vulnerability → {{3}}',
      answers: ['formatter', 'linter', 'type checker', 'security scanner'],
      distractors: ['code review', 'unit test'],
      explanation:
        'Formatters fix layout. Linters find patterns that often cause bugs. Type checkers find wrong types. Scanners find known vulnerabilities. Then reviewers can spend their time on logic and design.',
    },
    {
      id: 'cq-formatter-fixes',
      kind: 'multiple',
      prompt: 'Which of these can a formatter fix for you?',
      options: [
        'Spacing and indentation',
        'Single or double quotes',
        'A missing `await`',
        'An API key in the code',
      ],
      correct: [0, 1],
      explanation:
        'A formatter only changes how the code looks. A missing `await` is a job for the linter. A key in the code is a job for a secret scanner and for you.',
    },
    {
      id: 'cq-source-of-truth',
      kind: 'single',
      scenario: 'Your pre-commit hook passed, but the lint check on the PR failed.',
      prompt: 'Which one decides whether you can merge?',
      options: [
        'The hook, because it ran first',
        'CI, because it runs the agreed settings for everyone',
        'Neither, lint is only advice',
        'Whatever the reviewer prefers',
      ],
      correct: [1],
      explanation:
        'People can skip a hook, or have an old version of it. CI is the same for everyone. Find out why your local result is different.',
    },
    {
      id: 'cq-review-diff',
      kind: 'patch',
      prompt: 'Flag the lines that a reviewer should ask you to change.',
      file: 'src/payments.ts',
      lines: [
        { type: 'context', text: "import { charge } from './stripe';" },
        { type: 'context', text: '' },
        { type: 'add', text: 'export async function pay(order: Order) {' },
        {
          type: 'add',
          text: "  const key = 'sk_live_51Hx9QeLk2...';",
          issue: 'A live secret key is written in the code. Read it from an environment variable.',
        },
        {
          type: 'add',
          text: '  // eslint-disable-next-line',
          issue:
            'This turns off every lint rule for the next line, with no comment that says why. Name the rule and give a reason.',
        },
        { type: 'add', text: '  const result: any = await charge(order, key);' },
        { type: 'add', text: '  return result;' },
        { type: 'add', text: '}' },
      ],
    },
    {
      id: 'cq-leaked-secret',
      kind: 'branch',
      prompt: 'What do you do? Each decision changes what happens next.',
      steps: [
        {
          id: 'start',
          text: 'You pushed to your feature branch 20 minutes ago. Now you see that `config.ts` contains the real Stripe live key.',
          choices: [
            {
              label: 'Rotate the key in the Stripe dashboard first',
              good: true,
              outcome:
                'The leaked key stops working. Whatever happens to the repo now, that value is useless.',
              next: 'code',
            },
            {
              label: 'Delete the line, commit and push',
              good: false,
              outcome:
                'It is gone from the latest version. But the previous commit still has it, and the key still works.',
              next: 'code',
            },
            {
              label: 'Force-push to remove the commit and say nothing',
              good: false,
              outcome:
                'It is harder to find. But clones, forks and CI logs may already have it, and the key still works.',
              next: 'code',
            },
          ],
        },
        {
          id: 'code',
          text: 'Now you need to fix the code itself.',
          choices: [
            {
              label:
                "Read the key from an environment variable, store it as a CI secret, and tell the key's owner",
              good: true,
              outcome:
                "This is the correct fix. The owner can also check Stripe's logs for unusual activity during those 20 minutes.",
            },
            {
              label: 'Paste the new key into `config.ts`',
              good: false,
              outcome: 'This is the same problem again, only with a new key.',
            },
          ],
        },
      ],
    },
  ],
};
