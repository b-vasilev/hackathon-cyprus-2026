import { Check, Play, RotateCcw, User, Wrench } from 'lucide-react';
import { useRef, useState } from 'react';
import { setFlag, withAchievements } from '../foundations/achievements';
import { Inline } from './Inline';
import type { CodeToolsBlock } from '../foundations/types';

interface Line {
  text: string;
  statement: number;
  // Last line of its statement; findings are shown under it.
  last: boolean;
}

// The code after the given tools have been applied, in any order.
function render(block: CodeToolsBlock, applied: Set<string>): Line[] {
  const formatter = block.tools.find((tool) => tool.formats === true)?.name;
  return block.statements.flatMap((statement, index) => {
    const fixes = (statement.fixes ?? []).filter((fix) => applied.has(fix.tool));
    if (fixes.some((fix) => fix.remove === true)) {
      return [];
    }
    let text =
      formatter !== undefined && applied.has(formatter)
        ? (statement.formatted ?? statement.text)
        : statement.text;
    for (const fix of fixes) {
      for (const [from, to] of fix.replace ?? []) {
        text = text.replace(from, to);
      }
    }
    const lines = text.split('\n');
    return lines.map((line, at) => ({
      text: line,
      statement: index,
      last: at === lines.length - 1,
    }));
  });
}

function CodePane({
  label,
  lines,
  findings,
  changedFrom,
}: Readonly<{
  label: string;
  lines: Line[];
  findings?: (statement: number) => string[];
  changedFrom?: Set<string>;
}>) {
  return (
    <div className="code-pane">
      <p className="code-pane-label">{label}</p>
      <pre>
        {lines.map((line, index) => {
          const notes = findings === undefined ? [] : findings(line.statement);
          const changed = changedFrom !== undefined && !changedFrom.has(line.text);
          return (
            <span
              // eslint-disable-next-line react/no-array-index-key -- code lines are positional
              key={index}
              className={`code-line ${notes.length > 0 ? 'flagged' : ''} ${changed ? 'changed' : ''}`}
            >
              <span className="line-no">{index + 1}</span>
              <code>{line.text === '' ? ' ' : line.text}</code>
              {line.last &&
                notes.map((note) => (
                  <span key={note} className="code-finding">
                    {note}
                  </span>
                ))}
            </span>
          );
        })}
      </pre>
    </div>
  );
}

// Run any tool at any time: the left shows what it finds, the right shows the result.
export function CodeTools({ block }: Readonly<{ block: CodeToolsBlock }>) {
  const [applied, setApplied] = useState<string[]>([]);
  const [view, setView] = useState<string>();
  const tool = block.tools.find((item) => item.name === view);
  const current = render(block, new Set(applied));
  const before =
    view === undefined ? current : render(block, new Set(applied.filter((name) => name !== view)));

  function run(name: string) {
    if (!applied.includes(name)) {
      setApplied([...applied, name]);
    }
    setView(name);
  }

  const resets = useRef(0);

  function reset() {
    setApplied([]);
    setView(undefined);
    resets.current += 1;
    if (resets.current === 5) {
      withAchievements(() => setFlag('perfectionist'));
    }
  }

  const findings = (statement: number) =>
    (block.statements[statement].findings ?? [])
      .filter((item) => item.tool === view)
      .map((item) => item.text);

  return (
    <figure className="git-diagram code-tools">
      {block.title !== undefined && (
        <p className="scene-title">
          <Inline text={block.title} />
        </p>
      )}
      <div className="code-tools-bar">
        {block.tools.map((item) => {
          const done = applied.includes(item.name);
          return (
            <button
              type="button"
              key={item.name}
              className={`code-tools-run ${item.name === view ? 'active' : ''} ${done ? 'done' : ''}`}
              aria-pressed={item.name === view}
              onClick={() => run(item.name)}
            >
              {done ? <Check size={15} /> : <Play size={15} />}
              {done ? item.name : `Run ${item.name.toLowerCase()}`}
            </button>
          );
        })}
        {applied.length > 0 && (
          <button type="button" className="text-button" onClick={reset}>
            <RotateCcw size={15} />
            Start again
          </button>
        )}
      </div>
      <div className="code-tools-panes">
        {tool === undefined ? (
          <>
            <CodePane label="Your code" lines={current} />
            <div className="code-pane">
              <p className="code-pane-label">After</p>
              <p className="code-pane-empty">
                Run any tool, in any order, to see what it finds and fixes.
              </p>
            </div>
          </>
        ) : (
          <>
            <CodePane
              label={`Before: what the ${tool.name.toLowerCase()} finds`}
              lines={before}
              findings={findings}
            />
            <CodePane
              label="After: the cleaner version"
              lines={current}
              changedFrom={new Set(before.map((line) => line.text))}
            />
          </>
        )}
      </div>
      {tool !== undefined && (
        <figcaption>
          <span className="code-fixed-by">
            {tool.fixedBy === 'tool' ? <Wrench size={14} /> : <User size={14} />}
            {tool.fixedBy === 'tool'
              ? 'Fixed automatically by the tool'
              : 'The tool reports it. You fix it.'}
          </span>{' '}
          <Inline text={tool.summary} />
        </figcaption>
      )}
    </figure>
  );
}
