import type { FoundationModule } from '../types';

export const aiAssistants: FoundationModule = {
  id: 'ai-assistants',
  title: 'Working with AI assistants',
  summary: 'Which tools to use, how to check what they write, and what never to paste in.',
  outcome: 'Use an AI assistant to go faster without shipping its mistakes or leaking data.',
  minutes: 15,
  lessons: [
    {
      id: 'tools',
      title: 'The tools we use',
      summary: "Three assistants you'll come across, and why approved tools matter.",
      blocks: [
        {
          type: 'p',
          text: "AI assistants are now part of normal engineering work. They're good at explaining unfamiliar code, writing a first draft, generating tests and spotting obvious bugs. Use them. Just use the approved ones, and stay in charge of the result.",
        },
        {
          type: 'table',
          columns: ['Tool', 'What it is', 'Good for'],
          rows: [
            [
              'Claude Code in VS Code',
              "Anthropic's coding assistant as a VS Code extension (it also runs in the terminal). It reads your project, proposes edits as diffs you accept or reject, and runs commands when you allow it.",
              'Understanding a codebase, multi-file changes, writing tests, explaining a stack trace.',
            ],
            [
              'Codex',
              "OpenAI's coding agent. Works from the terminal, an IDE extension, or as cloud tasks that work on a copy of the repo and propose changes for you to review.",
              'Well-defined tasks you can hand off and review later, like a small fix or a refactor.',
            ],
            [
              'SecondStack',
              "A self-hosted AI workspace for chat, code and agents. It runs behind the company's own LLM gateway with single sign-on and per-team access rules.",
              'Work involving internal code or documents, because the data stays inside company-controlled infrastructure.',
            ],
          ],
        },
        {
          type: 'callout',
          tone: 'warning',
          title: 'Personal accounts are not approved tools',
          text: "Pasting work code into a personal chatbot account puts company data under that service's consumer terms, which can include storing it or using it for training. Stick to the tools and accounts your team has approved.",
        },
      ],
    },
    {
      id: 'checking-output',
      title: 'Checking what it writes',
      summary: "AI output looks confident whether it's right or not.",
      blocks: [
        {
          type: 'p',
          text: "Assistants produce code that looks right. Usually it is. Sometimes it calls a function that doesn't exist, uses an API from three versions ago, ignores an edge case, or quietly removes error handling. It reads just as confidently either way.",
        },
        {
          type: 'list',
          items: [
            "Read every line before you commit it. If you can't explain it, don't ship it. Ask the assistant to explain, then check the explanation.",
            'Run it and test it. "It compiles" proves very little.',
            'Check that suggested packages actually exist and are the real, maintained ones. Attackers publish lookalike packages with names assistants tend to invent.',
            'Watch for the classic slips: hard-coded secrets, logging personal data, swallowed errors, deleted tests.',
            'Give it context: the error message, the relevant file, what you already tried. Vague prompts get vague code.',
          ],
        },
        {
          type: 'p',
          text: 'Whoever commits the code owns it. "The AI wrote it" isn\'t an answer in a code review or an incident review.',
        },
      ],
    },
    {
      id: 'data-and-honesty',
      title: 'What not to paste, and being open about it',
      summary: 'Data rules, and saying when AI did the heavy lifting.',
      blocks: [
        {
          type: 'p',
          text: 'Even with approved tools, some things should never go into a prompt:',
        },
        {
          type: 'list',
          items: [
            'Secrets: API keys, passwords, tokens, private keys, connection strings.',
            'Customer personal data: names, emails, addresses, payment details, anything from production databases.',
            "Confidential documents that aren't yours to share, such as contracts or client material.",
          ],
        },
        {
          type: 'p',
          text: "Need to debug with real-looking data? Replace it with fake values first. `jane.doe@example.com` debugs just as well as a real customer's email.",
        },
        {
          type: 'p',
          text: 'Be open about it. If an assistant wrote a substantial part of a change, say so in the PR, for example: "Generated with Claude Code, reviewed and tested by me." Reviewers then know to look extra carefully for the kinds of mistakes AI makes.',
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
          title: 'Prompt',
          body: 'Why does this fail?\nDATABASE_URL=postgres://admin:Pr0d-2026!@db.internal:5432/app',
          flaw: 'admin:Pr0d-2026!',
          feedback:
            "That's a production password. Remove credentials before you paste anything, and rotate this one.",
        },
        {
          title: 'Prompt',
          body: 'Format this as a table:\nMaria Kowalska, maria.k@gmail.com, +357 99 123456, card ending 4421…',
          flaw: 'Maria Kowalska, maria.k@gmail.com, +357 99 123456',
          feedback: 'Real customer personal data. Use made-up values with the same shape instead.',
        },
        {
          title: 'Prompt',
          body: 'This test fails with "TypeError: cannot read properties of undefined (reading \'id\')". Here is the function and the test. What am I missing?',
          correct: true,
          feedback:
            "The error, the code and the question, with no secrets or personal data. That's a useful, safe prompt.",
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
        'An assistant just wrote a 60-line function for you. What should you do before committing it?',
      options: [
        'Read it and make sure you can explain each part',
        'Run it, and the tests',
        'Check that any new packages it added really exist and are the ones you expect',
        'Commit it straight away; AI code is already reviewed',
      ],
      correct: [0, 1, 2],
      explanation:
        "You're the author of anything you commit. AI code gets the same review as your own, plus a check for invented APIs and packages.",
    },
    {
      id: 'ai-tool-choice',
      kind: 'single',
      scenario: 'You want help understanding a module in a private company repo.',
      prompt: 'Which is the right way to do it?',
      options: [
        'Paste the files into a free chatbot on your personal account',
        'Use an approved tool like Claude Code in VS Code, Codex or SecondStack, signed in with your work account',
        'Email the code to a friend who knows the framework',
        'Post it on a public forum',
      ],
      correct: [1],
      explanation:
        'Approved tools run under agreements the company has checked. Personal accounts and public posts put the code outside its control.',
    },
    {
      id: 'ai-disclosure',
      kind: 'single',
      scenario:
        'Most of the code in your PR was generated with an assistant. You reviewed and tested it.',
      prompt: 'What do you write in the PR?',
      options: [
        "Nothing; it's your PR now",
        'A short note, e.g. "Generated with Claude Code, reviewed and tested by me"',
        'Ask the reviewer to approve without reading because the AI checked it',
        "Mark every line with a comment saying it's AI-written",
      ],
      correct: [1],
      explanation:
        'One line is enough. It tells the reviewer what to look out for, and you still own the change.',
    },
  ],
};
