import { useQuery } from '@tanstack/react-query';
import { ArrowRight, FileCode2, Quote } from 'lucide-react';
import { useState } from 'react';
import {
  api,
  type Claim,
  type EvidenceView,
  type ProductModel,
  type ProductObject,
  type Revision,
  type Scenario,
  type Section,
} from '../lib/api';
import { DiagramViewer } from './DiagramViewer';
import { Badge, Dialog, ErrorNotice, ExternalLink, Loading } from './ui';

const basisLabels: Record<string, string> = {
  source_backed: 'Source-backed',
  inferred: 'Interpretation',
  unknown: 'Unknown',
};

export function ClaimList({
  claims,
  onEvidence,
}: Readonly<{
  claims: Claim[];
  onEvidence: (id: string) => void;
}>) {
  return (
    <div className="claim-list">
      {claims.map((claim) => (
        <div className="claim" key={claim.id}>
          <div className="claim-label">
            <Quote size={14} aria-hidden="true" />
            <Badge tone={claim.basis === 'source_backed' ? '' : 'warning'}>
              {basisLabels[claim.basis] ?? claim.basis}
            </Badge>
          </div>
          <p>{claim.text}</p>
          <div className="evidence-links">
            {claim.evidence_ids?.map((id, index) => (
              <button type="button" className="source-link" key={id} onClick={() => onEvidence(id)}>
                <FileCode2 size={13} />
                Source {index + 1}
              </button>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

export function ObjectDetail({
  object,
  model,
  onClose,
  onEvidence,
}: Readonly<{
  object: ProductObject;
  model: ProductModel;
  onClose: () => void;
  onEvidence: (id: string) => void;
}>) {
  const claims = (model.claims ?? []).filter((claim) => object.claim_ids?.includes(claim.id));
  return (
    <Dialog title={object.name} onClose={onClose}>
      <div className="detail-meta">
        <Badge>{object.kind.replaceAll('_', ' ')}</Badge>
        <Badge>{object.status.replaceAll('_', ' ')}</Badge>
        {object.technology !== '' && <code>{object.technology}</code>}
      </div>
      <p className="detail-summary">{object.summary}</p>
      {[
        { title: 'Inputs', items: object.inputs },
        { title: 'Outputs', items: object.outputs },
        { title: 'Contracts and boundaries', items: object.contracts },
      ].map(
        ({ title, items }) =>
          (items?.length ?? 0) > 0 && (
            <div className="detail-section" key={title}>
              <h3>{title}</h3>
              <ul>
                {items?.map((item) => (
                  <li key={item}>{item}</li>
                ))}
              </ul>
            </div>
          )
      )}
      <div className="detail-section">
        <h3>Supporting claims</h3>
        <ClaimList claims={claims} onEvidence={onEvidence} />
      </div>
    </Dialog>
  );
}

export function EvidenceDialog({
  revisionId,
  evidenceId,
  onClose,
}: Readonly<{
  revisionId: string;
  evidenceId: string;
  onClose: () => void;
}>) {
  const evidence = useQuery({
    queryKey: ['evidence', revisionId, evidenceId],
    queryFn: ({ signal }) =>
      api<EvidenceView>(`/passport-revisions/${revisionId}/evidence/${evidenceId}`, { signal }),
  });
  return (
    <Dialog title="Source evidence" wide onClose={onClose}>
      {evidence.isPending && <Loading />}
      {evidence.isError && <ErrorNotice error={evidence.error} />}
      {evidence.data !== undefined && (
        <>
          <div className="evidence-heading">
            <FileCode2 size={18} aria-hidden="true" />
            <strong>{evidence.data.path}</strong>
            <Badge>
              Lines {evidence.data.start_line}–{evidence.data.end_line}
            </Badge>
          </div>
          <p className="meta">
            {evidence.data.locator} · revision <code>{evidence.data.revision.slice(0, 12)}</code>
          </p>
          <pre className="source-code">
            {evidence.data.excerpt.split('\n').map((line, index) => (
              <span className="code-line" key={index}>
                <span className="line-number">{evidence.data.start_line + index}</span>
                {line}
                {'\n'}
              </span>
            ))}
          </pre>
          {evidence.data.source_url !== '' && (
            <ExternalLink href={evidence.data.source_url}>Open this source revision</ExternalLink>
          )}
          <p className="field-help">
            This is the collected source text. A source pointer supports inspection; it does not
            establish expert approval.
          </p>
        </>
      )}
    </Dialog>
  );
}

export function SectionContent({
  section,
  revision,
  onObject,
  onEvidence,
}: Readonly<{
  section: Section;
  revision: Revision;
  onObject: (object: ProductObject) => void;
  onEvidence: (id: string) => void;
}>) {
  const model = revision.content.model;
  const objects = (model.objects ?? []).filter((object) => section.object_ids?.includes(object.id));
  const claims = (model.claims ?? []).filter((claim) => section.claim_ids?.includes(claim.id));
  const diagrams = (revision.content.views.diagrams ?? []).filter((diagram) =>
    section.diagram_ids?.includes(diagram.id)
  );
  const scenarios = (model.scenarios ?? []).filter((scenario) =>
    section.scenario_ids?.includes(scenario.id)
  );
  return (
    <section className="passport-section" aria-label={section.title}>
      <div className="section-title">
        <p className="eyebrow">Product passport</p>
        <h2>{section.title}</h2>
        {section.status !== 'available' && (
          <Badge tone="warning">{section.status.replaceAll('_', ' ')}</Badge>
        )}
      </div>
      <p className="section-summary">{section.summary}</p>
      {diagrams.map((diagram) => (
        <DiagramViewer key={diagram.id} diagram={diagram} model={model} onSelect={onObject} />
      ))}
      {scenarios.length > 0 && (
        <Scenarios
          scenarios={scenarios}
          revision={revision}
          onObject={onObject}
          onEvidence={onEvidence}
        />
      )}
      {objects.length > 0 && (
        <div className="object-grid">
          {objects.map((object) => (
            <button
              type="button"
              className="object-card"
              onClick={() => onObject(object)}
              key={object.id}
            >
              <span className="eyebrow">{object.kind.replaceAll('_', ' ')}</span>
              <h3>{object.name}</h3>
              <p>{object.summary}</p>
              <div className="object-card-footer">
                <Badge>{object.status.replaceAll('_', ' ')}</Badge>
                <ArrowRight size={16} />
              </div>
            </button>
          ))}
        </div>
      )}
      <ClaimList claims={claims} onEvidence={onEvidence} />
      {(section.checklist?.length ?? 0) > 0 && (
        <div className="checklist">
          <h3>Practical checkpoints</h3>
          <ul>
            {section.checklist?.map((item) => (
              <li key={item}>
                <span className="checkpoint" aria-hidden="true" />
                {item}
              </li>
            ))}
          </ul>
        </div>
      )}
      {section.id === 'ownership' && (model.open_questions?.length ?? 0) > 0 && (
        <div className="open-questions">
          <h3>Questions for the product owner</h3>
          <ol>
            {model.open_questions?.map((question) => (
              <li key={question}>{question}</li>
            ))}
          </ol>
        </div>
      )}
    </section>
  );
}

function Scenarios({
  scenarios,
  revision,
  onObject,
  onEvidence,
}: Readonly<{
  scenarios: Scenario[];
  revision: Revision;
  onObject: (object: ProductObject) => void;
  onEvidence: (id: string) => void;
}>) {
  const [selectedId, setSelectedId] = useState(scenarios.at(0)?.id ?? '');
  const [stepId, setStepId] = useState('');
  const scenario = scenarios.find((item) => item.id === selectedId) ?? scenarios.at(0);
  if (scenario === undefined) {
    return null;
  }
  const step = scenario.steps?.find((item) => item.id === stepId) ?? scenario.steps?.at(0);
  const claims = (revision.content.model.claims ?? []).filter((claim) =>
    step?.claim_ids?.includes(claim.id)
  );
  const diagram =
    revision.content.views.diagrams?.find((item) => item.kind === 'components') ??
    revision.content.views.diagrams?.at(0);
  return (
    <div className="scenario-view">
      <div className="scenario-tabs">
        {scenarios.map((item) => (
          <button
            type="button"
            className={item.id === scenario.id ? 'active' : ''}
            aria-pressed={item.id === scenario.id}
            onClick={() => {
              setSelectedId(item.id);
              setStepId('');
            }}
            key={item.id}
          >
            {item.title}
          </button>
        ))}
      </div>
      <p>{scenario.summary}</p>
      <div className="scenario-steps">
        {scenario.steps?.map((item, index) => (
          <button
            type="button"
            key={item.id}
            className={`scenario-step ${item.id === step?.id ? 'active' : ''}`}
            aria-pressed={item.id === step?.id}
            onClick={() => setStepId(item.id)}
          >
            <span className="step-number">{String(index + 1).padStart(2, '0')}</span>
            <strong>{item.title}</strong>
            <span>{item.summary}</span>
          </button>
        ))}
      </div>
      {diagram !== undefined && (
        <DiagramViewer
          diagram={diagram}
          model={revision.content.model}
          onSelect={onObject}
          highlighted={step?.object_ids ?? []}
        />
      )}
      <div className="scenario-contract">
        {[
          { title: 'Input', items: scenario.inputs },
          { title: 'Outcome', items: scenario.outputs },
          { title: 'Failure conditions', items: scenario.failure_modes },
        ].map(({ title, items }) => (
          <div key={title}>
            <h3>{title}</h3>
            <ul>
              {items?.map((item) => (
                <li key={item}>{item}</li>
              ))}
            </ul>
          </div>
        ))}
      </div>
      <ClaimList claims={claims} onEvidence={onEvidence} />
    </div>
  );
}
