import { ArrowRight, Check, X } from 'lucide-react';
import { useState, type CSSProperties, type ReactNode } from 'react';
import { Inline } from './Inline';
import type {
  AnatomyBlock,
  BoundaryBlock,
  IssueSimulatorBlock,
  RangeBlock,
  RunsBlock,
  SemverBlock,
  SequenceBlock,
  Tone,
} from '../foundations/types';

function Frame({
  title,
  className,
  children,
}: Readonly<{ title?: string; className: string; children: ReactNode }>) {
  return (
    <figure className={`git-diagram ${className}`}>
      {title !== undefined && (
        <p className="scene-title">
          <Inline text={title} />
        </p>
      )}
      {children}
    </figure>
  );
}

function Tabs({
  names,
  index,
  onChange,
}: Readonly<{ names: string[]; index: number; onChange: (index: number) => void }>) {
  if (names.length < 2) {
    return null;
  }
  return (
    <div className="git-diagram-tabs" role="group" aria-label="Choose what to show">
      {names.map((name, itemIndex) => (
        <button
          type="button"
          key={name}
          className={itemIndex === index ? 'active' : ''}
          aria-pressed={itemIndex === index}
          onClick={() => onChange(itemIndex)}
        >
          <Inline text={name} />
        </button>
      ))}
    </div>
  );
}

function Caption({ summary, description }: Readonly<{ summary: string; description?: string }>) {
  return (
    <figcaption>
      <strong>
        <Inline text={summary} />
      </strong>
      {description !== undefined && (
        <>
          {' '}
          <Inline text={description} />
        </>
      )}
    </figcaption>
  );
}

function Legend({ items }: Readonly<{ items?: { tone: Tone; label: string }[] }>) {
  if (items === undefined) {
    return null;
  }
  return (
    <ul className="git-diagram-legend">
      {items.map((item) => (
        <li key={item.label}>
          <span className={`scene-swatch tone-${item.tone}`} aria-hidden="true" />
          <Inline text={item.label} />
        </li>
      ))}
    </ul>
  );
}

// Keeps a tab index valid when the content changes.
function useTab(count: number) {
  const [index, setIndex] = useState(0);
  return [Math.min(index, count - 1), setIndex] as const;
}

// ── Issue simulator ────────────────────────────────────────────────────────────

const closing = /\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)$/i;

interface IssueResult {
  issue: string;
  closes: boolean;
  reason: string;
}

