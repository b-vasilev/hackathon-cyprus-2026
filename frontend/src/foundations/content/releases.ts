import type { FoundationModule } from '../types';

const NOT_ALLOWED = 'not allowed';
const OLD_VERSION = 'old version';
const PR_META = 'dependabot · 1 file changed';

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
      summary: 'What the three numbers in a version like 2.4.1 tell you.',
      blocks: [
        {
          type: 'p',
          text: 'A dependency is a package that your project uses, for example a library from npm. Most packages use semantic versioning (SemVer). A version has three numbers: MAJOR.MINOR.PATCH.',
        },
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
          type: 'scenes',
          title: 'Which part of the version changes',
          legend: [
            { tone: 'neutral', label: 'Unchanged' },
            { tone: 'brand', label: 'Goes up' },
            { tone: 'muted', label: 'Reset to 0' },
          ],
          scenes: [
            {
              name: 'Patch',
              summary: 'A bug fix raises the last number.',
              description: 'Nothing breaks for users. MAJOR and MINOR stay the same.',
              rows: [
                {
                  label: 'Before',
                  blocks: [{ text: '2' }, { text: '4' }, { text: '1' }],
                },
                {
                  label: 'After',
                  blocks: [
                    { text: '2' },
                    { text: '4' },
                    { text: '2', tone: 'brand', note: 'bug fixed' },
                  ],
                },
              ],
            },
            {
              name: 'Minor',
              summary: 'A new feature raises the middle number.',
              description: 'Nothing breaks for users. PATCH goes back to 0.',
              rows: [
                {
                  label: 'Before',
                  blocks: [{ text: '2' }, { text: '4' }, { text: '1' }],
                },
                {
                  label: 'After',
                  blocks: [
                    { text: '2' },
                    { text: '5', tone: 'brand', note: 'feature added' },
                    { text: '0', tone: 'muted', note: 'reset' },
                  ],
                },
              ],
            },
            {
              name: 'Major',
              summary: 'A breaking change raises the first number.',
              description: 'Users may need to change their code. MINOR and PATCH go back to 0.',
              rows: [
                {
                  label: 'Before',
                  blocks: [{ text: '2' }, { text: '4' }, { text: '1' }],
                },
                {
                  label: 'After',
                  blocks: [
                    { text: '3', tone: 'brand', note: 'breaking change' },
                    { text: '0', tone: 'muted', note: 'reset' },
                    { text: '0', tone: 'muted', note: 'reset' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'p',
          text: 'Versions below `1.0.0` are early versions. The API is not stable yet, and any release might break things. A pre-release, a test version before the real release, looks like `3.0.0-beta.1`.',
        },
        {
          type: 'p',
          text: 'In `package.json` you usually see ranges, not exact versions. A range says which versions are allowed:',
        },
        {
          type: 'table',
          columns: ['Range', 'Allows', 'In plain words'],
          rows: [
            ['`^1.2.3`', '`>=1.2.3 <2.0.0`', 'Minor and patch updates'],
            ['`~1.2.3`', '`>=1.2.3 <1.3.0`', 'Patch updates only'],
            ['`^0.2.3`', '`>=0.2.3 <0.3.0`', 'Below 1.0, the caret is stricter'],
            ['`1.2.3`', 'Exactly `1.2.3`', 'Pinned'],
          ],
        },
        {
          type: 'scenes',
          title: 'Which versions a range allows',
          legend: [
            { tone: 'brand', label: 'Allowed' },
            { tone: 'danger', label: 'Not allowed' },
          ],
          scenes: [
            {
              name: '`^1.2.3`',
              summary: 'The caret (`^`) allows minor and patch updates.',
              description: 'Anything from `1.2.3` up to, but not including, `2.0.0`.',
              rows: [
                {
                  label: 'Versions',
                  blocks: [
                    { text: '`1.2.3`', tone: 'brand' },
                    { text: '`1.2.9`', tone: 'brand' },
                    { text: '`1.7.0`', tone: 'brand' },
                    { text: '`2.0.0`', tone: 'danger', note: NOT_ALLOWED },
                  ],
                },
              ],
            },
            {
              name: '`~1.2.3`',
              summary: 'The tilde (`~`) allows patch updates only.',
              description: 'Anything from `1.2.3` up to, but not including, `1.3.0`.',
              rows: [
                {
                  label: 'Versions',
                  blocks: [
                    { text: '`1.2.3`', tone: 'brand' },
                    { text: '`1.2.9`', tone: 'brand' },
                    { text: '`1.3.0`', tone: 'danger', note: NOT_ALLOWED },
                    { text: '`2.0.0`', tone: 'danger', note: NOT_ALLOWED },
                  ],
                },
              ],
            },
            {
              name: '`^0.2.3`',
              summary: 'Below 1.0, the caret allows patch updates only.',
              description:
                'For `0.x` versions, a minor change may break things. So `0.3.0` is blocked.',
              rows: [
                {
                  label: 'Versions',
                  blocks: [
                    { text: '`0.2.3`', tone: 'brand' },
                    { text: '`0.2.8`', tone: 'brand' },
                    { text: '`0.3.0`', tone: 'danger', note: NOT_ALLOWED },
                    { text: '`1.0.0`', tone: 'danger', note: NOT_ALLOWED },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      id: 'lockfiles',
      title: 'Lockfiles and updates',
      summary:
        'Why two people who install the same project can get different code, and how a lockfile prevents it.',
      blocks: [
        {
          type: 'p',
          text: "`package.json` (or `pyproject.toml`) says which versions are allowed. The lockfile (`package-lock.json`, `poetry.lock`, Go's `go.sum`) records exactly which versions were installed, with checksums. A checksum is a short code that proves a file has not changed. With the lockfile, your laptop, your colleague's laptop and CI all install the same code.",
        },
        {
          type: 'list',
          items: [
            'Commit the lockfile.',
            'In CI use `npm ci`. It installs exactly what the lockfile says. It fails if the lockfile does not match `package.json`.',
            'Look at lockfile changes in PRs. One line in `package.json` can bring in dozens of updated packages.',
            'Let Dependabot or Renovate open small, regular update PRs. Let CI test each one.',
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
            'A changelog lists what changed in each version, for the people who have to upgrade.',
            'An environment is a separate copy of the running app. Changes usually go through environments (dev, then staging, then production), so they are checked before real users see them.',
            'Feature flags are on/off settings in the app. They let you deploy code switched off and turn it on later without another deploy.',
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'Merged is not released',
          text: 'A PR merged into `main` might wait there until the next release. When you tell someone a bug is fixed, say which version or environment has the fix.',
        },
        {
          type: 'scenes',
          title: 'Where the fix is',
          legend: [
            { tone: 'brand', label: 'Has the fix' },
            { tone: 'muted', label: 'Not yet' },
          ],
          scenes: [
            {
              name: 'Merged',
              summary: 'The fix is in `main`, but no environment runs it yet.',
              description: 'Users still see the bug.',
              rows: [
                {
                  label: 'Path',
                  blocks: [
                    { text: '`main`', tone: 'brand', note: 'fix merged' },
                    { text: 'Staging', tone: 'muted', note: OLD_VERSION },
                    { text: 'Production', tone: 'muted', note: OLD_VERSION },
                  ],
                },
              ],
            },
            {
              name: 'On staging',
              summary: 'The fix is deployed to staging for testing.',
              description: 'The team can check it there. Real users still see the bug.',
              rows: [
                {
                  label: 'Path',
                  blocks: [
                    { text: '`main`', tone: 'brand' },
                    { text: 'Staging', tone: 'brand', note: 'being tested' },
                    { text: 'Production', tone: 'muted', note: OLD_VERSION },
                  ],
                },
              ],
            },
            {
              name: 'Released',
              summary: 'The fix is in production, so users have it.',
              description: 'Now you can tell users the version, for example `v2.5.1`.',
              rows: [
                {
                  label: 'Path',
                  blocks: [
                    { text: '`main`', tone: 'brand' },
                    { text: 'Staging', tone: 'brand' },
                    { text: 'Production', tone: 'brand', note: '`v2.5.1`' },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'rel-bumps',
      kind: 'fill',
      prompt: 'The library is at `4.2.7`. What is the next version for each change?',
      template:
        'Removed a public function → {{0}}\nAdded an optional parameter → {{1}}\nFixed a crash on empty input → {{2}}',
      answers: ['5.0.0', '4.3.0', '4.2.8'],
      distractors: ['4.2.7-1', '4.3.1'],
      explanation:
        'A breaking change raises MAJOR. A new compatible feature raises MINOR. A compatible fix raises PATCH. The numbers to the right go back to zero.',
    },
    {
      id: 'rel-caret',
      kind: 'single',
      prompt:
        'A dependency is listed as `^1.4.0`. Which version could you get after you regenerate the lockfile?',
      options: ['`2.0.0`', '`1.9.2`', '`1.3.9`', '`0.9.0`'],
      correct: [1],
      explanation: 'The caret allows any `1.x.y` from `1.4.0` up, but not `2.0.0`.',
    },
    {
      id: 'rel-zero-caret',
      kind: 'single',
      prompt: 'A dependency is listed as `^0.2.3`. Which of these versions is allowed?',
      options: ['`0.3.0`', '`0.2.9`', '`1.0.0`', '`0.2.2`'],
      correct: [1],
      explanation:
        'Below 1.0, the caret allows `>=0.2.3 <0.3.0`. Only patch updates like `0.2.9` are allowed.',
    },
    {
      id: 'rel-tilde-cards',
      kind: 'cards',
      look: 'pull-request',
      prompt:
        'The project allows `~2.3.0` for `chart-lib`. Which update PR stays inside this range?',
      cards: [
        {
          title: 'Bump chart-lib from 2.3.1 to 3.0.0',
          meta: PR_META,
          flaw: '3.0.0',
          feedback: '`3.0.0` is a new MAJOR version. The tilde allows patch updates only.',
        },
        {
          title: 'Bump chart-lib from 2.3.1 to 2.4.0',
          meta: PR_META,
          flaw: '2.4.0',
          feedback: '`2.4.0` is a new MINOR version. `~2.3.0` allows only `>=2.3.0 <2.4.0`.',
        },
        {
          title: 'Bump chart-lib from 2.3.1 to 2.3.4',
          meta: PR_META,
          correct: true,
          feedback: 'Correct. `2.3.4` is a patch update, so the tilde allows it.',
        },
      ],
    },
    {
      id: 'rel-npm-ci',
      kind: 'single',
      scenario:
        'CI runs `npm install`. Sometimes the build is different from what developers have on their laptops.',
      prompt: 'What makes CI installs reproducible (the same every time)?',
      options: [
        'Delete `package-lock.json`',
        'Use `npm ci` with a committed lockfile',
        'Put `^` in front of every version',
        'Run `npm update` before each build',
      ],
      correct: [1],
      explanation:
        '`npm ci` installs exactly what is in the lockfile. It fails if the lockfile and `package.json` do not match.',
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
      explanation:
        'Merged does not mean deployed. Tell them which version or environment has the fix.',
    },
  ],
};
