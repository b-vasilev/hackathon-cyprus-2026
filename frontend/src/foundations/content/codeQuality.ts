import type { FoundationModule } from '../types';

export const codeQuality: FoundationModule = {
  id: 'code-quality',
  title: 'Linters, formatters and repo hygiene',
  summary: 'Let tools handle style and catch mistakes. Keep secrets out of Git.',
  outcome:
    'Know which tool is complaining and why, run checks before you push, and never commit a credential.',
  minutes: 20,
  lessons: [
    {
      id: 'tools',
      title: 'Formatter, linter, type checker',
      summary: 'Which tool checks what.',
      blocks: [
        {
          type: 'p',
          text: 'They all show up as red squiggles, but they look for different things:',
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
              'Patterns that tend to cause bugs: unused variables, a missing `await`, unsafe calls.',
              'ESLint, `golangci-lint`, Ruff, Pylint',
            ],
            [
              'Type checker',
              'Whether values have the types the code expects.',
              '`tsc`, mypy, the Go compiler',
            ],
            [
              'Security scanner',
              'Known vulnerable dependencies, leaked secrets.',
              '`npm audit`, `govulncheck`, CodeQL',
            ],
          ],
        },
        {
          type: 'p',
          text: 'Some tools do more than one job. Biome and Ruff both format and lint. Their config lives in the repo (`eslint.config.js`, `biome.json`, `.golangci.yml`, `pyproject.toml`), so your laptop and CI check exactly the same rules.',
        },
        {
          type: 'p',
          text: 'The real win: once a formatter decides where the brackets go, nobody argues about it in review anymore.',
        },
      ],
    },
    {
      id: 'running-checks',
      title: 'When to run them',
      summary: 'In the editor, before each commit, and in CI.',
      blocks: [
        {
          type: 'list',
          items: [
            'In your editor: install the formatter and linter extensions the project uses, and turn on format on save.',
            'Before each commit: a pre-commit hook (pre-commit, Husky, Lefthook) runs the fast checks for you.',
            'In CI: the full set, on every PR. This is the one that counts, because nobody can skip it locally.',
          ],
        },
        {
          type: 'p',
          text: 'Most linters can fix a lot by themselves: `eslint --fix`, `ruff check --fix`. Look at what changed before you commit it.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Turning a rule off',
          text: "`// eslint-disable-next-line` or `//nolint` switches a rule off for one line. Sometimes that's fine. Keep it to one line and add a comment saying why. Changing the shared config is a team decision, so do it in its own PR.",
        },
        {
          type: 'p',
          text: "There's also `.editorconfig`, which sets indentation and line endings in pretty much every editor before any of the other tools run.",
        },
      ],
    },
    {
      id: 'secrets',
      title: 'Keeping secrets out of the repo',
      summary: 'Once a secret is pushed, treat it as compromised.',
      blocks: [
        {
          type: 'list',
          items: [
            'Read credentials from environment variables or a secret manager, not from constants in the code.',
            'Commit a `.env.example` with placeholder values, and put the real `.env` in `.gitignore`.',
            "Look at `git status` and the diff before committing. `git add .` will happily add things you didn't mean to share.",
            'Turn on secret scanning and push protection if your platform has them.',
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Pushed a secret by accident?',
          text: "Revoke or rotate it right away, then tell whoever owns it. Deleting the file in a new commit doesn't help: it's still in the history, and in every clone and fork. Rewriting history afterwards is cleanup, not a fix.",
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'cq-safe-file',
      kind: 'cards',
      look: 'file',
      prompt: "You're about to commit. Which of these files is safe to push?",
      cards: [
        {
          title: 'src/config.ts',
          body: "export const stripeKey =\n  'sk_live_51Hx9QeLk2...';",
          flaw: "'sk_live_51Hx9QeLk2...'",
          feedback:
            'A live API key hard-coded in the source. Anyone with access to the repo, or any clone of it, now has it. Read it from the environment.',
        },
        {
          title: '.env',
          body: 'DATABASE_URL=postgres://admin:Winter2026!@db.internal:5432/app',
          flaw: 'admin:Winter2026!',
          feedback:
            'This is the real `.env` with a real password in it. It belongs in `.gitignore`.',
        },
        {
          title: '.env.example',
          body: 'DATABASE_URL=postgres://user:password@localhost:5432/app\nSTRIPE_KEY=',
          correct: true,
          feedback:
            'Placeholders only. It shows people which variables exist without giving away any values.',
        },
        {
          title: 'deploy.sh',
          body: 'curl -H "Authorization: Bearer ghp_7rT2kLm9..." \\\n  https://api.github.com/repos/acme/app/dispatches',
          flaw: 'ghp_7rT2kLm9...',
          feedback:
            'A GitHub token pasted into a script. Pass it in through an environment variable or a CI secret.',
        },
      ],
    },
    {
      id: 'cq-which-tool',
      kind: 'fill',
      prompt: 'Match each problem to the tool that catches it.',
      template:
        'Tabs and spaces mixed in one file → {{0}}\nA variable declared and never used → {{1}}\nPassing a string where a number is expected → {{2}}\nA dependency with a known vulnerability → {{3}}',
      answers: ['formatter', 'linter', 'type checker', 'security scanner'],
      distractors: ['code review', 'unit test'],
      explanation:
        'Formatters handle layout, linters handle suspicious patterns, type checkers handle types, scanners handle known vulnerabilities. Then reviewers can spend their time on logic and design.',
    },
    {
      id: 'cq-source-of-truth',
      kind: 'single',
      scenario: 'Your pre-commit hook passed, but the lint check on the PR is red.',
      prompt: 'Which one decides whether you can merge?',
      options: [
        'The hook, because it ran first',
        'CI, because it runs the agreed config for everyone',
        'Neither, lint is only advice',
        'Whatever the reviewer prefers',
      ],
      correct: [1],
      explanation:
        'Hooks can be skipped or out of date. CI is the same for everyone. Find out why your local run is different.',
    },
    {
      id: 'cq-leaked-secret',
      kind: 'branch',
      prompt: 'What do you do? Each decision changes what happens next.',
      steps: [
        {
          id: 'start',
          text: 'You pushed to your feature branch 20 minutes ago. Now you notice `config.ts` contains the real Stripe live key.',
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
                "It's gone from the latest version, but the previous commit still has it, and the key still works.",
              next: 'code',
            },
            {
              label: 'Force-push to remove the commit and say nothing',
              good: false,
              outcome:
                'Harder to find, but clones, forks and CI logs may already have it, and the key still works.',
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
                "Fixed properly, and the owner can check Stripe's logs for anything odd during those 20 minutes.",
            },
            {
              label: 'Paste the new key into `config.ts`',
              good: false,
              outcome: 'Same problem again, just with a new key.',
            },
          ],
        },
      ],
    },
  ],
};