function evaluate(
  text: string,
  branch: string,
  place: string,
  defaultBranch: string
): IssueResult[] {
  const results: IssueResult[] = [];
  for (const match of text.matchAll(/#\d+/g)) {
    // Include an owner/repo prefix written right before the number.
    let start = match.index;
    while (start > 0 && /[\w./-]/.test(text[start - 1])) {
      start--;
    }
    const issue = text.slice(start, match.index + match[0].length);
    if (results.some((item) => item.issue === issue)) {
      continue;
    }
    const before = text.slice(0, start).trimEnd();
    const hasKeyword = closing.test(before);
    let reason = 'Closes when the PR is merged.';
    let closes = true;
    if (!hasKeyword) {
      closes = false;
      reason = `No closing keyword directly before it. Write "Fixes ${issue}".`;
    } else if (place === 'comment') {
      closes = false;
      reason = 'Keywords in PR comments only mention the issue. Use the description.';
    } else if (branch !== defaultBranch) {
      closes = false;
      reason = `The PR targets ${branch}, not the default branch ${defaultBranch}.`;
    } else if (place === 'commit') {
      reason = `Closes when this commit reaches ${defaultBranch}.`;
    }
    results.push({ issue, closes, reason });
  }
  return results;
}

export function IssueSimulator({ block }: Readonly<{ block: IssueSimulatorBlock }>) {
  const [text, setText] = useState(block.initialText);
  const [branch, setBranch] = useState(block.defaultBranch);
  const [place, setPlace] = useState('description');
  const results = evaluate(text, branch, place, block.defaultBranch);

  return (
    <Frame title="Try it: will the issue close?" className="issue-sim">
      <div className="issue-sim-form">
        <label>
          Target branch
          <select value={branch} onChange={(event) => setBranch(event.target.value)}>
            {block.branches.map((item) => (
              <option key={item} value={item}>
                {item}
                {item === block.defaultBranch ? ' (default)' : ''}
              </option>
            ))}
          </select>
        </label>
        <label>
          Written in
          <select value={place} onChange={(event) => setPlace(event.target.value)}>
            <option value="description">PR description</option>
            <option value="commit">Commit message</option>
            <option value="comment">PR comment</option>
          </select>
        </label>
        <label className="issue-sim-text">
          Text
          <textarea rows={2} value={text} onChange={(event) => setText(event.target.value)} />
        </label>
      </div>
      <div className="issue-sim-examples">
        <span>Examples:</span>
        {block.examples.map((example) => (
          <button
            type="button"
            key={example}
            className="chip"
            onClick={() => {
              setText(example);
              setBranch(block.defaultBranch);
              setPlace('description');
            }}
          >
            {example}
          </button>
        ))}
      </div>
      <ul className="issue-sim-results" aria-live="polite">
        {results.length === 0 && <li className="muted">No issue number in the text yet.</li>}
        {results.map((item) => (
          <li key={item.issue} className={item.closes ? 'closes' : 'open'}>
            <span className="issue-sim-badge">
              {item.closes ? <Check size={14} /> : <X size={14} />}
              {item.issue} {item.closes ? 'closes' : 'stays open'}
            </span>
            <span>{item.reason}</span>
          </li>
        ))}
      </ul>
    </Frame>
  );
}

// ── Anatomy ───────────────────────────────────────────────────────────────────

export function Anatomy({ block }: Readonly<{ block: AnatomyBlock }>) {
  const [active, setActive] = useState(block.parts[0].id);
  const part = block.parts.find((item) => item.id === active) ?? block.parts[0];

  return (
    <Frame title={block.title} className="anatomy">
      <pre className={`anatomy-artifact ${block.large === true ? 'large' : ''}`}>
        {block.segments.map((segment, index) => {
          if (segment.part === undefined) {
            // eslint-disable-next-line react/no-array-index-key -- segments are positional
            return <span key={index}>{segment.text}</span>;
          }
          const blank = segment.text.trim() === '';
          return (
            // eslint-disable-next-line react/no-array-index-key -- segments are positional
            <span key={index}>
              <span
                role="button"
                tabIndex={0}
                aria-pressed={segment.part === active}
                className={`anatomy-part ${blank ? 'blank' : ''} ${segment.part === active ? 'active' : ''}`}
                onClick={() => setActive(segment.part ?? active)}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    event.preventDefault();
                    setActive(segment.part ?? active);
                  }
                }}
              >
                {blank ? '↵ blank line' : segment.text}
              </span>
              {blank && segment.text}
            </span>
          );
        })}
      </pre>
      <div className="anatomy-chips" role="group" aria-label="Parts">
        {block.parts.map((item) => (
          <button
            type="button"
            key={item.id}
            className={`chip ${item.id === active ? 'selected' : ''}`}
            aria-pressed={item.id === active}
            onClick={() => setActive(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>
      <p className="anatomy-explanation" aria-live="polite">
        <strong>{part.label}.</strong> <Inline text={part.explanation} />
      </p>
    </Frame>
  );
}

// ── SemVer bump ───────────────────────────────────────────────────────────────

const changes = [
  { kind: 'Breaking change', position: 0, why: 'Existing users may need to change their code.' },
  { kind: 'New feature', position: 1, why: 'Something new is added. Existing code keeps working.' },
  { kind: 'Bug fix', position: 2, why: 'A mistake is fixed. Nothing new, nothing breaks.' },
];
const partNames = ['MAJOR', 'MINOR', 'PATCH'];

export function SemverBump({ block }: Readonly<{ block: SemverBlock }>) {
  const [choice, setChoice] = useState<number>();
  const start = block.start.split('.').map(Number);
  const change = choice === undefined ? undefined : changes[choice];
  const next = start.map((value, index) => {
    if (change === undefined || index < change.position) {
      return value;
    }
    return index === change.position ? value + 1 : 0;
  });

  return (
    <Frame title="Try it: what is the next version?" className="semver">
      <div className="git-diagram-tabs" role="group" aria-label="Kind of change">
        {changes.map((item, index) => (
          <button
            type="button"
            key={item.kind}
            className={index === choice ? 'active' : ''}
            aria-pressed={index === choice}
            onClick={() => setChoice(index)}
          >
            {item.kind}
          </button>
        ))}
      </div>
      <div className="semver-row">
        <Version values={start} />
        <ArrowRight size={22} aria-hidden="true" className="semver-arrow" />
        {change === undefined ? (
          <span className="semver-placeholder">Pick a kind of change</span>
        ) : (
          <Version values={next} changed={change.position} />
        )}
      </div>
      {change !== undefined && (
        <figcaption>
          <strong>
            {change.kind}: {partNames[change.position]} goes up.
          </strong>{' '}
          {change.why}
          {change.position < 2 && ' The numbers to the right go back to 0.'}
        </figcaption>
      )}
    </Frame>
  );
}

function Version({ values, changed }: Readonly<{ values: number[]; changed?: number }>) {
  return (
    <span className="semver-version">
      {values.map((value, index) => {
        let state = '';
        if (changed !== undefined && index === changed) {
          state = 'up';
        } else if (changed !== undefined && index > changed) {
          state = 'reset';
        }
        return (
          <span key={partNames[index]} className="semver-part-wrap">
            {index > 0 && <span className="semver-dot">.</span>}
            <span className={`semver-part ${state}`}>
              <span className="semver-number">{value}</span>
              <span className="semver-name">{partNames[index]}</span>
            </span>
          </span>
        );
      })}
    </span>
  );
}

// ── Sequence diagram ──────────────────────────────────────────────────────────

export function Sequence({ block }: Readonly<{ block: SequenceBlock }>) {
  const [index, setIndex] = useTab(block.scenes.length);
  const scene = block.scenes[index];
  const columns = { gridTemplateColumns: `repeat(${block.actors.length}, minmax(0, 1fr))` };

  return (
    <Frame title={block.title} className="sequence">
      <Tabs names={block.scenes.map((item) => item.name)} index={index} onChange={setIndex} />
      <div className="sequence-canvas">
        <div className="sequence-grid" style={columns}>
          {block.actors.map((actor) => (
            <span key={actor} className="sequence-actor">
              {actor}
            </span>
          ))}
        </div>
        <div className="sequence-body">
          <div className="sequence-grid sequence-lifelines" style={columns} aria-hidden="true">
            {block.actors.map((actor) => (
              <span key={actor} />
            ))}
          </div>
          {scene.messages.map((message, messageIndex) => {
            const left = Math.min(message.from, message.to);
            const right = Math.max(message.from, message.to);
            const self = message.from === message.to;
            let direction = message.to > message.from ? 'right' : 'left';
            if (self) {
              direction = 'self';
            }
            const style = {
              gridColumn: self ? `${left + 1}` : `${left + 1} / ${right + 2}`,
              '--span': right - left + 1,
            } as CSSProperties;
            return (
              // eslint-disable-next-line react/no-array-index-key -- messages are positional
              <div key={`${scene.name}-${messageIndex}`} className="sequence-grid" style={columns}>
                <div
                  className={`sequence-message tone-${message.tone ?? 'neutral'} ${direction}`}
                  style={style}
                >
                  <span className="sequence-label">
                    <Inline text={message.label} />
                  </span>
                  {!self && <span className="sequence-arrow" aria-hidden="true" />}
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <Caption summary={scene.summary} description={scene.description} />
    </Frame>
  );
}

// ── Number line for version ranges ────────────────────────────────────────────

function compare(a: string, b: string) {
  const left = a.split('.').map(Number);
  const right = b.split('.').map(Number);
  for (let index = 0; index < 3; index++) {
    if (left[index] !== right[index]) {
      return left[index] - right[index];
    }
  }
  return 0;
}

export function RangeLine({ block }: Readonly<{ block: RangeBlock }>) {
  const [index, setIndex] = useTab(block.ranges.length);
  const range = block.ranges[index];

  return (
    <Frame title={block.title} className="range">
      <Tabs names={block.ranges.map((item) => item.name)} index={index} onChange={setIndex} />
      <ol className="range-line">
        {range.versions.map((version, versionIndex) => {
          const isAllowed = (value: string | undefined) =>
            value !== undefined &&
            compare(value, range.from) >= 0 &&
            compare(value, range.below) < 0;
          const allowed = isAllowed(version);
          const previous = isAllowed(range.versions[versionIndex - 1]);
          const next = isAllowed(range.versions[versionIndex + 1]);
          let band = '';
          if (allowed) {
            band = `${previous ? '' : 'band-start'} ${next ? '' : 'band-end'}`;
          } else if (previous) {
            band = 'band-after';
          }
          return (
            <li key={version} className={`${allowed ? 'allowed' : 'blocked'} ${band}`}>
              <span className="range-tick" aria-hidden="true" />
              <span className="range-version">{version}</span>
              <span className="range-state">
                {allowed ? <Check size={13} /> : <X size={13} />}
                {allowed ? 'allowed' : 'not allowed'}
              </span>
            </li>
          );
        })}
      </ol>
      <Caption
        summary={range.summary}
        description={`Allowed: \`>=${range.from}\` and \`<${range.below}\`.`}
      />
    </Frame>
  );
}

// ── Boundary map ──────────────────────────────────────────────────────────────

export function BoundaryMap({ block }: Readonly<{ block: BoundaryBlock }>) {
  const [index, setIndex] = useTab(block.scenes.length);
  const scene = block.scenes[index];

  return (
    <Frame title={block.title} className="boundary">
      <Tabs names={block.scenes.map((item) => item.name)} index={index} onChange={setIndex} />
      <div className="boundary-canvas">
        {scene.zones.map((zone, zoneIndex) => {
          const links = (scene.links ?? []).filter((link) => link.from === zoneIndex);
          return (
            // eslint-disable-next-line react/no-array-index-key -- zones are positional
            <div className="boundary-step" key={`${scene.name}-${zoneIndex}`}>
              <section className={`boundary-zone tone-${zone.tone ?? 'neutral'}`}>
                <p className="boundary-label">
                  <Inline text={zone.label} />
                </p>
                <ul>
                  {zone.items.map((item) => (
                    <li key={item.text} className={`scene-block tone-${item.tone ?? 'neutral'}`}>
                      <span className="scene-box">
                        <Inline text={item.text} />
                      </span>
                      {item.note !== undefined && (
                        <span className="scene-note">
                          <Inline text={item.note} />
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              </section>
              {links.length > 0 && (
                <ul className="boundary-links">
                  {links.map((link) => (
                    <li key={link.label} className={link.ok ? 'ok' : 'blocked'}>
                      <span className="boundary-link-icon" aria-hidden="true">
                        {link.ok ? <ArrowRight size={16} /> : <X size={16} />}
                      </span>
                      <Inline text={link.label} />
                    </li>
                  ))}
                </ul>
              )}
            </div>
          );
        })}
      </div>
      <Caption summary={scene.summary} description={scene.description} />
      <Legend items={block.legend} />
    </Frame>
  );
}

// ── Run history ───────────────────────────────────────────────────────────────

export function RunStrip({ block }: Readonly<{ block: RunsBlock }>) {
  const [index, setIndex] = useTab(block.scenes.length);
  const scene = block.scenes[index];

  return (
    <Frame title={block.title} className="runs">
      <Tabs names={block.scenes.map((item) => item.name)} index={index} onChange={setIndex} />
      <div className="runs-canvas">
        {scene.rows.map((row) => (
          <div className="runs-row" key={`${scene.name}-${row.label}`}>
            <span className="scene-label">
              <Inline text={row.label} />
            </span>
            <ol className="runs-list">
              {row.runs.map((run, runIndex) => (
                // eslint-disable-next-line react/no-array-index-key -- runs are positional
                <li key={runIndex} className={`run ${run}`} title={`Run ${runIndex + 1}: ${run}`}>
                  {run === 'pass' ? <Check size={14} /> : <X size={14} />}
                  <span className="visually-hidden">
                    Run {runIndex + 1} {run === 'pass' ? 'passed' : 'failed'}
                  </span>
                </li>
              ))}
            </ol>
          </div>
        ))}
      </div>
      <Caption summary={scene.summary} description={scene.description} />
      <ul className="git-diagram-legend">
        <li>
          <span className="run pass small" aria-hidden="true">
            <Check size={11} />
          </span>
          Passed
        </li>
        <li>
          <span className="run fail small" aria-hidden="true">
            <X size={11} />
          </span>
          Failed
        </li>
      </ul>
    </Frame>
  );
}
