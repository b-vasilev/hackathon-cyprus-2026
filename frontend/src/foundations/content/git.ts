import type { FoundationModule } from '../types';

export const git: FoundationModule = {
  id: 'git',
  title: 'Git: branches, history and rebase',
  summary: 'Commits, branches, and when to merge, rebase or squash.',
  outcome:
    'Keep your branch up to date with main and get yourself out of trouble when something goes wrong.',
  minutes: 25,
  lessons: [
    {
      id: 'commits-and-branches',
      title: 'Commits and branches',
      summary: 'What Git actually stores, and how to start a piece of work.',
      blocks: [
        {
          type: 'p',
          text: 'Every commit is a full snapshot of the project, plus a message and a link to the commit before it (its parent). The commit ID, the SHA, is calculated from all of that. Change anything, even the parent, and you get a different SHA. Keep that in mind; it explains most of what follows.',
        },
        {
          type: 'p',
          text: 'A branch is just a name pointing at a commit. When you commit, the name moves forward to the new commit. `HEAD` means "where I am right now".',
        },
        {
          type: 'p',
          text: "The usual routine: get the latest `main`, make a branch for your change, commit, push. Don't commit straight to `main` in a shared repo. Everything goes through a pull request.",
        },
        {
          type: 'code',
          code: 'git switch main\ngit pull\ngit switch -c feat/42-export-csv\n# ...work...\ngit add -p        # pick what goes in, chunk by chunk\ngit commit\ngit push -u origin feat/42-export-csv',
          caption:
            '`git add -p` is worth getting used to. You see every change before it goes into the commit, which catches stray debug lines.',
        },
        {
          type: 'p',
          text: 'Name the branch after the work, not after yourself. A good name tells anyone looking at the branch list what it is and which issue it belongs to, without opening it. A common pattern is `type/issue-number-short-description`, in lowercase with hyphens.',
        },
        {
          type: 'table',
          columns: ['Branch name', 'Verdict'],
          rows: [
            ['`fix/87-safari-export-button`', 'Good. Type, issue number, and what it fixes.'],
            ['`feat/42-export-csv`', 'Good. Anyone can find issue #42 from it.'],
            ['`tatiana`', "Says who, not what. You'll have five pieces of work and one name."],
            ['`fix`', 'Fix what? Every second branch would be called this.'],
            ['`my-branch`, `test2`, `new`', 'Means nothing to anyone, including you next month.'],
            [
              '`Fix_Safari Export!!`',
              'Spaces, capitals and symbols cause trouble in scripts and URLs.',
            ],
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Your team may have a convention',
          text: "Some teams use `feature/` instead of `feat/`, or put the ticket key first, like `PROJ-87-safari-export`. Check the repo's existing branches and follow what's there.",
        },
        {
          type: 'p',
          text: 'Commit messages: write the subject as an instruction ("Add CSV export", not "added stuff") and use the body to say why. Plenty of teams also put a type prefix in front, a format called Conventional Commits:',
        },
        {
          type: 'table',
          columns: ['Prefix', 'For', 'Example'],
          rows: [
            ['`feat:`', 'Something new for users', '`feat: add CSV export to reports`'],
            ['`fix:`', 'A bug fix', '`fix: retry login after token expiry`'],
            ['`docs:`', 'Docs only', '`docs: describe local setup`'],
            ['`refactor:`', 'Restructuring, same behavior', '`refactor: extract price calculator`'],
            ['`test:`', 'Tests', '`test: cover empty cart checkout`'],
            ['`ci:` / `chore:`', 'Pipeline, tooling, housekeeping', '`ci: cache npm dependencies`'],
          ],
        },
      ],
    },
    {
      id: 'merge-rebase-squash',
      title: 'Merge, rebase and squash',
      summary: 'Bringing main into your branch, and your branch into main.',
      blocks: [
        {
          type: 'p',
          text: "While you're working, other people merge into `main`. At some point you need their changes in your branch, and eventually your branch has to land in `main`. There are three ways to combine history.",
        },
        { type: 'git-history' },
        {
          type: 'table',
          columns: ['', 'What it does', 'History afterwards', 'When to use it'],
          rows: [
            [
              'Merge',
              'Adds a merge commit with two parents. Nothing existing is touched.',
              'Shows exactly what happened, merge commits included.',
              'Bringing `main` into a branch other people also use.',
            ],
            [
              'Rebase',
              'Takes your commits and replays them on top of the latest `main`. Each one gets a new SHA.',
              "A straight line, as if you'd started from today's `main`.",
              'Tidying your own branch before review.',
            ],
            [
              'Squash',
              'Turns the whole branch into one commit.',
              'One commit per PR on `main`.',
              'Landing a PR full of "wip" and "fix typo" commits.',
            ],
          ],
        },
        {
          type: 'code',
          code: 'git fetch origin\ngit rebase origin/main\ngit push --force-with-lease',
          caption:
            'Updating your own branch. After a rebase the remote rejects a normal push, because the history changed.',
        },
        {
          type: 'callout',
          tone: 'warning',
          title: "Only rebase what's yours",
          text: 'Rebasing makes new commits. If a teammate already has the old ones (a shared feature branch, or `main` itself), their copy and yours no longer match and someone ends up untangling it. For shared branches, merge.',
        },
        {
          type: 'p',
          text: 'Get into the habit of `--force-with-lease` instead of `--force`. If someone pushed to the branch since you last fetched, it refuses instead of quietly wiping out their commits.',
        },
        {
          type: 'p',
          text: 'Before asking for review you can clean up with `git rebase -i origin/main`: reorder commits, fix messages, fold small fixes into the commit they belong to (`fixup`).',
        },
        {
          type: 'p',
          text: 'GitHub\'s merge button has the same three choices: "Create a merge commit", "Squash and merge" and "Rebase and merge". Repo settings control which are allowed. Note that GitHub\'s "Rebase and merge" always creates new SHAs, even when it could have fast-forwarded.',
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
          text: "You get a conflict when both sides changed the same lines, or one side edited a file the other deleted. It's routine. Git can't guess which version is right, so it stops and marks the file:",
        },
        {
          type: 'code',
          code: '<<<<<<< HEAD\nconst timeout = 30;\n=======\nconst timeout = 60;\n>>>>>>> feat/slow-network',
          caption:
            'The two competing versions sit between the markers. Delete the markers and leave the code you actually want.',
        },
        {
          type: 'list',
          ordered: true,
          items: [
            '`git status` shows which files are conflicted.',
            'Fix each file and remove every marker. Often the right result is a mix of both sides.',
            '`git add` the fixed files.',
            '`git merge --continue` or `git rebase --continue`.',
            'Run the tests. Code without conflict markers can still be broken.',
          ],
        },
        {
          type: 'p',
          text: 'Changed your mind? `git merge --abort` or `git rebase --abort` puts everything back the way it was.',
        },
        {
          type: 'callout',
          tone: 'tip',
          title: '"Ours" and "theirs" flip during a rebase',
          text: 'In a merge, "ours" is your branch. In a rebase, Git is replaying your commits onto `main`, so "ours" is `main` and "theirs" is you. Most people get caught by this once. Look at the code, not the labels.',
        },
      ],
    },
    {
      id: 'undo',
      title: 'Undoing things',
      summary: 'Which undo to use depends on whether anyone else has the commit yet.',
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
              "Yes (fine, it's local)",
            ],
            [
              "Undo a commit that's already on `main`",
              '`git revert <sha>`',
              'No, adds a new commit',
            ],
            ['Move your local branch back', '`git reset --hard <sha>`', 'Yes'],
            ['Park your changes for a minute', '`git stash`, later `git stash pop`', 'No'],
          ],
        },
        {
          type: 'p',
          text: "If the commit is already shared, use `git revert`. It adds a commit that does the opposite, and nobody else's history changes.",
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Lost a commit? Check the reflog',
          text: '`git reflog` lists everywhere `HEAD` has been recently, including commits no branch points to anymore. After a bad reset or rebase, find the SHA there and run `git switch -c rescue <sha>`.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'git-branch-name',
      kind: 'cards',
      look: 'branch',
      scenario: 'Issue #87: "Export button does nothing on Safari". It\'s assigned to you.',
      prompt: 'Which branch name would you create?',
      cards: [
        {
          title: 'tatiana',
          meta: 'git switch -c tatiana',
          flaw: 'tatiana',
          feedback:
            'That says who is working, not what. GitHub already shows the author. Name it after the work.',
        },
        {
          title: 'fix',
          meta: 'git switch -c fix',
          flaw: 'fix',
          feedback: 'Fix what? Half the branches in the repo could be called this.',
        },
        {
          title: 'fix/87-safari-export-button',
          meta: 'git switch -c fix/87-safari-export-button',
          correct: true,
          feedback:
            'Type, issue number and a short description. Anyone can tell what it is and find the issue from it.',
        },
        {
          title: 'my-changes-v2',
          meta: 'git switch -c my-changes-v2',
          flaw: 'my-changes-v2',
          feedback: 'Tells nobody anything, and "v2" suggests there\'s a v1 lying around too.',
        },
      ],
    },
    {
      id: 'git-best-commit',
      kind: 'cards',
      look: 'commit',
      prompt:
        'You fixed a bug where users were logged out when their token expired. Which commit would you be happy to see in `main`?',
      cards: [
        {
          title: 'fixed stuff',
          meta: 'a41c9e2 · 3 files changed',
          flaw: 'fixed stuff',
          feedback: 'Six months from now nobody will know what "stuff" was. Say what changed.',
        },
        {
          title: 'fix: retry login after token expiry',
          meta: '7d02b1f · 2 files changed',
          body: 'Users were logged out when the access token expired mid-session. Refresh the token once before giving up.\n\nFixes #42',
          correct: true,
          feedback:
            'Clear subject, a body that says why, and it closes the issue when it lands on `main`.',
        },
        {
          title: 'fix: retry login after token expiry + new navbar + bump deps',
          meta: 'c93e77a · 41 files changed',
          flaw: '+ new navbar + bump deps',
          feedback:
            "Three unrelated changes in one commit. If the navbar breaks something, you can't revert it without losing the fix.",
        },
        {
          title: 'WIP',
          meta: '0be5d18 · 2 files changed',
          flaw: 'WIP',
          feedback:
            "Fine on your own branch, but it shouldn't reach `main`. Squash or reword it first.",
        },
      ],
    },
    {
      id: 'git-update-branch',
      kind: 'fill',
      code: true,
      prompt:
        'Fill in the commands to update your own feature branch with the latest `main` and push it.',
      template: 'git {{0}} origin\ngit {{1}} origin/main\ngit push {{2}}',
      answers: ['fetch', 'rebase', '--force-with-lease'],
      distractors: ['--force', 'reset', 'clone'],
      explanation:
        "Fetch first so you have the latest `main`, rebase onto it, then push with `--force-with-lease` because the history changed. Plain `--force` would also work, but it can silently delete a teammate's push.",
    },
    {
      id: 'git-golden-rule',
      kind: 'single',
      scenario:
        'You and a teammate both push to `feat/payments`. You want tidy history, so you run `git rebase origin/main` on it and force-push.',
      prompt: "What's going to happen?",
      options: [
        'Nothing bad. Rebase only touches your local files.',
        "Your teammate's copy now has the old commits and the remote has new ones, so their next pull or push gets messy.",
        'GitHub deletes the branch.',
        'All the commits get squashed into one.',
      ],
      correct: [1],
      explanation:
        'Rebase made new commits with new SHAs. Your teammate still has the old ones. On a shared branch, merge `main` in instead.',
    },
    {
      id: 'git-conflict-order',
      kind: 'fill',
      prompt: "You're in the middle of a rebase and hit a conflict. Put the steps in order.",
      template: '1. {{0}}\n2. {{1}}\n3. {{2}}\n4. {{3}}',
      answers: [
        'Run `git status` to find the conflicted files',
        'Edit the files and remove the markers',
        'Stage them with `git add`',
        'Run `git rebase --continue`',
      ],
      distractors: ['Run `git commit -m "merge"`', 'Delete the branch and start again'],
      explanation:
        "During a rebase Git makes the commit for you when you continue, so there's no manual commit. And if it all goes sideways, `git rebase --abort` gets you back to where you started.",
    },
    {
      id: 'git-revert',
      kind: 'single',
      scenario: "Yesterday's merge into `main` broke the build. Several people have pulled since.",
      prompt: 'How do you undo it?',
      options: [
        '`git reset --hard HEAD~1` on `main` and force-push',
        '`git revert <sha>` and open a PR with the revert',
        'Delete the files by hand and commit',
        '`git commit --amend`',
      ],
      correct: [1],
      explanation:
        "Revert adds a commit that cancels the change, so nobody's history breaks. Resetting `main` and force-pushing breaks it for everyone who pulled.",
    },
    {
      id: 'git-reflog',
      kind: 'single',
      prompt: 'You ran `git reset --hard` and lost a commit you never pushed. Where do you look?',
      options: ["Nowhere, it's gone", '`git reflog`', '`git status`', 'The branch page on GitHub'],
      correct: [1],
      explanation:
        "The reflog keeps track of where `HEAD` has been, so the SHA is still there for a while. Make a branch from it and you're back.",
    },
  ],
};
