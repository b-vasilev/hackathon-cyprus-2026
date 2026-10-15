import type { FoundationModule } from '../types';

const PROMPT = 'Prompt';
const AI_DRAFT = 'AI draft';
const YOU_READ = 'You read it';
const RUN_TESTS = 'Run tests';
const YOUR_LAPTOP = 'Your laptop';
const WORKFLOW = 'Workflow';

export const aiAssistants: FoundationModule = {
  id: 'ai-assistants',
  title: 'Working with AI assistants',
  summary: 'Which tools to use, how to check what they write, and what never to paste in.',
  outcome: 'Use an AI assistant to work faster without shipping its mistakes or leaking data.',
  minutes: 15,
  lessons: [
    {
      id: 'tools',
      title: 'The tools we use',
      summary: 'Three assistants you will use, and why approved tools matter.',
      blocks: [
        {
          type: 'p',
          text: 'AI assistants are now a normal part of engineering work. They are good at explaining code you do not know, writing a first draft, writing tests and finding obvious bugs. Use them. Use only the approved ones, and stay responsible for the result.',
        },
        {
          type: 'table',
          columns: ['Tool', 'What it is', 'Good for'],
          rows: [
            [
              'Claude Code in VS Code',
              "Anthropic's coding assistant as a VS Code extension. It also runs in the terminal. It reads your project and proposes edits as diffs, which you accept or reject. It runs commands when you allow it.",
              'Understanding a codebase, changes across many files, writing tests, explaining a stack trace (the list of function calls when an error happened).',
            ],
            [
              'Codex',
              "OpenAI's coding agent. It works from the terminal or an IDE extension. It can also run cloud tasks: these work on a copy of the repo and propose changes for you to review.",
              'Clear tasks you can hand off and review later, like a small fix or a refactor.',
            ],
            [
              'SecondStack',
              "A self-hosted AI workspace for chat, code and agents. It runs behind the company's own LLM gateway, with single sign-on and access rules for each team.",
              'Work with internal code or documents, because the data stays inside infrastructure that the company controls.',
            ],
          ],
        },
        {
          type: 'p',
          text: 'Where your code goes depends on the tool and the account you use.',
        },
        {
          type: 'boundary',
          title: 'Where your code goes',
          scenes: [
            {
              name: 'Approved tool',
              summary: 'Your code stays under rules that the company has checked.',
              description:
                'SecondStack keeps the data inside company infrastructure. Claude Code and Codex run with your work account.',
              zones: [
                { label: YOUR_LAPTOP, items: [{ text: 'Your code' }, { text: 'Your prompt' }] },
                {
                  label: 'Company-approved service',
                  tone: 'brand',
                  items: [
                    {
                      text: 'SecondStack',
                      tone: 'brand',
                      note: "behind the company's LLM gateway",
                    },
                    {
                      text: 'Claude Code, Codex',
                      tone: 'brand',
                      note: 'work account, company agreement',
                    },
                  ],
                },
              ],
              links: [{ from: 0, label: 'approved tool, work account', ok: true }],
            },
            {
              name: 'Personal account',
              summary: 'Your code leaves company control.',
              description:
                "The service's consumer terms apply. They can allow it to store your code or use it for training.",
              zones: [
                { label: YOUR_LAPTOP, items: [{ text: 'Your code' }, { text: 'Your prompt' }] },
                {
                  label: 'Personal chatbot account',
                  tone: 'danger',
                  items: [
                    { text: 'Consumer terms', tone: 'warning', note: 'not checked by the company' },
                    { text: 'May store your code', tone: 'danger' },
                    { text: 'May train on it', tone: 'danger' },
                  ],
                },
              ],
              links: [{ from: 0, label: 'leaves company control', ok: false }],
            },
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Personal accounts are not approved tools',
          text: "If you paste work code into a personal chatbot account, the company data falls under that service's consumer terms. These terms can allow the service to store the data or use it for training. Use only the tools and accounts that your team has approved.",
        },
      ],
    },
    {
      id: 'checking-output',
      title: 'Checking what it writes',
      summary: 'How to check AI code, because it looks confident even when it is wrong.',
      blocks: [
        {
          type: 'p',
          text: 'Assistants write code that looks right. Usually it is right. Sometimes it calls a function that does not exist, or uses an old version of an API. It can ignore an edge case, an unusual input such as an empty list. It can also remove error handling without telling you. The code looks just as confident when it is wrong.',
        },
        {
          type: 'scenes',
          title: 'From prompt to commit',
          legend: [
            { tone: 'brand', label: 'Step done' },
            { tone: 'soft', label: 'Written by AI' },
            { tone: 'strong', label: 'Result' },
            { tone: 'muted', label: 'Skipped' },
            { tone: 'warning', label: 'Needs action' },
            { tone: 'danger', label: 'Problem' },
          ],
          scenes: [
            {
              name: 'Good workflow',
              summary: 'You read and test the AI draft before you commit it.',
              rows: [
                {
                  label: WORKFLOW,
                  blocks: [
                    { text: PROMPT, tone: 'brand', note: 'error, file, what you tried' },
                    { text: AI_DRAFT, tone: 'soft' },
                    { text: YOU_READ, tone: 'brand' },
                    { text: RUN_TESTS, tone: 'brand' },
                    { text: 'Commit', tone: 'strong', note: 'you can explain it' },
                  ],
                },
              ],
            },
            {
              name: 'Skipped review',
              summary: 'Nobody read or tested the draft, so its mistake is now in the commit.',
              description: 'You own the code you commit, also when the AI wrote it.',
              rows: [
                {
                  label: WORKFLOW,
                  blocks: [
                    { text: PROMPT, tone: 'brand' },
                    { text: AI_DRAFT, tone: 'soft', note: 'calls a missing function' },
                    { text: YOU_READ, tone: 'muted', note: 'skipped' },
                    { text: RUN_TESTS, tone: 'muted', note: 'skipped' },
                    { text: 'Commit', tone: 'danger', note: 'the bug is committed' },
                  ],
                },
              ],
            },
            {
              name: 'Pasted a secret',
              summary:
                'The secret has left your machine, and a careful review cannot bring it back.',
              description:
                'Rotate the key: create a new one and turn off the old one. Remove secrets before you paste.',
              rows: [
                {
                  label: WORKFLOW,
                  blocks: [
                    { text: PROMPT, tone: 'danger', note: 'contains an API key' },
                    { text: AI_DRAFT, tone: 'soft' },
                    { text: YOU_READ, tone: 'brand' },
                    { text: RUN_TESTS, tone: 'brand' },
                  ],
                },
                {
                  label: 'Next',
                  blocks: [
                    { text: 'Rotate the key', tone: 'warning', note: 'new key, old one off' },
                  ],
                },
              ],
            },
          ],
        },
        {
          type: 'list',
          items: [
            'Read every line before you commit it. If you cannot explain it, do not ship it. Ask the assistant to explain it, then check that explanation.',
            'Run the code and test it. Code that compiles can still be wrong.',
            'Check that suggested packages really exist and are the real, maintained ones. Attackers publish fake packages with names that assistants often invent.',
            'Look for common mistakes: secrets written into the code, logging of personal data, errors that are caught and then ignored, and deleted tests.',
            'Give the assistant context: the error message, the file that matters and what you already tried. A vague prompt gives vague code.',
          ],
        },
        {
          type: 'p',
          text: 'The person who commits the code owns it. "The AI wrote it" is not an answer in a code review or an incident review.',
        },
      ],
    },
    {
      id: 'data-and-honesty',
      title: 'What not to paste, and being open about it',
      summary: 'What data never goes into a prompt, and how to say when AI wrote your code.',
      blocks: [
        {
          type: 'p',
          text: 'Some things must never go into a prompt, even with approved tools:',
        },
        {
          type: 'list',
          items: [
            'Secrets: API keys, passwords, tokens, private keys, connection strings.',
            'Personal data of customers: names, emails, addresses, payment details, and anything from production databases.',
            'Confidential documents that you may not share, such as contracts or client material.',
          ],
        },
        {
          type: 'p',
          text: "Do you need real-looking data to debug? Replace it with fake values first. `jane.doe@example.com` works just as well for debugging as a real customer's email.",
        },
        {
          type: 'p',
          text: 'Be open about AI use. If an assistant wrote a large part of a change, say so in the PR. For example: "Generated with Claude Code, reviewed and tested by me." Then reviewers know to look carefully for the kinds of mistakes that AI makes.',
        },
      ],
    },
  ],
  quiz: [
    {
      id: 'ai-safe-prompt',
      kind: 'cards',
      look: 'file',
      prompt: 'Which of these prompts is fine to send?',
      cards: [
        {
          title: PROMPT,
          body: 'Why does this fail?\nDATABASE_URL=postgres://admin:Pr0d-2026!@db.internal:5432/app',
          flaw: 'admin:Pr0d-2026!',
          feedback:
            'That is a production password. Remove credentials before you paste anything. Then rotate this password: replace it with a new one.',
        },
        {
          title: PROMPT,
          body: 'Format this as a table:\nMaria Kowalska, maria.k@gmail.com, +357 99 123456, card ending 4421…',
          flaw: 'Maria Kowalska, maria.k@gmail.com, +357 99 123456',
          feedback:
            'This is real personal data of a customer. Use made-up values with the same format instead.',
        },
        {
          title: PROMPT,
          body: 'This test fails with "TypeError: cannot read properties of undefined (reading \'id\')". Here is the function and the test. What am I missing?',
          correct: true,
          feedback:
            'It has the error, the code and the question, with no secrets or personal data. This is a useful and safe prompt.',
        },
        {
          title: 'Prompt (personal ChatGPT account)',
          body: "Here's our whole internal billing service, can you refactor it?",
          flaw: 'personal ChatGPT account',
          feedback:
            'The content might be fine for an approved tool, but not for a personal account. Company code goes to approved tools only.',
        },
      ],
    },
    {
      id: 'ai-check-before-commit',
      kind: 'multiple',
      prompt:
        'An assistant just wrote a 60-line function for you. What should you do before you commit it?',
      options: [
        'Read it and make sure you can explain each part',
        'Run it, and run the tests',
        'Check that any new packages it added really exist and are the ones you expect',
        'Commit it immediately, because AI code is already reviewed',
      ],
      correct: [0, 1, 2],
      explanation:
        'You are the author of everything you commit. AI code gets the same review as your own code. Also check for APIs and packages that the AI invented.',
    },
    {
      id: 'ai-tool-choice',
      kind: 'single',
      scenario: 'You want help to understand a module in a private company repo.',
      prompt: 'What is the right way to do it?',
      options: [
        'Paste the files into a free chatbot on your personal account',
        'Use an approved tool like Claude Code in VS Code, Codex or SecondStack, signed in with your work account',
        'Email the code to a friend who knows the framework',
        'Post it on a public forum',
      ],
      correct: [1],
      explanation:
        'Approved tools run under agreements that the company has checked. Personal accounts and public posts put the code outside company control.',
    },
    {
      id: 'ai-disclosure',
      kind: 'single',
      scenario: 'An assistant generated most of the code in your PR. You reviewed and tested it.',
      prompt: 'What do you write in the PR?',
      options: [
        'Nothing, because it is your PR now',
        'A short note, for example "Generated with Claude Code, reviewed and tested by me"',
        'Ask the reviewer to approve without reading, because the AI checked it',
        'Add a comment to every line to say that AI wrote it',
      ],
      correct: [1],
      explanation:
        'One line is enough. It tells the reviewer what to look for, and you still own the change.',
    },
    {
      id: 'ai-fake-data',
      kind: 'single',
      scenario:
        'A function fails for one customer record from production. You want an assistant to help you debug it.',
      prompt: 'What do you paste into the prompt?',
      options: [
        'The real record, because the assistant needs the exact data',
        'The same record with fake values in the same format, like `jane.doe@example.com`',
        'The real record, but only in an approved tool',
        'A screenshot of the record from the production database',
      ],
      correct: [1],
      explanation:
        'Personal data of customers never goes into a prompt, also with approved tools. Fake values in the same format are enough to debug.',
    },
  ],
};
