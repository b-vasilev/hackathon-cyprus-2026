import type { FoundationModule } from '../types';

const safariTitle = 'Fix Safari export button';

export const pullRequests: FoundationModule = {
  id: 'pull-requests',
  title: 'Issues, pull requests and code review',
  summary: 'Describe work in issues, ship it in PRs, and link the two properly.',
  outcome: 'Open a PR people can review quickly, which closes its issue by itself when merged.',
  minutes: 25,
  lessons: [
    {
      id: 'issues',
      title: 'Writing an issue someone can pick up',
      summary: 'What a good bug report or task description contains.',
      blocks: [
        {
          type: 'p',
          text: 'Issues are where work gets described before anyone writes code. Each one has a number like `#42`, and you can mention it from commits, PRs and other issues in the same repo.',
        },
        {
          type: 'p',
          text: "Write it for someone who wasn't in the meeting. For a bug, that means: what you did, what you expected, what actually happened, how to reproduce it, and where (version, browser, OS). For a feature or task: what problem it solves, who for, and how we'll know it's done.",
        },
        {
          type: 'list',
          items: [
            "One problem per issue. If it won't fit in one PR, split it.",
            "Use labels, assignees and the project board for type, owner and status. Don't bury that in comments.",
            'Search before you open one. Adding detail to an existing issue beats a duplicate.',
          ],
        },
      ],
    },
    {
      id: 'linking',
      title: 'Linking a PR to its issue',
      summary: 'The keywords that make GitHub close the issue when the PR merges.',
      blocks: [
        {
          type: 'p',
          text: 'Put a closing keyword and the issue number in the PR description, for example `Closes #42`. GitHub links the two, and when the PR is merged into the default branch (usually `main`) the issue closes on its own.',
        },
        {
          type: 'p',
          text: 'Keywords GitHub understands: `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, `resolved`.',
        },
        {
          type: 'table',
          columns: ['Case', 'Write', 'Result'],
          rows: [
            ['Issue in the same repo', '`Closes #42`', 'Linked, closes on merge'],
            ['Issue in another repo', '`Fixes octo-org/billing#7`', 'Linked, closes on merge'],
            ['Several issues', '`Resolves #10, resolves #12`', 'Both close (keyword repeated)'],
            ['Related, not finished', '`Part of #42`', 'Shows up on the issue, which stays open'],
          ],
        },
        {
          type: 'p',
          text: 'Where people slip up:',
        },
        {
          type: 'list',
          items: [
            'Keywords work in the PR description and in commit messages. In a PR comment they do nothing.',
            'If the PR goes into `develop` or `release/1.2` instead of the default branch, GitHub ignores the keyword and the issue stays open.',
            '`Closes #10, #12` only closes #10. Every issue needs its own keyword.',
            'In a commit message, the issue closes when that commit reaches the default branch.',
            'You can also link by hand from the PR sidebar, under "Development".',
          ],
        },
        {
          type: 'code',
          code: 'git commit -m "fix: retry login after token expiry" -m "Fixes #42"',
          caption: 'The second `-m` becomes the commit body.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: "Doing half the work? Don't close the issue",
          text: 'If the PR covers only part of an issue, write "Part of #42". A closing keyword would close it with work still left.',
        },
      ],
    },
    {
      id: 'good-pr',
      title: "A PR that's easy to review",
      summary: 'Size, description, and what to check before you ask anyone to look.',
      blocks: [
        {
          type: 'p',
          text: 'A pull request asks to merge your branch into another one, usually `main`. Review, CI checks and discussion all happen there.',
        },
        {
          type: 'p',
          text: 'Size matters more than anything. People review 200 lines carefully. They skim 2,000 and hit approve. If your PR is getting big, see whether it splits into a refactor PR and a feature PR.',
        },
        {
          type: 'p',
          text: "Before you request a review, read your own diff on GitHub. You'll find the `console.log`, the commented-out block and the file you didn't mean to touch. Wait for CI to go green too. Nobody enjoys reviewing something that doesn't build.",
        },
        {
          type: 'code',
          code: '## What\nAdds CSV export to the monthly report page.\n\n## Why\nFinance copies these numbers into Excel by hand every month.\n\n## How to test\n1. Open Reports, pick any month\n2. Click "Export CSV"\n3. Totals in the file match the page\n\n## Notes\nBig reports are streamed. No new dependencies.\n\nCloses #42',
          caption:
            'A description like this takes five minutes to write and saves the reviewer twenty. Repos often ship a template in `.github/pull_request_template.md`.',
        },
        {
          type: 'p',
          text: 'Want early feedback on something unfinished? Open it as a draft PR. Also worth knowing: a `CODEOWNERS` file automatically requests reviewers for the paths you touched, and branch protection can block merging until there are approvals and green checks.',
        },
      ],
    },
    {
      id: 'code-review',
      title: 'Reviewing and being reviewed',
      summary: 'How to leave useful comments, and how to take them.',
      blocks: [
        {
          type: 'table',
          columns: ['When you submit a review', 'It means'],
          rows: [
            ['Comment', 'Questions or thoughts, no verdict yet.'],
            ['Approve', 'Good to merge, maybe after small agreed tweaks.'],
            ['Request changes', 'Something has to change before this merges.'],
          ],
        },
        {
          type: 'p',
          text: "As a reviewer, start with the big questions. Does it solve the issue? Could it break something? Is anything unsafe? Formatting is the linter's job.",
        },
        {
          type: 'list',
          items: [
            "Make it clear what blocks the merge and what doesn't. Many teams prefix optional comments with `nit:`.",
            'Ask instead of assuming: "What happens if the list is empty?" lands better than "This is broken."',
            "For small fixes, use GitHub's suggestion feature so the author can apply it in one click.",
          ],
        },
        {
          type: 'p',
          text: "As the author, reply to every comment, either with a change or a reason. Comments are about the code. Resolve the conversation once it's handled.",
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'pr-which-closes',
      kind: 'cards',
      look: 'pull-request',
      scenario: 'Issue #87: "Export button does nothing on Safari". `main` is the default branch.',
      prompt: "Which of these PRs will close #87 automatically when it's merged?",
      cards: [
        {
          title: safariTitle,
          meta: 'develop ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nCloses #87',
          flaw: 'develop',
          feedback:
            'The keyword is right, but the PR targets `develop`. GitHub only acts on closing keywords for PRs into the default branch.',
        },
        {
          title: safariTitle,
          meta: 'main ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nRelated to #87',
          flaw: 'Related to #87',
          feedback:
            '"Related to" just mentions the issue. It needs a closing keyword: closes, fixes or resolves.',
        },
        {
          title: 'Fix Safari export and empty-state bugs',
          meta: 'main ← fix/export-bugs',
          body: 'Two small fixes in the export flow.\n\nCloses #86, #87',
          flaw: '#87',
          feedback:
            'Only #86 gets closed. Each issue needs its own keyword: "Closes #86, closes #87".',
        },
        {
          title: safariTitle,
          meta: 'main ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nFixes #87',
          correct: true,
          feedback:
            'Targets the default branch and has a closing keyword right before the issue number. Merging closes #87.',
        },
      ],
    },
    {
      id: 'pr-ready-for-review',
      kind: 'cards',
      look: 'pull-request',
      prompt: 'Which PR would you be happy to be asked to review?',
      cards: [
        {
          title: 'Updates',
          meta: 'main ← dev/stuff · 63 files changed',
          body: '(no description)',
          flaw: '(no description)',
          feedback:
            "No title worth reading, no description, 63 files. The reviewer has to work out what it's for before they can start.",
        },
        {
          title: 'Add CSV export to monthly reports',
          meta: 'main ← feat/42-export-csv · 4 files changed · checks failing',
          body: 'What, why and how to test, all filled in.\n\nCloses #42',
          flaw: 'checks failing',
          feedback:
            'Good description, but CI is red. Fix it first, otherwise the review might be wasted.',
        },
        {
          title: 'Add CSV export to monthly reports',
          meta: 'main ← feat/42-export-csv · 4 files changed · checks passing',
          body: 'What, why and how to test, all filled in.\n\nCloses #42',
          correct: true,
          feedback: 'Small, explained, green, and linked to its issue. Easy to review.',
        },
        {
          title: 'Add CSV export, redesign settings page, upgrade React',
          meta: 'main ← feat/big-update · 118 files changed · checks passing',
          body: 'Lots of improvements!',
          flaw: 'redesign settings page, upgrade React',
          feedback:
            'Three unrelated changes. Split them so each can be reviewed and reverted on its own.',
        },
      ],
    },
    {
      id: 'pr-review-ai-patch',
      kind: 'patch',
      scenario:
        'A teammate asked an AI assistant to "make getUser handle errors better" and opened a PR with the result. You\'re the reviewer.',
      prompt: "Flag every line you'd leave a comment on.",
      file: 'src/api/users.ts',
      lines: [
        { type: 'context', text: 'export async function getUser(id: string) {' },
        { type: 'remove', text: "  const res = await fetch('/api/users/' + id);" },
        {
          type: 'add',
          text: "  const url = 'https://api.example.com/users/' + id + '?token=sk_live_9f8a7c';",
          issue:
            'A live API token hard-coded in the source. It belongs in an environment variable, and this one now needs rotating.',
        },
        { type: 'add', text: '  const res = await fetch(url);' },
        {
          type: 'add',
          text: "  console.log('user response', res);",
          issue: 'Leftover debug logging. It also writes user data into the logs.',
        },
        { type: 'context', text: '  if (!res.ok) {' },
        { type: 'remove', text: "    throw new Error('Failed to load user ' + id);" },
        {
          type: 'add',
          text: '    return null;',
          issue:
            'Errors are now swallowed. Callers can\'t tell "user not found" from "server is down". That\'s worse error handling, not better.',
        },
        { type: 'context', text: '  }' },
        { type: 'context', text: '  return res.json();' },
        { type: 'context', text: '}' },
      ],
    },
    {
      id: 'pr-multiple-issues',
      kind: 'fill',
      prompt: 'Complete the PR description so that merging into `main` closes both issues.',
      template: '{{0}} #10, {{1}} #12',
      answers: ['Closes', 'closes'],
      distractors: ['and', 'See', 'Part of'],
      explanation:
        'Every issue needs its own closing keyword. "See" and "Part of" only mention the issue.',
    },
    {
      id: 'pr-non-default',
      kind: 'single',
      scenario:
        'Your team merges into `develop` and releases to `main` every two weeks. `main` is the default branch. A PR into `develop` says "Fixes #120" and gets merged.',
      prompt: 'What happens to #120?',
      options: [
        'It closes straight away.',
        "It stays open, because the PR didn't go into the default branch.",
        'It gets deleted.',
        'It turns into a discussion.',
      ],
      correct: [1],
      explanation:
        "Keywords only work for PRs into the default branch. It'll close when the commit reaches `main`, or you can close it by hand.",
    },
    {
      id: 'pr-where-keywords',
      kind: 'multiple',
      prompt: 'Where does `Fixes #42` actually do something?',
      options: [
        'In the PR description',
        'In a commit message that ends up on the default branch',
        'In a comment on the PR',
        'In a Slack message about the PR',
      ],
      correct: [0, 1],
      explanation:
        "GitHub reads closing keywords from PR descriptions and commit messages. In comments they're just a mention.",
    },
  ],
};
