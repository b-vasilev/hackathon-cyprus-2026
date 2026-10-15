import { useState } from 'react';
import { Inline } from './Inline';
import type { ScenesBlock } from '../foundations/types';

// Tabbed picture: each tab shows rows of connected coloured blocks and a caption.
export function SceneDiagram({ block }: Readonly<{ block: ScenesBlock }>) {
  const [index, setIndex] = useState(0);
  const scene = block.scenes[Math.min(index, block.scenes.length - 1)];

  return (
    <figure className="git-diagram scene-diagram">
      {block.title !== undefined && (
        <p className="scene-title">
          <Inline text={block.title} />
        </p>
      )}
      {block.scenes.length > 1 && (
        <div className="git-diagram-tabs" role="group" aria-label="Choose what to show">
          {block.scenes.map((item, itemIndex) => (
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
      )}
      <div className={`scene-canvas ${block.layout ?? 'flow'}`}>
        {scene.rows.map((row, rowIndex) => (
          // eslint-disable-next-line react/no-array-index-key -- rows are positional
          <div className="scene-row" key={`${scene.name}-${rowIndex}`}>
            {row.label !== undefined && <span className="scene-label">{row.label}</span>}
            <ol className="scene-blocks">
              {row.blocks.map((item, blockIndex) => (
                // eslint-disable-next-line react/no-array-index-key -- blocks are positional
                <li key={blockIndex} className={`scene-block tone-${item.tone ?? 'neutral'}`}>
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
            </ol>
          </div>
        ))}
      </div>
      <figcaption>
        <strong>
          <Inline text={scene.summary} />
        </strong>
        {scene.description !== undefined && (
          <>
            {' '}
            <Inline text={scene.description} />
          </>
        )}
      </figcaption>
      {block.legend !== undefined && (
        <ul className="git-diagram-legend">
          {block.legend.map((item) => (
            <li key={item.label}>
              <span className={`scene-swatch tone-${item.tone}`} aria-hidden="true" />
              <Inline text={item.label} />
            </li>
          ))}
        </ul>
      )}
    </figure>
  );
}
