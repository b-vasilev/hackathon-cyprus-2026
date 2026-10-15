// Lesson and quiz content is plain data so curated, generated and product-specific
// material can share one renderer. Text fields support `inline code` in backticks.

export type Block =
  | { type: 'p'; text: string }
  | { type: 'list'; items: string[]; ordered?: boolean }
  | { type: 'code'; code: string; caption?: string }
  | { type: 'callout'; tone: 'tip' | 'warning'; title: string; text: string }
  | { type: 'table'; columns: string[]; rows: string[][] }
  // Interactive merge / rebase / squash history picture.
  | { type: 'git-history' }
  | ScenesBlock
  | CodeToolsBlock
  | IssueSimulatorBlock
  | AnatomyBlock
  | SemverBlock
  | SequenceBlock
  | RangeBlock
  | BoundaryBlock
  | RunsBlock;

// Colour roles for diagram blocks. Each also differs in border or fill style,
// so meaning never depends on colour alone.
export type Tone =
  | 'neutral' // plain white block: existing or unaffected parts
  | 'brand' // solid violet: the thing this view is about
  | 'soft' // dashed violet: new, replaced or simulated
  | 'strong' // thick dark outline: a result or join point
  | 'muted' // dashed grey: skipped, unused or left behind
  | 'warning' // amber: needs attention
  | 'danger'; // red: failed or not allowed

export interface SceneBlock {
  text: string;
  tone?: Tone;
  // Short line under the block.
  note?: string;
}

export interface SceneRow {
  label?: string;
  blocks: SceneBlock[];
}

export interface Scene {
  name: string;
  // Bold first sentence of the caption.
  summary: string;
  description?: string;
  rows: SceneRow[];
}

// Tabs that each show rows of connected coloured blocks, like the Git history picture.
export interface ScenesBlock {
  type: 'scenes';
  title?: string;
  // 'flow' (default) joins blocks with lines, like steps. 'stack' centres each row
  // without lines, for counts and proportions such as the test pyramid.
  layout?: 'flow' | 'stack';
  legend?: { tone: Tone; label: string }[];
  scenes: Scene[];
}

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

export interface CodeTool {
  name: string;
  // Formatters rewrite code themselves; most other tools only report.
  fixedBy: 'tool' | 'you';
  // True for the formatter: applying it uses each statement's `formatted` text.
  formats?: boolean;
  summary: string;
}

export interface CodeStatement {
  // The code as first written. May span several lines.
  text: string;
  // Layout after the formatter runs. Defaults to `text`.
  formatted?: string;
  // Problems a tool reports on this statement.
  findings?: { tool: string; text: string }[];
  // How a tool's fix changes this statement: remove it, or replace text in it.
  // Replacements are applied when found, so list one for each layout if they differ.
  fixes?: { tool: string; remove?: boolean; replace?: [string, string][] }[];
}

// Code playground: run the tools in any order and compare the code before and after.
export interface CodeToolsBlock {
  type: 'code-tools';
  title?: string;
  tools: CodeTool[];
  statements: CodeStatement[];
}

// Interactive PR form: the learner edits a description and target branch and sees
// which issues GitHub would close on merge.
export interface IssueSimulatorBlock {
  type: 'issue-simulator';
  defaultBranch: string;
  branches: string[];
  initialText: string;
  // One-click example descriptions.
  examples: string[];
}

// A real artefact (version number, HTTP request, YAML…) with clickable parts.
export interface AnatomyBlock {
  type: 'anatomy';
  title?: string;
  // Text pieces in order; a piece with `part` is clickable. Use \n for new lines.
  segments: { text: string; part?: string }[];
  parts: { id: string; label: string; explanation: string }[];
  large?: boolean;
}

// Pick a kind of change and see how a SemVer version number moves.
export interface SemverBlock {
  type: 'semver';
  start: string;
}

// Who talks to whom, in time order. Tabs like scenes.
export interface SequenceBlock {
  type: 'sequence';
  title?: string;
  actors: string[];
  scenes: {
    name: string;
    summary: string;
    description?: string;
    // `from === to` draws a note in that actor's column.
    messages: { from: number; to: number; label: string; tone?: Tone }[];
  }[];
}

// Which versions a dependency range allows, on a number line. Tabs per range.
export interface RangeBlock {
  type: 'range';
  title?: string;
  ranges: {
    name: string;
    summary: string;
    // Allowed when from <= version < below.
    from: string;
    below: string;
    versions: string[];
  }[];
}

// Containers (machines, companies…) with items inside, and what can cross between them.
export interface BoundaryBlock {
  type: 'boundary';
  title?: string;
  legend?: { tone: Tone; label: string }[];
  scenes: {
    name: string;
    summary: string;
    description?: string;
    zones: { label: string; tone?: Tone; items: { text: string; tone?: Tone; note?: string }[] }[];
    // Shown between zone `from` and zone `from + 1`.
    links?: { from: number; label: string; ok: boolean }[];
  }[];
}

// A history of runs, like the list of CI runs for a branch.
export interface RunsBlock {
  type: 'runs';
  title?: string;
  scenes: {
    name: string;
    summary: string;
    description?: string;
    rows: { label: string; runs: ('pass' | 'fail')[] }[];
  }[];
}
