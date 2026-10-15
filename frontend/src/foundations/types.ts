// Lesson and quiz content is plain data so curated, generated and product-specific
// material can share one renderer. Text fields support `inline code` in backticks.

export type Block =
  | { type: 'p'; text: string }
  | { type: 'list'; items: string[]; ordered?: boolean }
  | { type: 'code'; code: string; caption?: string }
  | { type: 'callout'; tone: 'tip' | 'warning'; title: string; text: string }
  | { type: 'table'; columns: string[]; rows: string[][] }
  // Interactive merge / rebase / squash history picture.
  | { type: 'git-history' };

export interface Lesson {
  id: string;
  title: string;
  summary: string;
  blocks: Block[];
}

interface QuestionBase {
  id: string;
  // Optional situation the learner reasons about before answering.
  scenario?: string;
  prompt: string;
}

export interface ChoiceQuestion extends QuestionBase {
  kind: 'single' | 'multiple';
  options: string[];
  // Indexes into options.
  correct: number[];
  explanation: string;
}

export interface Card {
  title: string;
  meta?: string;
  body?: string;
  // Exact text inside title, meta or body to highlight when this card is wrong.
  flaw?: string;
  feedback: string;
  correct?: boolean;
}

// Mock GitHub objects; the learner keeps picking until they find the right one.
export interface CardQuestion extends QuestionBase {
  kind: 'cards';
  look: 'pull-request' | 'commit' | 'branch' | 'file';
  cards: Card[];
}

// Template gaps are written as {{0}}, {{1}}…; answers[n] fills gap n.
export interface FillQuestion extends QuestionBase {
  kind: 'fill';
  template: string;
  code?: boolean;
  answers: string[];
  distractors: string[];
  explanation: string;
}

export interface BranchChoice {
  label: string;
  // What happens as a result of this choice.
  outcome: string;
  good: boolean;
  // Step to continue with; the scenario ends when omitted.
  next?: string;
}

// A situation that unfolds differently depending on each decision. Counts as
// correct only when every decision on the learner's path was a good one.
export interface BranchQuestion extends QuestionBase {
  kind: 'branch';
  steps: { id: string; text: string; choices: BranchChoice[] }[];
}

export interface PatchLine {
  type: 'add' | 'remove' | 'context';
  text: string;
  // Present on lines a reviewer should flag; explains the problem.
  issue?: string;
}

// Review a small diff by flagging problem lines. Correct when exactly the
// lines with issues are flagged.
export interface PatchQuestion extends QuestionBase {
  kind: 'patch';
  file: string;
  lines: PatchLine[];
}

export type Question =
  | ChoiceQuestion
  | CardQuestion
  | FillQuestion
  | BranchQuestion
  | PatchQuestion;

export interface FoundationModule {
  id: string;
  title: string;
  summary: string;
  outcome: string;
  minutes: number;
  lessons: Lesson[];
  quiz: Question[];
}
