import { AlertTriangle, Lightbulb } from 'lucide-react';
import { CodeTools } from './CodeTools';
import { GitHistoryDiagram } from './GitHistoryDiagram';
import { Inline } from './Inline';
import { SceneDiagram } from './SceneDiagram';
import type { Block } from '../foundations/types';

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
    case 'scenes':
      return <SceneDiagram block={block} />;
    case 'code-tools':
      return <CodeTools block={block} />;
  }
}
