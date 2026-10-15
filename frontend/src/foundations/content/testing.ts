import type { FoundationModule, SceneBlock, Tone } from '../types';

// A row of identical test blocks for the pyramid picture.
const tests = (count: number, text: string, tone: Tone): SceneBlock[] =>
  Array.from({ length: count }, () => ({ text, tone }));

// Run history for the CI runs picture: '.' is a passed run, 'x' a failed one.
const history = (pattern: string): ('pass' | 'fail')[] =>
  Array.from(pattern, (mark) => (mark === 'x' ? 'fail' : 'pass'));
const ALL_PASS = history('..........');

// Repeated diagram labels.
const SYSTEM = 'System';
const E2E = 'End-to-end';
const UI = 'UI';
const API = 'API';
const SERVICE = 'Service';
const DATABASE = 'Database';
const PAYMENT = 'Payment provider';
const NOT_USED = 'not used';
const TESTED = 'tested';
const CHECKOUT_SPEC = '`checkout.spec.ts`';
const CART_SPEC = '`cart.spec.ts`';
const LOGIN_SPEC = '`login.spec.ts`';
const REFACTOR = 'Refactor';
const TEST = 'Test';

export const testing: FoundationModule = {
  id: 'testing',
  title: 'Automated testing basics',
  summary: 'Kinds of tests, what makes a good test, and why some tests are flaky.',
  outcome: 'Pick the right kind of test for a change, and recognise a flaky test.',
  minutes: 15,
  lessons: [
    {
      id: 'test-levels',
      title: 'Unit, integration and end-to-end',
      summary: 'The three levels of tests, and what each one costs.',
      blocks: [
        {
          type: 'p',
          text: 'An automated test is code that runs your code and checks the result. Tests come in three levels. The level says how much of the system one test uses.',
        },
        {
          type: 'table',
          columns: ['Level', 'Tests', 'Speed', 'Breaks when'],
          rows: [
            [
              'Unit',
              'One function or class on its own',
              'Milliseconds',
              "That code's logic changes",
            ],
            [
              'Integration',
              'A few parts together, like your code and a real database',
              'Seconds',
              'The contract (the agreed inputs and outputs) between them changes',
            ],
            [
              E2E,
              'The whole app through the UI or public API, like a user',
              'Seconds to minutes',
              'Almost anything changes',
            ],
          ],
        },
        {
          type: 'p',
          text: 'Parts that a test does not use are often replaced. A mock is a small piece of fake code that takes the place of a real part, such as the database. The test tells the mock what to answer. So the test is fast, and it does not depend on that real part.',
        },
        {
          type: 'scenes',
          title: 'Which parts each test level uses',
          legend: [
            { tone: 'brand', label: 'Real part, tested' },
            { tone: 'soft', label: 'Replaced by a mock or test version' },
            { tone: 'muted', label: 'Not used' },
          ],
          scenes: [
            {
              name: 'Unit',
              summary: 'A unit test checks one part on its own.',
              description:
                'The parts it talks to are mocks. If the test fails, the problem is in that one part.',
              rows: [
                {
                  label: SYSTEM,
                  blocks: [
                    { text: UI, tone: 'muted', note: NOT_USED },
                    { text: API, tone: 'muted', note: NOT_USED },
                    { text: SERVICE, tone: 'brand', note: TESTED },
                    { text: DATABASE, tone: 'soft', note: 'mock' },
                    { text: PAYMENT, tone: 'soft', note: 'mock' },
                  ],
                },
              ],
            },
            {
              name: 'Integration',
              summary: 'An integration test checks a few real parts together.',
              description:
                'Here the API, the service and a real database work together. The outside payment provider is still replaced.',
              rows: [
                {
                  label: SYSTEM,
                  blocks: [
                    { text: UI, tone: 'muted', note: NOT_USED },
                    { text: API, tone: 'brand', note: TESTED },
                    { text: SERVICE, tone: 'brand', note: TESTED },
                    { text: DATABASE, tone: 'brand', note: 'real database' },
                    { text: PAYMENT, tone: 'soft', note: 'mock' },
                  ],
                },
              ],
            },
            {
              name: E2E,
              summary: 'An end-to-end test uses the whole app, like a user does.',
              description:
                'It starts in the UI and goes through every part. Teams often use a test version of outside services, such as payments.',
              rows: [
                {
                  label: SYSTEM,
                  blocks: [
                    { text: UI, tone: 'brand', note: TESTED },
                    { text: API, tone: 'brand', note: TESTED },
                    { text: SERVICE, tone: 'brand', note: TESTED },
                    { text: DATABASE, tone: 'brand', note: TESTED },
                    { text: PAYMENT, tone: 'soft', note: 'test version' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'p',
          text: 'The usual advice is the test pyramid. Write many unit tests, fewer integration tests, and only a few end-to-end tests. Use end-to-end tests for the user journeys that matter most, such as sign up, pay and export. A higher level proves more in each test. But it is slower, and when it fails it is harder to see what went wrong.',
        },
        {
          type: 'scenes',
          title: 'Same number of tests, different mix',
          layout: 'stack',
          legend: [
            { tone: 'warning', label: 'Slow: end-to-end' },
            { tone: 'soft', label: 'Medium: integration' },
            { tone: 'brand', label: 'Fast: unit' },
          ],
          scenes: [
            {
              name: 'Good: pyramid',
              summary:
                '14 tests. Most of them are fast unit tests, and only 2 are slow end-to-end tests.',
              description:
                'The whole test run is fast. When a test fails, it points to one small part of the code, so the problem is easy to find.',
              rows: [
                { label: E2E, blocks: tests(2, 'E2E', 'warning') },
                { label: 'Integration', blocks: tests(4, 'Int.', 'soft') },
                { label: 'Unit', blocks: tests(8, 'Unit', 'brand') },
              ],
            },
            {
              name: 'Bad: upside down',
              summary:
                'Also 14 tests. But now 8 of them are slow end-to-end tests, and only 2 are fast unit tests.',
              description:
                'Each end-to-end test goes through the whole app, so the test run takes much longer. When one fails, many parts could be the cause.',
              rows: [
                { label: E2E, blocks: tests(8, 'E2E', 'warning') },
                { label: 'Integration', blocks: tests(4, 'Int.', 'soft') },
                { label: 'Unit', blocks: tests(2, 'Unit', 'brand') },
              ],
            },
          ],
        },
      ],
    },
    {
      id: 'good-tests',
      title: 'What makes a good test',
      summary: 'How to write a test that is easy to read, reliable and focused on behavior.',
      blocks: [
        {
          type: 'code',
          code: "test('gives 10% off orders over 100', () => {\n  // Arrange\n  const order = { total: 200 };\n  // Act\n  const result = applyDiscount(order);\n  // Assert\n  expect(result.total).toBe(180);\n});",
          caption:
            'Arrange, act, assert: set up the data, do one thing, check the result. The name says which behavior the test protects, so a failure explains itself.',
        },
        {
          type: 'list',
          items: [
            'Test what the code does, through its public interface (the functions other code calls). Do not test how it works inside. A refactor (a change to the inside that keeps the behavior the same) should not break tests.',
            'Fixing a bug? First write a test that fails because of the bug. Then fix the bug and see the test pass.',
            'Coverage tells you which lines ran during the tests. It does not tell you whether the tests checked the results. 100% coverage with weak assertions (checks) tells you very little.',
          ],
        },
        {
          type: 'scenes',
          title: 'What happens to tests after a refactor',
          legend: [
            { tone: 'neutral', label: 'Unchanged' },
            { tone: 'soft', label: 'Rewritten inside' },
            { tone: 'strong', label: 'Result' },
            { tone: 'brand', label: 'Test passes' },
            { tone: 'danger', label: 'Test fails' },
          ],
          scenes: [
            {
              name: 'Tests behavior',
              summary: 'The test checks the result, so it still passes.',
              description: 'Nothing changed for the caller, and the test agrees.',
              rows: [
                {
                  label: REFACTOR,
                  blocks: [
                    { text: '`total: 200`', note: 'input' },
                    { text: '`applyDiscount`', tone: 'soft', note: 'inside rewritten' },
                    { text: '`total: 180`', tone: 'strong', note: 'same output' },
                  ],
                },
                {
                  label: TEST,
                  blocks: [{ text: 'Checks the output', tone: 'brand', note: 'still passes' }],
                },
              ],
            },
            {
              name: 'Tests internals',
              summary: 'The test checks private details, so it fails for no reason.',
              description:
                'The output is still correct. You now spend time fixing tests instead of code.',
              rows: [
                {
                  label: REFACTOR,
                  blocks: [
                    { text: '`total: 200`', note: 'input' },
                    { text: '`applyDiscount`', tone: 'soft', note: 'inside rewritten' },
                    { text: '`total: 180`', tone: 'strong', note: 'same output' },
                  ],
                },
                {
                  label: TEST,
                  blocks: [
                    { text: 'Checks private call', tone: 'danger', note: 'fails, nothing broke' },
                  ],
                },
              ],
            },
          ],
        },
      ],
    },
    {
      id: 'flaky-tests',
      title: 'Flaky tests',
      summary: 'Why some tests pass and fail with no code change, and what to do.',
      blocks: [
        {
          type: 'p',
          text: 'A flaky test is a test that sometimes passes and sometimes fails, with no change to the code. These are the common causes:',
        },
        {
          type: 'list',
          items: [
            "Time: the test depends on today's date, the time zone, or a fixed wait like `sleep(2000)`.",
            'Order: tests share state (data that one test leaves for the next), so they only pass in a certain order.',
            'Network: the test calls a real external service.',
            'Concurrency: the test does not wait correctly for async work (work that finishes later) to finish.',
            'Randomness: the test uses random data without a fixed seed (the start value that makes random data repeat).',
          ],
        },
        {
          type: 'runs',
          title: 'Ten CI runs of each test',
          scenes: [
            {
              name: 'Stable test',
              summary: 'Same code, same result, every time.',
              rows: [
                { label: CHECKOUT_SPEC, runs: ALL_PASS },
                { label: CART_SPEC, runs: ALL_PASS },
                { label: LOGIN_SPEC, runs: ALL_PASS },
              ],
            },
            {
              name: 'Flaky test',
              summary: '`checkout.spec.ts` fails in runs 3 and 8, with no code change.',
              description:
                'The cause is outside your change: time, test order, network, async work or random data.',
              rows: [
                { label: CHECKOUT_SPEC, runs: history('..x....x..') },
                { label: CART_SPEC, runs: ALL_PASS },
                { label: LOGIN_SPEC, runs: ALL_PASS },
              ],
            },
            {
              name: 'Real failure',
              summary: '`cart.spec.ts` passes in runs 1 to 5, then fails in every run after that.',
              description:
                'A code change merged before run 6 broke it. This is a real bug, so fix the code.',
              rows: [
                { label: CHECKOUT_SPEC, runs: ALL_PASS },
                { label: CART_SPEC, runs: history('.....xxxxx') },
                { label: LOGIN_SPEC, runs: ALL_PASS },
              ],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'A flaky test is a bug',
          text: 'If you re-run it until it passes, people learn to ignore failed builds. Then a real failure is merged without anyone noticing. Report the flaky test. Quarantine it (move it out of the required checks for a short time) if you must, and fix the cause.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'test-aaa',
      kind: 'fill',
      code: true,
      prompt: 'Label the three parts of this test.',
      template:
        "test('rejects an expired coupon', () => {\n  // {{0}}\n  const coupon = { code: 'SPRING', expires: '2020-01-01' };\n  // {{1}}\n  const result = applyCoupon(cart, coupon);\n  // {{2}}\n  expect(result.error).toBe('Coupon expired');\n});",
      answers: ['Arrange', 'Act', 'Assert'],
      distractors: ['Mock', 'Retry'],
      explanation: 'Set up the data, do the one thing you are testing, then check the result.',
    },
    {
      id: 'test-level-choice',
      kind: 'single',
      scenario:
        'You fixed a rounding bug in `calculateTax(amount)`. It is a pure function: it only uses its input and changes nothing else.',
      prompt: 'What is the most efficient way to make sure it stays fixed?',
      options: [
        'A unit test for `calculateTax`',
        'A full end-to-end checkout test',
        'A manual test plan',
        'No test needed',
      ],
      correct: [0],
      explanation:
        'A pure function is ideal for a quick unit test. Make sure the test fails without your fix and passes with it.',
    },
    {
      id: 'test-mock',
      kind: 'single',
      scenario: 'Your unit test for `createOrder` uses a mock in place of the payment provider.',
      prompt: 'What does the mock do?',
      options: [
        'It sends real payments in a safe mode',
        'It replaces the real provider and returns answers the test chooses',
        'It checks the code style of the payment code',
        'It makes the test run in CI only',
      ],
      correct: [1],
      explanation:
        'A mock is fake code that stands in for a real part. The test decides what it answers, so the test is fast and does not depend on the real service.',
    },
    {
      id: 'test-flaky-causes',
      kind: 'multiple',
      prompt: 'Which of these often make tests flaky?',
      options: [
        'Depending on the current time',
        'Sharing state between tests',
        'Calling a real third-party API',
        'Descriptive test names',
      ],
      correct: [0, 1, 2],
      explanation: 'All three make the result depend on something other than your code.',
    },
    {
      id: 'test-flaky-patch',
      kind: 'patch',
      prompt: 'Flag the lines that can make this test flaky.',
      file: 'tests/report.test.ts',
      lines: [
        { type: 'add', text: "test('shows the report for today', async () => {" },
        {
          type: 'add',
          text: '  const today = new Date();',
          issue: "The result depends on today's date and time zone. Use a fixed date.",
        },
        {
          type: 'add',
          text: "  const rates = await fetch('https://api.exchange.example/rates');",
          issue: 'The test calls a real external service over the network. Use a mock.',
        },
        { type: 'add', text: '  render(<Report date={today} rates={rates} />);' },
        {
          type: 'add',
          text: '  await sleep(2000);',
          issue: 'A fixed wait. Sometimes the work takes longer. Wait for the result itself.',
        },
        { type: 'add', text: "  expect(screen.getByText('Total')).toBeVisible();" },
        { type: 'add', text: '});' },
      ],
    },
    {
      id: 'test-refactor',
      kind: 'single',
      scenario:
        'You refactor a class without changing what it does. 20 tests fail, because they check which private methods were called.',
      prompt: 'What does that tell you?',
      options: [
        'The refactor is wrong',
        'The tests check implementation details instead of behavior',
        "Don't refactor",
        'Delete all the tests',
      ],
      correct: [1],
      explanation:
        'Tests that check the inside of the code break on harmless changes. Check the output instead.',
    },
    {
      id: 'test-flaky-branch',
      kind: 'branch',
      prompt: 'What do you do? Each decision changes what happens next.',
      steps: [
        {
          id: 'start',
          text: 'Your PR changes only a README file. One test in CI fails. Yesterday the same test passed on `main`.',
          choices: [
            {
              label: 'Re-run CI until it passes and merge',
              good: false,
              outcome:
                'It passes on the third run. The flaky test stays, and the team learns to ignore failed builds.',
            },
            {
              label: 'Look at the failed test and report it as flaky',
              good: true,
              outcome:
                'You see the test uses `new Date()`. It fails when CI runs close to midnight.',
              next: 'fix',
            },
          ],
        },
        {
          id: 'fix',
          text: 'The owner of the test is busy this week, and the test blocks other PRs.',
          choices: [
            {
              label: 'Quarantine the test, open an issue, and link it to the owner',
              good: true,
              outcome:
                'Other PRs can be merged again, and the fix is tracked. The owner sets a fixed date in the test.',
            },
            {
              label: 'Delete the test',
              good: false,
              outcome:
                'The flaky failures stop, but the behavior it protected is no longer tested.',
            },
          ],
        },
      ],
    },
  ],
};
