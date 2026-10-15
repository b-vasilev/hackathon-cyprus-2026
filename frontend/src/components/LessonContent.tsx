import { AlertTriangle, Lightbulb } from 'lucide-react';
import { GitHistoryDiagram } from './GitHistoryDiagram';
import type { Block } from '../foundations/types';

// Renders `backtick` spans as code; everything else stays plain text.
export function Inline({ text }: Readonly<{ text: string }>) {
  return (
    <>
      {text.split(/(`[^`]+`)/).map((part, index) =>
        part.startsWith('`') && part.endsWith('`') && part.length > 1 ? (
          // eslint-disable-next-line react/no-array-index-key -- parts of a fixed string
          <code key={index}>{part.slice(1, -1)}</code>
        ) : (
          part
        )
      )}
    </>
  );
}

export function LessonBlocks({ blocks }: Readonly<{ blocks: Block[] }>) {
  return (
    <div className="lesson-body">
      {blocks.map((block, index) => (
        // eslint-disable-next-line react/no-array-index-key -- static content, never reordered
        <LessonBlock key={index} block={block} />
      ))}
    </div>
  );
}

function LessonBlock({ block }: Readonly<{ block: Block }>) {
  switch (block.type) {
    case 'p':
      return (
        <p>
          <Inline text={block.text} />
        </p>
      );

    case 'list': {
      const items = block.items.map((item) => (
        <li key={item}>
          <Inline text={item} />
        </li>
      ));
      return block.ordered ? <ol>{items}</ol> : <ul>{items}</ul>;
    }
    case 'code':
      return (
        <figure className="lesson-code">
          <pre>
            <code>{block.code}</code>
          </pre>
          {block.caption !== undefined && (
            <figcaption>
              <Inline text={block.caption} />
            </figcaption>
          )}
        </figure>
      );

    case 'callout': {
      const Icon = block.tone === 'warning' ? AlertTriangle : Lightbulb;
      return (
        <aside className={`lesson-callout ${block.tone}`}>
          <Icon size={18} aria-hidden="true" />
          <div>
            <strong>{block.title}</strong>
            <p>
              <Inline text={block.text} />
            </p>
          </div>
        </aside>
      );
    }
    case 'table':
      return (
        <div className="lesson-table">
          <table>
            <thead>
              <tr>
                {block.columns.map((column) => (
                  <th key={column} scope="col">
                    {column}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {block.rows.map((row) => (
                <tr key={row[0]}>
                  {row.map((cell, index) => (
                    // eslint-disable-next-line react/no-array-index-key -- cells are positional
                    <td key={index}>
                      <Inline text={cell} />
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      );
    case 'git-history':
      return <GitHistoryDiagram />;
  }
}
