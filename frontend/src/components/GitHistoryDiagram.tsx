import { useState } from 'react';

type Kind = 'main' | 'mine' | 'copy' | 'merge' | 'squash' | 'ghost';
type Lane = 'main' | 'feature' | 'old';

interface Commit {
  id: string;
  label: string;
  column: number;
  lane: Lane;
  kind: Kind;
}

interface View {
  name: string;
  summary: string;
  description: string;
  commits: Commit[];
  edges: [string, string][];
  tags: { at: string; label: string }[];
}

const laneY: Record<Lane, number> = { main: 44, feature: 124, old: 196 };
const columnX = (column: number) => 36 + column * 70;
const size = 34;

const base: Commit[] = [
  { id: 'A', label: 'A', column: 0, lane: 'main', kind: 'main' },
  { id: 'B', label: 'B', column: 1, lane: 'main', kind: 'main' },
  { id: 'X', label: 'X', column: 2, lane: 'main', kind: 'main' },
  { id: 'Y', label: 'Y', column: 3, lane: 'main', kind: 'main' },
];
const mine = (kind: Kind, lane: Lane = 'feature'): Commit[] => [
  { id: 'C', label: 'C', column: 2, lane, kind },
  { id: 'D', label: 'D', column: 3, lane, kind },
  { id: 'E', label: 'E', column: 4, lane, kind },
];
const baseEdges: [string, string][] = [
  ['A', 'B'],
  ['B', 'X'],
  ['X', 'Y'],
  ['B', 'C'],
  ['C', 'D'],
  ['D', 'E'],
];

const views: View[] = [
  {
    name: 'Before',
    summary: 'You branched from B and made C, D, E. Meanwhile teammates added X and Y to main.',
    description: 'The starting point: the two histories have split.',
    commits: [...base, ...mine('mine')],
    edges: baseEdges,
    tags: [
      { at: 'Y', label: 'main' },
      { at: 'E', label: 'feature' },
    ],
  },
  {
    name: 'Merge',
    summary: 'A new merge commit M joins both lines. C, D and E are untouched and keep their SHAs.',
    description:
      'Nothing is rewritten, so it is safe on shared branches. History shows the fork and the join.',
    commits: [
      ...base,
      ...mine('mine'),
      { id: 'M', label: 'M', column: 5, lane: 'main', kind: 'merge' },
    ],
    edges: [...baseEdges, ['Y', 'M'], ['E', 'M']],
    tags: [
      { at: 'M', label: 'main' },
      { at: 'E', label: 'feature' },
    ],
  },
  {
    name: 'Rebase',
    summary:
      'Your commits are replayed on top of Y as new copies C′ D′ E′ with new SHAs. The originals are dropped from the branch.',
    description:
      'History becomes one straight line. Because the commits are new, anyone who had the old C, D, E now has a different history: only rebase your own branch.',
    commits: [
      ...base,
      { id: 'C2', label: 'C′', column: 4, lane: 'feature', kind: 'copy' },
      { id: 'D2', label: 'D′', column: 5, lane: 'feature', kind: 'copy' },
      { id: 'E2', label: 'E′', column: 6, lane: 'feature', kind: 'copy' },
      ...mine('ghost', 'old'),
    ],
    edges: [
      ['A', 'B'],
      ['B', 'X'],
      ['X', 'Y'],
      ['Y', 'C2'],
      ['C2', 'D2'],
      ['D2', 'E2'],
      ['B', 'C'],
      ['C', 'D'],
      ['D', 'E'],
    ],
    tags: [
      { at: 'Y', label: 'main' },
      { at: 'E2', label: 'feature' },
    ],
  },
  {
    name: 'Squash',
    summary: 'All changes from C, D and E land on main as one new commit.',
    description:
      'main gets one tidy commit per PR. The individual commits stay only on the feature branch, which is usually deleted after merging.',
    commits: [
      ...base,
      ...mine('ghost'),
      { id: 'S', label: 'C+D+E', column: 4, lane: 'main', kind: 'squash' },
    ],
    edges: [...baseEdges, ['Y', 'S']],
    tags: [
      { at: 'S', label: 'main' },
      { at: 'E', label: 'feature' },
    ],
  },
];

