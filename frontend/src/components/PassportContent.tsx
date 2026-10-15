import { useQuery } from '@tanstack/react-query';
import { ArrowRight, FileCode2, PanelRightOpen, Quote } from 'lucide-react';
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
import { objectTypeLabel } from '../lib/passportDocument';
import { sectionTitles } from '../lib/passportSections';
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
              <button
                type="button"
                className="source-link source-action"
                key={id}
                onClick={() => onEvidence(id)}
              >
                <FileCode2 size={16} aria-hidden="true" />
                View source {index + 1}
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
        <Badge>{objectTypeLabel(object)}</Badge>
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
            {evidence.data.excerpt
              .split('\n')
              .map((text, offset) => ({ text, number: evidence.data.start_line + offset }))
              .map((line) => (
                <span className="code-line" key={line.number}>
                  <span className="line-number">{line.number}</span>
                  {line.text}
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
  const scenarios =
    section.id === 'scenarios'
      ? (model.scenarios ?? []).filter((scenario) => section.scenario_ids?.includes(scenario.id))
      : [];
  const title = sectionTitles[section.id] ?? section.title;
  const tableSection = section.id === 'technologies' || section.id === 'repositories';
  return (
    <section className="passport-section" aria-label={title}>
      <div className="section-title">
        <h2>{title}</h2>
        {section.status !== 'available' && (
          <Badge tone="warning">{section.status.replaceAll('_', ' ')}</Badge>
        )}
      </div>
      <p className="section-summary">{section.summary}</p>
      {section.id === 'overview' &&
        objects.some((item) => item.kind === 'capability' || item.kind === 'actor') && (
          <div className="overview-outcomes">
            <h3>Users and outcomes</h3>
            <ul>
              {objects
                .filter((item) => item.kind === 'capability' || item.kind === 'actor')
                .map((item) => (
                  <li key={item.id}>
                    <button type="button" className="source-link" onClick={() => onObject(item)}>
                      {item.name}
                    </button>
                    <p>{item.summary}</p>
                  </li>
                ))}
            </ul>
          </div>
        )}
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
      {tableSection && (
        <ObjectTable sectionId={section.id} objects={objects} model={model} onObject={onObject} />
      )}
      {objects.length > 0 && !tableSection && section.id !== 'overview' && (
        <div className="object-grid">
          {objects.map((object) => (
            <button
              type="button"
              className="object-card"
              onClick={() => onObject(object)}
              key={object.id}
            >
              <span className="eyebrow">{objectTypeLabel(object)}</span>
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
      {claims.length > 0 && (
        <details className="section-evidence">
          <summary>Supporting evidence ({claims.length})</summary>
          <ClaimList claims={claims} onEvidence={onEvidence} />
        </details>
      )}
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

export function ObjectTable({
  sectionId,
  objects,
  model,
  onObject,
}: Readonly<{
  sectionId: string;
  objects: ProductObject[];
  model: ProductModel;
  onObject: (object: ProductObject) => void;
}>) {
  const repositories = sectionId === 'repositories';
  const technologies = sectionId === 'technologies';
  const rows = repositories ? objects.filter((object) => object.kind === 'repository') : objects;
  const column = technologies ? 'Technology' : 'Type';
  let caption = 'Objects and their responsibilities';
  let nameColumn = 'Object';
  if (repositories) {
    caption = 'Code ownership and repository responsibilities';
    nameColumn = 'Repository';
  } else if (technologies) {
    caption = 'Technologies and their responsibilities';
    nameColumn = 'Component or tool';
  }
  return (
    <div className="passport-table-scroll">
      <table className="passport-table">
        <caption>{caption}</caption>
        <thead>
          <tr>
            <th scope="col">{nameColumn}</th>
            <th scope="col">{repositories ? 'Owned components' : column}</th>
            <th scope="col">Responsibility</th>
            {repositories && <th scope="col">Repository connections</th>}
            <th scope="col">Status</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((object) => {
            const owned = (model.relations ?? [])
              .filter((relation) => relation.kind === 'ownership' && relation.from === object.id)
              .flatMap((relation) =>
                (model.objects ?? []).filter((item) => item.id === relation.to)
              );
            const connections = (model.relations ?? []).filter(
              (relation) => relation.from === object.id && relation.kind !== 'ownership'
            );
            const technology = object.technology === '' ? 'Not specified' : object.technology;
            return (
              <tr key={object.id}>
                <th scope="row">
                  <button type="button" className="source-link" onClick={() => onObject(object)}>
                    {object.name}
                  </button>
                </th>
                <td>
                  {!repositories && (technologies ? technology : objectTypeLabel(object))}
                  {repositories && owned.length === 0 && 'Ownership not established'}
                  {repositories &&
                    owned.map((item) => (
                      <button
                        type="button"
                        className="source-link"
                        onClick={() => onObject(item)}
                        key={item.id}
                      >
                        {item.name}
                      </button>
                    ))}
                </td>
                <td>{object.summary}</td>
                {repositories && (
                  <td>
                    {connections.flatMap((relation) => {
                      const target = model.objects?.find(
                        (item) => item.id === relation.to && item.kind === 'repository'
                      );
                      return target === undefined
                        ? []
                        : [
                            <div key={relation.id}>
                              <button
                                type="button"
                                className="source-link"
                                onClick={() => onObject(target)}
                              >
                                {target.name}
                              </button>
                              <span>{relation.label}</span>
                            </div>,
                          ];
                    })}
                  </td>
                )}
                <td>
                  <Badge>{object.status.replaceAll('_', ' ')}</Badge>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

export function Scenarios({
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
  const participants = (revision.content.model.objects ?? []).filter((object) =>
    step?.object_ids?.includes(object.id)
  );
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
      {participants.length > 0 && (
        <div className="scenario-participants">
          <h3>Components in this step</h3>
          {participants.map((object) => (
            <button
              type="button"
              className="source-link source-action"
              key={object.id}
              onClick={() => onObject(object)}
            >
              {object.name}
              <PanelRightOpen size={16} aria-hidden="true" />
            </button>
          ))}
        </div>
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
