import type { FoundationModule } from '../types';

const safariTitle = 'Fix Safari export button';
const openPr = 'Open PR';
const switchMain = '`git switch main`';
const commitA = 'Commit A';
const commitB = 'Commit B';
const commitC = 'Commit C';
const laptopLabel = 'Your laptop';
const gitPull = '`git pull`';
const switchNew = '`git switch -c`';
const gitCommit = '`git commit`';
const gitPush = '`git push`';
const prLabel = 'Pull request';

export const git: FoundationModule = {
  id: 'git',
  title: 'Git and GitHub: from branch to merged PR',
  summary:
    'Branches, commits, rebase, conflicts, issues, pull requests and code review in one workflow.',
  outcome:
    'Start work on a branch, keep it up to date, open a PR that is easy to review, and close its issue when it is merged.',
  minutes: 40,
  lessons: [
    {
      id: 'commits-and-branches',
      title: 'Commits and branches',
      summary: 'What Git stores, and how to start a new piece of work.',
      blocks: [
        {
          type: 'p',
          text: 'A commit is a full snapshot of the project. It also stores a message and a link to the commit before it. That earlier commit is called its parent. Each commit has an ID called a SHA. Git calculates the SHA from all of this data. If anything changes, even the parent, the SHA changes too. This fact explains most of this module.',
        },
        {
          type: 'p',
          text: 'A branch is only a name that points at one commit. When you commit, the name moves forward to the new commit. `HEAD` means "the commit you are on right now".',
        },
        {
          type: 'p',
          text: "The daily routine has six steps. Get the latest `main`. Make a branch for your change. Commit. Push. Open a pull request (PR), a request to merge your branch into `main`. Don't commit directly to `main` in a shared repo. Every change goes through a PR.",
        },
        {
          type: 'scenes',
          title: 'The daily routine',
          legend: [
            { tone: 'neutral', label: 'Normal step' },
            { tone: 'brand', label: 'Done well' },
            { tone: 'muted', label: 'Skipped' },
            { tone: 'warning', label: 'Causes problems later' },
            { tone: 'danger', label: 'Not allowed' },
          ],
          scenes: [
            {
              name: 'Good routine',
              summary: 'You start from the latest main and every change goes through a PR.',
              rows: [
                {
                  label: laptopLabel,
                  blocks: [
                    { text: switchMain },
                    { text: gitPull, tone: 'brand', note: 'get the latest main' },
                    { text: switchNew, tone: 'brand', note: 'new branch' },
                    { text: gitCommit },
                    { text: gitPush },
                  ],
                },
                {
                  label: 'GitHub',
                  blocks: [{ text: openPr, tone: 'brand', note: 'review and CI' }],
                },
              ],
            },
            {
              name: 'No pull',
              summary: 'Your branch starts from an old main.',
              description:
                'Your work does not include changes from your teammates. You get more conflicts later.',
              rows: [
                {
                  label: laptopLabel,
                  blocks: [
                    { text: switchMain },
                    { text: gitPull, tone: 'muted', note: 'skipped' },
                    { text: switchNew, tone: 'warning', note: 'starts from old main' },
                    { text: gitCommit },
                    { text: gitPush },
                  ],
                },
                {
                  label: 'GitHub',
                  blocks: [{ text: openPr, tone: 'warning', note: 'may conflict with main' }],
                },
              ],
            },
            {
              name: 'Commit on main',
              summary: 'You skip the branch and the PR.',
              description: 'Nobody reviews the change. In a shared repo, this is not allowed.',
              rows: [
                {
                  label: laptopLabel,
                  blocks: [
                    { text: switchMain },
                    { text: gitPull },
                    { text: switchNew, tone: 'muted', note: 'skipped' },
                    { text: gitCommit, tone: 'danger', note: 'directly on main' },
                    { text: gitPush, tone: 'danger' },
                  ],
                },
                { label: 'GitHub', blocks: [{ text: openPr, tone: 'muted', note: 'no review' }] },
              ],
            },
          ],
        },
        {
          type: 'code',
          code: 'git switch main\ngit pull\ngit switch -c feat/42-export-csv\n# ...work...\ngit add -p        # choose what goes in, one part at a time\ngit commit\ngit push -u origin feat/42-export-csv',
          caption:
            '`git add -p` shows you every change before it goes into the commit. This helps you find debug lines you forgot to remove.',
        },
        {
          type: 'p',
          text: 'Name the branch after the work. Do not name it after yourself. A good name tells people what the branch is for and which issue it belongs to. They do not need to open it. A common pattern is `type/issue-number-short-description`. Use lowercase letters and hyphens.',
        },
        {
          type: 'table',
          columns: ['Branch name', 'Verdict'],
          rows: [
            [
              '`fix/87-safari-export-button`',
              'Good. It has the type, the issue number and what it fixes.',
            ],
            ['`feat/42-export-csv`', 'Good. Anyone can find issue #42 from it.'],
            [
              '`tatiana`',
              'It says who, but not what. You will have five pieces of work and one name.',
            ],
            ['`fix`', 'It does not say what it fixes. Many branches could have this name.'],
            [
              '`my-branch`, `test2`, `new`',
              'These names mean nothing to anyone, and to you next month.',
            ],
            [
              '`Fix_Safari Export!!`',
              'Spaces, capital letters and symbols cause problems in scripts and URLs.',
            ],
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: "Follow your team's convention",
          text: "Some teams use `feature/` instead of `feat/`. Some put the ticket key first, like `PROJ-87-safari-export`. Look at the repo's existing branches and use the same style.",
        },
        {
          type: 'p',
          text: 'Write the commit subject as an instruction, for example "Add CSV export". Do not write "added stuff". Use the body of the message to explain why you made the change. Many teams also add a type prefix at the start. This format is called Conventional Commits.',
        },
        {
          type: 'anatomy',
          title: 'A good commit message',
          segments: [
            { text: 'fix:', part: 'type' },
            { text: ' ' },
            { text: 'retry login after token expiry', part: 'subject' },
            { text: '\n' },
            { text: '\n', part: 'blank' },
            {
              text: 'Users were logged out when the access token expired\nmid-session. Refresh the token once before giving up.',
              part: 'body',
            },
            { text: '\n\n' },
            { text: 'Fixes #42', part: 'footer' },
          ],
          parts: [
            {
              id: 'type',
              label: 'Type prefix',
              explanation:
                'Says what kind of change this is. `fix:` means a bug fix. The table below lists the common types.',
            },
            {
              id: 'subject',
              label: 'Subject',
              explanation:
                'A short summary, written as an instruction. Keep it under about 50 characters.',
            },
            {
              id: 'blank',
              label: 'Blank line',
              explanation:
                'An empty line separates the subject from the body. Git and GitHub use it to find where the subject ends.',
            },
            {
              id: 'body',
              label: 'Body',
              explanation: 'Explains why you made the change. The code already shows what changed.',
            },
            {
              id: 'footer',
              label: 'Closing keyword',
              explanation:
                '`Fixes #42` closes issue #42 when this commit reaches the default branch.',
            },
          ],
        },
        {
          type: 'table',
          columns: ['Prefix', 'For', 'Example'],
          rows: [
            ['`feat:`', 'Something new for users', '`feat: add CSV export to reports`'],
            ['`fix:`', 'A bug fix', '`fix: retry login after token expiry`'],
            ['`docs:`', 'Documentation only', '`docs: describe local setup`'],
            [
              '`refactor:`',
              'Code restructured, same behavior',
              '`refactor: extract price calculator`',
            ],
            ['`test:`', 'Tests', '`test: cover empty cart checkout`'],
            ['`ci:` / `chore:`', 'Pipeline, tools, maintenance', '`ci: cache npm dependencies`'],
          ],
        },
      ],
    },
    {
      id: 'merge-rebase-squash',
      title: 'Keeping your branch up to date: merge, rebase and squash',
      summary: 'Three ways to bring main into your branch and your branch into main.',
      blocks: [
        {
          type: 'p',
          text: 'While you work, other people merge their changes into `main`. At some point you need their changes in your branch. Later, your branch must be merged into `main`. Git has three ways to combine history.',
        },
        { type: 'git-history' },
        {
          type: 'table',
          columns: ['', 'What it does', 'History afterwards', 'When to use it'],
          rows: [
            [
              'Merge',
              'Adds a merge commit with two parents. Existing commits do not change.',
              'Shows exactly what happened, including merge commits.',
              'Bringing `main` into a branch that other people also use.',
            ],
            [
              'Rebase',
              'Copies your commits on top of the latest `main`. Each copy gets a new SHA.',
              "A straight line, as if you started from today's `main`.",
              'Cleaning up your own branch before review.',
            ],
            [
              'Squash',
              'Turns the whole branch into one commit.',
              'One commit per PR on `main`.',
              'Merging a PR that has many "wip" and "fix typo" commits.',
            ],
          ],
        },
        {
          type: 'code',
          code: 'git fetch origin\ngit rebase origin/main\ngit push --force-with-lease',
          caption:
            'This updates your own branch. After a rebase, a normal push is rejected because the history changed.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'The golden rule: only rebase your own branch',
          text: 'A rebase makes new commits. A teammate may already have the old ones, for example on a shared feature branch or on `main`. Then their copy and yours no longer match, and someone must fix it by hand. On shared branches, use merge.',
        },
        {
          type: 'p',
          text: 'Use `--force-with-lease` instead of `--force`. Someone may have pushed to the branch since your last fetch. In that case `--force-with-lease` refuses to push. Plain `--force` deletes their commits without a warning.',
        },
        {
          type: 'p',
          text: 'Before you ask for review, you can clean up with an interactive rebase: `git rebase -i origin/main`. You can reorder commits and fix messages. You can also join a small fix into the commit it belongs to (`fixup`).',
        },
        {
          type: 'p',
          text: 'The merge button on GitHub has the same three options: "Create a merge commit", "Squash and merge" and "Rebase and merge". The repo settings control which options are allowed. GitHub\'s "Rebase and merge" always creates new SHAs. It does this even when a fast-forward was possible. A fast-forward only moves the branch name forward and makes no new commits.',
        },
      ],
    },
    {
      id: 'conflicts',
      title: 'Resolving conflicts',
      summary: 'What to do when Git stops and asks you to choose.',
      blocks: [
        {
          type: 'p',
          text: 'A conflict happens when both sides changed the same lines. It also happens when one side edited a file and the other side deleted it. Conflicts are normal. Git cannot know which version is right, so it stops and marks the file:',
        },
        {
          type: 'code',
          code: '<<<<<<< HEAD\nconst timeout = 30;\n=======\nconst timeout = 60;\n>>>>>>> feat/slow-network',
          caption:
            'The two versions are between the markers. Delete the markers and keep the code you want.',
        },
        {
          type: 'list',
          ordered: true,
          items: [
            '`git status` shows which files have conflicts.',
            'Fix each file and remove every marker. Often the right result combines both sides.',
            'Run `git add` on the fixed files.',
            'Run `git merge --continue` or `git rebase --continue`.',
            'Run the tests. Code without conflict markers can still be broken.',
          ],
        },
        {
          type: 'p',
          text: 'Want to stop? `git merge --abort` or `git rebase --abort` puts everything back as it was before.',
        },
        {
          type: 'callout',
          tone: 'tip',
          title: '"Ours" and "theirs" swap during a rebase',
          text: 'In a merge, "ours" is your branch. In a rebase, Git copies your commits onto `main`. So "ours" is `main` and "theirs" is your work. This confuses many people. Read the code and ignore the labels.',
        },
      ],
    },
    {
      id: 'undo',
      title: 'Undoing changes',
      summary: 'Which undo command to use, based on who already has the commit.',
      blocks: [
        {
          type: 'table',
          columns: ['You want to…', 'Command', 'Rewrites history?'],
          rows: [
            [
              'Throw away uncommitted edits in a file',
              '`git restore <file>`',
              'No, but those edits are gone',
            ],
            ['Unstage a file but keep the edits', '`git restore --staged <file>`', 'No'],
            [
              'Fix the last commit before you push',
              '`git commit --amend`',
              'Yes (this is fine, it is only local)',
            ],
            [
              'Undo a commit that is already on `main`',
              '`git revert <sha>`',
              'No, it adds a new commit',
            ],
            ['Move your local branch back', '`git reset --hard <sha>`', 'Yes'],
            ['Put your changes aside for a moment', '`git stash`, later `git stash pop`', 'No'],
          ],
        },
        {
          type: 'p',
          text: "If other people already have the commit, use `git revert`. It adds a new commit that does the opposite. Nobody else's history changes.",
        },
        {
          type: 'scenes',
          title: 'Undoing a bad commit on a shared main',
          legend: [
            { tone: 'neutral', label: 'Normal commit' },
            { tone: 'warning', label: 'Bad commit' },
            { tone: 'brand', label: 'The fix' },
            { tone: 'muted', label: 'Removed' },
            { tone: 'danger', label: 'Out of sync' },
          ],
          scenes: [
            {
              name: 'Revert',
              summary: 'Revert adds a new commit that cancels Commit C.',
              description:
                'Your teammate pulls it like any other commit. Both histories still match.',
              rows: [
                {
                  label: 'GitHub main',
                  blocks: [
                    { text: commitA },
                    { text: commitB },
                    { text: commitC, tone: 'warning', note: 'broke the build' },
                    { text: 'Revert C', tone: 'brand', note: 'new commit' },
                  ],
                },
                {
                  label: 'Teammate',
                  blocks: [
                    { text: commitA },
                    { text: commitB },
                    { text: commitC, tone: 'warning' },
                    { text: 'Revert C', tone: 'brand', note: 'normal `git pull`' },
                  ],
                },
              ],
            },
            {
              name: 'Reset',
              summary: 'Reset and force-push remove Commit C from main.',
              description:
                'Your teammate already has Commit C. Now the two histories do not match.',
              rows: [
                {
                  label: 'GitHub main',
                  blocks: [
                    { text: commitA },
                    { text: commitB },
                    { text: commitC, tone: 'muted', note: 'removed by reset' },
                  ],
                },
                {
                  label: 'Teammate',
                  blocks: [
                    { text: commitA },
                    { text: commitB },
                    { text: commitC, tone: 'danger', note: 'still has it' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Lost a commit? Check the reflog',
          text: '`git reflog` lists every place `HEAD` has been recently. It also shows commits that no branch points to anymore. After a bad reset or rebase, find the SHA there. Then run `git switch -c rescue <sha>`.',
        },
      ],
    },
    {
      id: 'issues-and-linking',
      title: 'Issues and linking a PR to an issue',
      summary: 'How to write a good issue and how to make a PR close it when merged.',
      blocks: [
        {
          type: 'p',
          text: 'An issue describes a piece of work before anyone writes code. Each issue has a number like `#42`. You can mention it from commits, PRs and other issues in the same repo.',
        },
        {
          type: 'p',
          text: 'Write the issue for someone who was not in the meeting. For a bug, write what you did, what you expected and what happened. Add the steps to reproduce it and where it happened (version, browser, operating system). For a feature or task, write what problem it solves, for whom, and how you will know it is done.',
        },
        {
          type: 'list',
          items: [
            'One problem per issue. If it does not fit in one PR, split it.',
            'Use labels, assignees and the project board for type, owner and status. Do not hide this information in comments.',
            'Search before you open a new issue. Adding detail to an existing issue is better than a duplicate.',
          ],
        },
        {
          type: 'p',
          text: 'To link a PR to its issue, write a closing keyword and the issue number in the PR description, for example `Closes #42`. The default branch is the main branch of the repo, usually `main`. When the PR is merged into the default branch, GitHub closes the issue.',
        },
        {
          type: 'p',
          text: 'GitHub understands nine keywords: `close`, `closes`, `closed`, `fix`, `fixes`, `fixed`, `resolve`, `resolves`, `resolved`.',
        },
        {
          type: 'p',
          text: 'Try it. Write your own PR description and switch the target branch. See which issues GitHub would close.',
        },
        {
          type: 'issue-simulator',
          defaultBranch: 'main',
          branches: ['main', 'develop', 'release/1.2'],
          initialText: 'Fixes #87',
          examples: [
            'Fixes #87',
            'Closes #86, #87',
            'Closes #86, closes #87',
            'Related to #87',
            'Fixes octo-org/billing#7',
          ],
        },
        {
          type: 'table',
          columns: ['Case', 'Write', 'Result'],
          rows: [
            ['Issue in the same repo', '`Closes #42`', 'Linked, closes on merge'],
            ['Issue in another repo', '`Fixes octo-org/billing#7`', 'Linked, closes on merge'],
            ['Several issues', '`Resolves #10, resolves #12`', 'Both close (keyword repeated)'],
            ['Related, not finished', '`Part of #42`', 'Shown on the issue, which stays open'],
          ],
        },
        {
          type: 'list',
          items: [
            'Keywords work in the PR description and in commit messages. In a PR comment they do nothing.',
            'If the PR goes into `develop` or `release/1.2`, GitHub ignores the keyword. The issue stays open.',
            '`Closes #10, #12` closes only #10. Every issue needs its own keyword.',
            'For an issue in another repo, write `owner/repo#7`.',
            'In a commit message, the issue closes when that commit reaches the default branch.',
            'You can also link an issue by hand in the PR sidebar, under "Development".',
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
          title: 'Only part of the work? Do not close the issue',
          text: 'If the PR covers only part of an issue, write "Part of #42". A closing keyword would close the issue while work is still left.',
        },
      ],
    },
    {
      id: 'pull-requests-and-review',
      title: 'Pull requests and code review',
      summary: 'How to open a PR that is easy to review, and how to review one.',
      blocks: [
        {
          type: 'p',
          text: 'A pull request asks to merge your branch into another branch, usually `main`. Review, discussion and CI checks all happen there. CI (continuous integration) means automatic checks, like tests, that run on every push.',
        },
        {
          type: 'p',
          text: 'Keep PRs small. People review 200 lines carefully. With 2,000 lines, they read quickly and approve without real checks. If your PR grows big, try to split it, for example into one refactor PR and one feature PR.',
        },
        {
          type: 'p',
          text: 'Before you ask for a review, read your own diff on GitHub. You will often find a `console.log`, a block of commented-out code or a file you changed by mistake. Also wait until all CI checks pass (green). A reviewer should not spend time on code that does not build.',
        },
        {
          type: 'code',
          code: '## What\nAdds CSV export to the monthly report page.\n\n## Why\nFinance copies these numbers into Excel by hand every month.\n\n## How to test\n1. Open Reports, pick any month\n2. Click "Export CSV"\n3. Totals in the file match the page\n\n## Notes\nBig reports are streamed. No new dependencies.\n\nCloses #42',
          caption:
            'This description takes five minutes to write. It saves the reviewer twenty minutes. Many repos have a template in `.github/pull_request_template.md`.',
        },
        {
          type: 'p',
          text: 'Want early feedback on unfinished work? Open a draft PR. A draft PR is visible to the team but cannot be merged yet. A `CODEOWNERS` file lists who owns which paths. GitHub then requests reviews from those people automatically. Branch protection is a repo setting. It can block the merge until the PR has approvals and passing checks.',
        },
        {
          type: 'scenes',
          title: 'The life of a pull request',
          legend: [
            { tone: 'neutral', label: 'Start' },
            { tone: 'brand', label: 'Passed' },
            { tone: 'soft', label: 'New commits' },
            { tone: 'warning', label: 'Needs work' },
            { tone: 'danger', label: 'Failed' },
            { tone: 'muted', label: 'Not reached' },
            { tone: 'strong', label: 'Merged' },
          ],
          scenes: [
            {
              name: 'Happy path',
              summary: 'CI passes, the reviewer approves, and the PR is merged.',
              rows: [
                {
                  label: prLabel,
                  blocks: [
                    { text: openPr },
                    { text: 'CI checks', tone: 'brand', note: 'all green' },
                    { text: 'Review', tone: 'brand', note: 'Approve' },
                    { text: 'Merge', tone: 'strong', note: 'issue closes' },
                  ],
                },
              ],
            },
            {
              name: 'Changes requested',
              summary: 'The reviewer asks for changes, and you push new commits.',
              description: 'The reviewer looks again. After approval, the PR is merged.',
              rows: [
                {
                  label: prLabel,
                  blocks: [
                    { text: openPr },
                    { text: 'CI checks', tone: 'brand' },
                    { text: 'Review', tone: 'warning', note: 'Request changes' },
                    { text: 'Push fixes', tone: 'soft', note: 'new commits' },
                    { text: 'Review again', tone: 'brand', note: 'Approve' },
                    { text: 'Merge', tone: 'strong' },
                  ],
                },
              ],
            },
            {
              name: 'CI fails',
              summary: 'A check fails, so the PR is not ready for review.',
              description:
                'Fix the failing check first. Branch protection can block the merge until all checks pass.',
              rows: [
                {
                  label: prLabel,
                  blocks: [
                    { text: openPr },
                    { text: 'CI checks', tone: 'danger', note: 'tests fail' },
                    { text: 'Review', tone: 'muted', note: 'not started' },
                    { text: 'Merge', tone: 'muted', note: 'blocked' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'table',
          columns: ['When you submit a review', 'It means'],
          rows: [
            ['Comment', 'Questions or thoughts, no decision yet.'],
            ['Approve', 'Ready to merge, maybe after small changes you agreed on.'],
            ['Request changes', 'Something must change before this is merged.'],
          ],
        },
        {
          type: 'p',
          text: 'As a reviewer, start with the big questions. Does it solve the issue? Could it break something? Is anything unsafe? Formatting is the job of the linter, a tool that reads your code and warns about likely mistakes and style problems.',
        },
        {
          type: 'list',
          items: [
            'Make it clear which comments block the merge and which are optional. Many teams start optional comments with `nit:`, short for "nitpick", a small detail.',
            'Ask questions instead of making claims. "What happens if the list is empty?" is better than "This is broken."',
            'For small fixes, use the GitHub suggestion feature. The author can then apply it with one click.',
          ],
        },
        {
          type: 'p',
          text: 'As the author, reply to every comment. Either make the change or explain why not. Comments are about the code, not about you. Resolve the conversation when it is handled.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'git-branch-name',
      kind: 'cards',
      look: 'branch',
      scenario: 'Issue #87: "Export button does nothing on Safari". It is assigned to you.',
      prompt: 'Which branch name do you create?',
      cards: [
        {
          title: 'tatiana',
          meta: 'git switch -c tatiana',
          flaw: 'tatiana',
          feedback:
            'This says who is working, not what the work is. GitHub already shows the author. Name the branch after the work.',
        },
        {
          title: 'fix',
          meta: 'git switch -c fix',
          flaw: 'fix',
          feedback:
            'It does not say what it fixes. Many branches in the repo could have this name.',
        },
        {
          title: 'fix/87-safari-export-button',
          meta: 'git switch -c fix/87-safari-export-button',
          correct: true,
          feedback:
            'It has the type, the issue number and a short description. Anyone can see what it is and find the issue.',
        },
        {
          title: 'my-changes-v2',
          meta: 'git switch -c my-changes-v2',
          flaw: 'my-changes-v2',
          feedback:
            'It tells nobody anything. Also, "v2" suggests that an old v1 branch exists too.',
        },
      ],
    },
    {
      id: 'git-best-commit',
      kind: 'cards',
      look: 'commit',
      prompt:
        'You fixed a bug: users were logged out when their token expired. Which commit do you want to see in `main`?',
      cards: [
        {
          title: 'fixed stuff',
          meta: 'a41c9e2 · 3 files changed',
          flaw: 'fixed stuff',
          feedback: 'In six months nobody will know what "stuff" means. Say what changed.',
        },
        {
          title: 'fix: retry login after token expiry',
          meta: '7d02b1f · 2 files changed',
          body: 'Users were logged out when the access token expired mid-session. Refresh the token once before giving up.\n\nFixes #42',
          correct: true,
          feedback:
            'A clear subject and a body that explains why. It also closes the issue when it reaches `main`.',
        },
        {
          title: 'fix: retry login after token expiry + new navbar + bump deps',
          meta: 'c93e77a · 41 files changed',
          flaw: '+ new navbar + bump deps',
          feedback:
            'Three unrelated changes in one commit. If the navbar breaks something, you cannot revert it without also losing the fix.',
        },
        {
          title: 'WIP',
          meta: '0be5d18 · 2 files changed',
          flaw: 'WIP',
          feedback:
            'This is fine on your own branch, but it should not reach `main`. Squash it or rename it first.',
        },
      ],
    },
    {
      id: 'git-update-branch',
      kind: 'fill',
      code: true,
      prompt:
        'Fill in the commands. They update your own feature branch with the latest `main` and push it.',
      template: 'git {{0}} origin\ngit {{1}} origin/main\ngit push {{2}}',
      answers: ['fetch', 'rebase', '--force-with-lease'],
      distractors: ['--force', 'reset', 'clone'],
      explanation:
        "First fetch to get the latest `main`. Then rebase onto it. Then push with `--force-with-lease`, because the history changed. Plain `--force` also works, but it can delete a teammate's push without a warning.",
    },
    {
      id: 'git-conflict-order',
      kind: 'fill',
      prompt: 'You are in the middle of a rebase and get a conflict. Put the steps in order.',
      template: '1. {{0}}\n2. {{1}}\n3. {{2}}\n4. {{3}}',
      answers: [
        'Run `git status` to find the files with conflicts',
        'Edit the files and remove the markers',
        'Stage them with `git add`',
        'Run `git rebase --continue`',
      ],
      distractors: ['Run `git commit -m "merge"`', 'Delete the branch and start again'],
      explanation:
        'During a rebase, Git makes the commit for you when you continue. You do not commit by hand. If you want to stop, `git rebase --abort` takes you back to where you started.',
    },
    {
      id: 'git-golden-rule',
      kind: 'single',
      scenario:
        'You and a teammate both push to `feat/payments`. You want a clean history. You run `git rebase origin/main` on it and force-push.',
      prompt: 'What happens next?',
      options: [
        'Nothing bad. A rebase only changes your local files.',
        'Your teammate has the old commits and the remote has new ones. Their next pull or push causes problems.',
        'GitHub deletes the branch.',
        'All the commits are squashed into one.',
      ],
      correct: [1],
      explanation:
        'The rebase made new commits with new SHAs. Your teammate still has the old ones. On a shared branch, merge `main` in instead.',
    },
    {
      id: 'git-revert',
      kind: 'single',
      scenario:
        "Yesterday's merge into `main` broke the build. Several people have pulled since then.",
      prompt: 'How do you undo it?',
      options: [
        '`git reset --hard HEAD~1` on `main` and force-push',
        '`git revert <sha>` and open a PR with the revert',
        'Delete the files by hand and commit',
        '`git commit --amend`',
      ],
      correct: [1],
      explanation:
        "Revert adds a commit that cancels the change, so nobody's history breaks. A reset and force-push on `main` breaks the history for everyone who pulled.",
    },
    {
      id: 'pr-which-closes',
      kind: 'cards',
      look: 'pull-request',
      scenario: 'Issue #87: "Export button does nothing on Safari". `main` is the default branch.',
      prompt: 'Which of these PRs closes #87 automatically when it is merged?',
      cards: [
        {
          title: safariTitle,
          meta: 'develop ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nCloses #87',
          flaw: 'develop',
          feedback:
            'The keyword is right, but the PR goes into `develop`. GitHub uses closing keywords only for PRs into the default branch.',
        },
        {
          title: safariTitle,
          meta: 'main ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nRelated to #87',
          flaw: 'Related to #87',
          feedback:
            '"Related to" only mentions the issue. You need a closing keyword, such as closes, fixes or resolves.',
        },
        {
          title: 'Fix Safari export and empty-state bugs',
          meta: 'main ← fix/export-bugs',
          body: 'Two small fixes in the export flow.\n\nCloses #86, #87',
          flaw: '#87',
          feedback:
            'Only #86 is closed. Each issue needs its own keyword: "Closes #86, closes #87".',
        },
        {
          title: safariTitle,
          meta: 'main ← fix/87-safari-export',
          body: 'Uses a Blob URL instead of a data URL.\n\nFixes #87',
          correct: true,
          feedback:
            'It goes into the default branch and has a closing keyword before the issue number. The merge closes #87.',
        },
      ],
    },
    {
      id: 'pr-multiple-issues',
      kind: 'fill',
      prompt: 'Complete the PR description so that a merge into `main` closes both issues.',
      template: '{{0}} #10, {{1}} #12',
      answers: ['Closes', 'closes'],
      distractors: ['and', 'See', 'Part of'],
      explanation:
        'Every issue needs its own closing keyword. "See" and "Part of" only mention the issue.',
    },
    {
      id: 'pr-where-keywords',
      kind: 'multiple',
      prompt: 'Where does `Fixes #42` close the issue?',
      options: [
        'In the PR description',
        'In a commit message that reaches the default branch',
        'In a comment on the PR',
        'In a Slack message about the PR',
      ],
      correct: [0, 1],
      explanation:
        'GitHub reads closing keywords in PR descriptions and in commit messages. In comments they are only a mention.',
    },
    {
      id: 'pr-review-ai-patch',
      kind: 'patch',
      scenario:
        'A teammate asked an AI assistant to "make getUser handle errors better". Then they opened a PR with the result. You are the reviewer.',
      prompt: 'Flag every line you would comment on.',
      file: 'src/api/users.ts',
      lines: [
        { type: 'context', text: 'export async function getUser(id: string) {' },
        { type: 'remove', text: "  const res = await fetch('/api/users/' + id);" },
        {
          type: 'add',
          text: "  const url = 'https://api.example.com/users/' + id + '?token=sk_live_9f8a7c';",
          issue:
            'A live API token is written directly in the source code. It belongs in an environment variable. This token must now be replaced with a new one.',
        },
        { type: 'add', text: '  const res = await fetch(url);' },
        {
          type: 'add',
          text: "  console.log('user response', res);",
          issue: 'Debug logging that was left in. It also writes user data into the logs.',
        },
        { type: 'context', text: '  if (!res.ok) {' },
        { type: 'remove', text: "    throw new Error('Failed to load user ' + id);" },
        {
          type: 'add',
          text: '    return null;',
          issue:
            'Errors are now hidden. Callers cannot tell "user not found" from "server is down". The error handling is now worse.',
        },
        { type: 'context', text: '  }' },
        { type: 'context', text: '  return res.json();' },
        { type: 'context', text: '}' },
      ],
    },
  ],
};
