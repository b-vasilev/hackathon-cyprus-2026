import type { FoundationModule } from '../types';

export const testing: FoundationModule = {
  id: 'testing',
  title: 'Automated testing basics',
  summary: 'Kinds of tests, what makes a good one, and why some tests are flaky.',
  outcome: 'Pick the right kind of test for a change, and spot a flaky one.',
  minutes: 15,
  lessons: [
    {
      id: 'test-levels',
      title: 'Unit, integration and end-to-end',
      summary: 'Three levels of test and what each costs.',
      blocks: [
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
              'The contract between them changes',
            ],
            [
              'End-to-end',
              'The whole app through the UI or public API, like a user',
              'Seconds to minutes',
              'Pretty much anything changes',
            ],
          ],
        },
        {
          type: 'p',
          text: 'The usual advice is the test pyramid: lots of unit tests, fewer integration tests, and a handful of end-to-end tests for the journeys that really matter (sign up, pay, export). The higher you go, the more each test proves, but the slower it is and the harder it is to tell what went wrong.',
        },
      ],
    },
    {
      id: 'good-tests',
      title: 'What makes a good test',
      summary: 'Readable, reliable, and focused on behavior.',
      blocks: [
        {
          type: 'code',
          code: "test('gives 10% off orders over 100', () => {\n  // Arrange\n  const order = { total: 200 };\n  // Act\n  const result = applyDiscount(order);\n  // Assert\n  expect(result.total).toBe(180);\n});",
          caption:
            'Arrange, act, assert. The name says what behavior it protects, so a failure explains itself.',
        },
        {
          type: 'list',
          items: [
            "Test what the code does through its public interface, not how it does it inside. A refactor that changes nothing for the caller shouldn't break tests.",
            'Fixing a bug? Write a test that fails because of the bug first. Then fix it.',
            "Coverage tells you which lines ran, not whether anything was checked. 100% coverage with weak assertions doesn't tell you much.",
          ],
        },
      ],
    },
    {
      id: 'flaky-tests',
      title: 'Flaky tests',
      summary: 'Tests that pass and fail with no code change.',
      blocks: [
        {
          type: 'p',
          text: 'Usual suspects:',
        },
        {
          type: 'list',
          items: [
            "Time: depending on today's date, the time zone, or a fixed `sleep(2000)`.",
            'Order: tests sharing state, so they only pass in a certain sequence.',
            'Network: calling a real external service.',
            'Concurrency: not waiting properly for async work to finish.',
            'Randomness: random test data without a fixed seed.',
          ],
        },
        {
          type: 'callout',
          tone: 'tip',
          title: 'A flaky test is a bug',
          text: 'Re-running until it goes green teaches everyone to ignore red builds, and then a real failure slips through. Report it, quarantine it if you have to, and fix the cause.',
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
      explanation: "Set up the data, do the one thing you're testing, check the result.",
    },
    {
      id: 'test-level-choice',
      kind: 'single',
      scenario: 'You fixed a rounding bug in a pure function, `calculateTax(amount)`.',
      prompt: "What's the most efficient way to make sure it stays fixed?",
      options: [
        'A unit test for `calculateTax`',
        'A full end-to-end checkout test',
        'A manual test plan',
        'No test needed',
      ],
      correct: [0],
      explanation:
        'A pure function is perfect for a quick unit test. Make it fail without your fix and pass with it.',
    },
    {
      id: 'test-flaky-causes',
      kind: 'multiple',
      prompt: 'Which of these commonly make tests flaky?',
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
      id: 'test-refactor',
      kind: 'single',
      scenario:
        'You refactor a class without changing what it does, and 20 tests fail because they were checking which private methods got called.',
      prompt: 'What does that tell you?',
      options: [
        'The refactor is wrong',
        'The tests are tied to implementation details, not behavior',
        "Don't refactor",
        'Delete all the tests',
      ],
      correct: [1],
      explanation:
        'Tests that check the internals break on harmless changes. Check what comes out instead.',
    },
  ],
};
