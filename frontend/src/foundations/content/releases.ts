import type { FoundationModule } from '../types';

export const releases: FoundationModule = {
  id: 'releases',
  title: 'Dependencies, versions and releases',
  summary: 'Version numbers, lockfiles, and how a merged change reaches users.',
  outcome: 'Update a dependency without surprises and know what a version number promises.',
  minutes: 15,
  lessons: [
    {
      id: 'semver',
      title: 'Semantic versioning',
      summary: 'What the three numbers in 2.4.1 tell you.',
      blocks: [
        {
          type: 'table',
          columns: ['Part', 'Goes up when', 'Example'],
          rows: [
            ['MAJOR', 'Something breaks for existing users', '`2.4.1` → `3.0.0`'],
            ['MINOR', 'Something new is added, nothing breaks', '`2.4.1` → `2.5.0`'],
            ['PATCH', 'A bug is fixed, nothing breaks', '`2.4.1` → `2.4.2`'],
          ],
        },
        {
          type: 'p',
          text: 'Anything below `1.0.0` is "still figuring it out", and any release might break things. Pre-releases look like `3.0.0-beta.1`.',
        },
        {
          type: 'p',
          text: "In `package.json` you'll see ranges rather than exact versions:",
        },
        {
          type: 'table',
          columns: ['Range', 'Allows', 'In plain words'],
          rows: [
            ['`^1.2.3`', '`>=1.2.3 <2.0.0`', 'Minor and patch updates'],
            ['`~1.2.3`', '`>=1.2.3 <1.3.0`', 'Patch updates only'],
            ['`^0.2.3`', '`>=0.2.3 <0.3.0`', 'Under 1.0, the caret is stricter'],
            ['`1.2.3`', 'Exactly `1.2.3`', 'Pinned'],
          ],
        },
      ],
    },
    {
      id: 'lockfiles',
      title: 'Lockfiles and updates',
      summary: 'Why two people installing "the same" project can get different code.',
      blocks: [
        {
          type: 'p',
          text: "`package.json` (or `pyproject.toml`) says which versions are allowed. The lockfile (`package-lock.json`, `poetry.lock`, Go's `go.sum`) records exactly which versions were installed, with checksums. That's how your laptop, your colleague's laptop and CI all end up with the same code.",
        },
        {
          type: 'list',
          items: [
            'Commit the lockfile.',
            "In CI use `npm ci`. It installs exactly what the lockfile says and fails if it doesn't match `package.json`.",
            'Look at lockfile changes in PRs. One line in `package.json` can pull in dozens of updated packages.',
            'Let Dependabot or Renovate open small, regular update PRs, and let CI test each one.',
            'Read the changelog before a major upgrade.',
          ],
        },
      ],
    },
    {
      id: 'releasing',
      title: 'Tags, releases and environments',
      summary: 'What happens between "merged" and "users have it".',
      blocks: [
        {
          type: 'list',
          items: [
            'A Git tag like `v2.5.0` marks the exact commit that was released.',
            'A GitHub release adds notes and files to that tag.',
            'A changelog lists what changed in each version, for whoever has to upgrade.',
            'Changes usually go through environments (dev, then staging, then production), so they get checked before real users see them.',
            'Feature flags let you ship code switched off and turn it on later without another deploy.',
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: "Merged isn't released",
          text: 'A PR merged into `main` might sit there until the next release. When you tell someone a bug is fixed, say which version or environment has the fix.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'rel-bumps',
      kind: 'fill',
      prompt: "The library is at `4.2.7`. What's the next version for each change?",
      template:
        'Removed a public function → {{0}}\nAdded an optional parameter → {{1}}\nFixed a crash on empty input → {{2}}',
      answers: ['5.0.0', '4.3.0', '4.2.8'],
      distractors: ['4.2.7-1', '4.3.1'],
      explanation:
        'Breaking change bumps MAJOR, new compatible feature bumps MINOR, compatible fix bumps PATCH. The numbers to the right reset to zero.',
    },
    {
      id: 'rel-caret',
      kind: 'single',
      prompt:
        'A dependency is listed as `^1.4.0`. Which version could you get after regenerating the lockfile?',
      options: ['`2.0.0`', '`1.9.2`', '`1.3.9`', '`0.9.0`'],
      correct: [1],
      explanation: 'The caret allows anything `1.x.y` from `1.4.0` up, but not `2.0.0`.',
    },
    {
      id: 'rel-npm-ci',
      kind: 'single',
      scenario:
        'CI runs `npm install`, and every so often the build differs from what developers have locally.',
      prompt: 'What makes CI installs reproducible?',
      options: [
        'Delete `package-lock.json`',
        'Use `npm ci` with a committed lockfile',
        'Put `^` in front of every version',
        'Run `npm update` before each build',
      ],
      correct: [1],
      explanation:
        "`npm ci` installs exactly what's in the lockfile and fails if the lockfile and `package.json` disagree.",
    },
    {
      id: 'rel-merged-released',
      kind: 'single',
      scenario:
        'A customer asks if bug #301 is fixed. The fix was merged into `main` this morning.',
      prompt: 'What do you tell them?',
      options: [
        '"Yes, it\'s live."',
        '"The fix is merged and goes out with the next release. I\'ll confirm the version once it\'s deployed."',
        '"No."',
        '"Check GitHub."',
      ],
      correct: [1],
      explanation: "Merged doesn't mean deployed. Tell them which version or environment has it.",
    },
  ],
};
