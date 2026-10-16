import { DiagramViewer } from './DiagramViewer';
import { ClaimList, ObjectTable, Scenarios } from './PassportContent';
import { SafeMarkdown } from './SafeMarkdown';
import type { Chapter, DocumentBlock, ProductObject, Revision } from '../lib/api';
import type { ReactNode } from 'react';

interface ContentProps {
  revision: Revision;
  onObject: (object: ProductObject) => void;
  onEvidence: (id: string) => void;
}

function DocumentBlockContent({
  block,
  revision,
  onObject,
  onEvidence,
}: Readonly<ContentProps & { block: DocumentBlock }>) {
  const { model, views } = revision.content;
  switch (block.kind) {
    case 'markdown':
      return <SafeMarkdown>{block.markdown ?? ''}</SafeMarkdown>;
    case 'diagram_ref': {
      const diagram = views.diagrams?.find((item) => item.id === block.reference_id);
      return diagram === undefined ? (
        <p className="notice">This diagram is unavailable in this revision.</p>
      ) : (
        <DiagramViewer diagram={diagram} model={model} onSelect={onObject} />
      );
    }
    case 'scenario_ref': {
      const scenario = model.scenarios?.find((item) => item.id === block.reference_id);
      return scenario === undefined ? (
        <p className="notice">This flow is unavailable in this revision.</p>
      ) : (
        <Scenarios
          scenarios={[scenario]}
          revision={revision}
          onObject={onObject}
          onEvidence={onEvidence}
        />
      );
    }
    case 'object_table': {
      const objectsById = new Map((model.objects ?? []).map((object) => [object.id, object]));
      const objects = (block.object_ids ?? []).flatMap((id) => {
        const object = objectsById.get(id);
        return object === undefined ? [] : [object];
      });
      return (
        <ObjectTable
          sectionId={block.table_style ?? 'objects'}
          objects={objects}
          model={model}
          onObject={onObject}
        />
      );
    }
    default:
      return <p className="notice">This content needs a newer passport viewer.</p>;
  }
}

export function DocumentContent({
  chapter,
  action,
  ...props
}: Readonly<ContentProps & { chapter: Chapter; action?: ReactNode }>) {
  const claimsById = new Map(
    (props.revision.content.model.claims ?? []).map((claim) => [claim.id, claim])
  );
  return (
    <section className="passport-section document-chapter" aria-labelledby="passport-chapter-title">
      <div className="section-title">
        <h2 id="passport-chapter-title" tabIndex={-1}>
          {chapter.title}
        </h2>
        {action}
      </div>
      {chapter.blocks?.map((block, index) => {
        const claims = (block.claim_ids ?? []).flatMap((id) => {
          const claim = claimsById.get(id);
          return claim === undefined ? [] : [claim];
        });
        return (
          // eslint-disable-next-line react/no-array-index-key -- Immutable blocks have no independent IDs; their authored position identifies them.
          <div className="document-block" key={`${chapter.id}:${index}`}>
            <DocumentBlockContent block={block} {...props} />
            {claims.length > 0 && (
              <details className="section-evidence block-evidence">
                <summary>Supporting evidence ({claims.length})</summary>
                <ClaimList claims={claims} onEvidence={props.onEvidence} />
              </details>
            )}
          </div>
        );
      })}
    </section>
  );
}