function width(commit: Commit) {
  return commit.kind === 'squash' ? 72 : size;
}

function center(commit: Commit) {
  return { x: columnX(commit.column) + width(commit) / 2, y: laneY[commit.lane] };
}

const legend: { kind: Kind; label: string }[] = [
  { kind: 'main', label: 'Commit on main' },
  { kind: 'mine', label: 'Your commit' },
  { kind: 'copy', label: 'New copy (new SHA)' },
  { kind: 'merge', label: 'Merge commit' },
  { kind: 'ghost', label: 'Old / left behind' },
];

export function GitHistoryDiagram() {
  const [index, setIndex] = useState(0);
  const view = views[index];
  const byId = new Map(view.commits.map((commit) => [commit.id, commit]));
  const height = view.commits.some((commit) => commit.lane === 'old') ? 244 : 160;

  return (
    <figure className="git-diagram">
      <div className="git-diagram-tabs" role="group" aria-label="Choose what to show">
        {views.map((item, itemIndex) => (
          <button
            type="button"
            key={item.name}
            className={itemIndex === index ? 'active' : ''}
            aria-pressed={itemIndex === index}
            onClick={() => setIndex(itemIndex)}
          >
            {item.name}
          </button>
        ))}
      </div>
      <div className="git-diagram-canvas">
        <svg viewBox={`0 0 540 ${height}`} role="img" aria-label={`${view.name}: ${view.summary}`}>
          {view.commits.some((commit) => commit.lane === 'old') && (
            <text x={columnX(2)} y={laneY.old + 36} className="git-diagram-note">
              Original C, D, E: no longer on your branch
            </text>
          )}
          {view.edges.map(([from, to]) => {
            const a = byId.get(from);
            const b = byId.get(to);
            if (a === undefined || b === undefined) {
              return null;
            }
            const start = center(a);
            const end = center(b);
            const midX = (start.x + end.x) / 2;
            const ghost = a.kind === 'ghost' || b.kind === 'ghost';
            return (
              <path
                key={`${from}-${to}`}
                className={`git-edge ${ghost ? 'ghost' : ''}`}
                d={`M${start.x} ${start.y} C${midX} ${start.y} ${midX} ${end.y} ${end.x} ${end.y}`}
              />
            );
          })}
          {view.commits.map((commit) => {
            const x = columnX(commit.column);
            const y = laneY[commit.lane] - size / 2;
            return (
              <g key={commit.id} className={`git-commit ${commit.kind}`}>
                <rect x={x} y={y} width={width(commit)} height={size} rx={8} />
                <text x={x + width(commit) / 2} y={y + size / 2 + 4.5} textAnchor="middle">
                  {commit.label}
                </text>
              </g>
            );
          })}
          {view.tags.map((tag) => {
            const commit = byId.get(tag.at);
            if (commit === undefined) {
              return null;
            }
            const x = columnX(commit.column) + width(commit) / 2;
            const above = commit.lane === 'main';
            const y = above ? laneY.main - size / 2 - 10 : laneY[commit.lane] + size / 2 + 18;
            return (
              <text key={tag.label} x={x} y={y} textAnchor="middle" className="git-tag">
                {tag.label}
              </text>
            );
          })}
        </svg>
      </div>
      <figcaption>
        <strong>{view.summary}</strong> {view.description}
      </figcaption>
      <ul className="git-diagram-legend">
        {legend.map((item) => (
          <li key={item.kind}>
            <svg viewBox="0 0 18 18" aria-hidden="true" className={`git-commit ${item.kind}`}>
              <rect x={1.5} y={1.5} width={15} height={15} rx={4} />
            </svg>
            {item.label}
          </li>
        ))}
      </ul>
    </figure>
  );
}
