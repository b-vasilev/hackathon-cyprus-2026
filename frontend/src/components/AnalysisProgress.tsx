import { Boxes, Check } from 'lucide-react';
import { duration } from '../lib/generation';
import { generationMessage } from '../lib/generationMessages';
import type { Run } from '../lib/api';
import './AnalysisProgress.css';

const synthesisSteps = [
  {
    id: 'architecture-synthesis',
    title: 'Architecture from research',
    output: 'Components, systems and contracts',
  },
  {
    id: 'passport-synthesis',
    title: 'Passport flows and supporting facts',
    output: 'Main scenarios, data and operations',
  },
  {
    id: 'final-reconciliation',
    title: 'Reconcile and finalize',
    output: 'References, citations and final model checks',
  },
];

type Analysis = Omit<NonNullable<Run['analysis']>, 'validation_issues'> & {
  validation_issues?: number;
};

function taskStatus(status: string, running: boolean): string {
  if (status === 'done') {
    return 'Completed';
  }
  if (status === 'active') {
    return running ? 'Running' : 'Stopped';
  }
  return 'Waiting';
}

export function AnalysisProgress({
  analysis,
  events,
  stage,
  running,
  now,
}: Readonly<{
  analysis: Analysis;
  events: Run['events'];
  stage: string;
  running: boolean;
  now: number;
}>) {
  const { area_title: area, action, checkpoint_revision: checkpointRevision } = analysis;
  const pastAnalysis = [
    'synthesizing',
    'composing',
    'validating',
    'repairing',
    'complete',
  ].includes(stage);
  const currentIndex = synthesisSteps.findIndex((task) => task.id === analysis.area_id);
  // Old workers already run this exact sequential plan, but do not emit tasks.
  // Do not infer a plan for older, repository-specific analysis queues.
  const tasks =
    analysis.tasks ??
    (currentIndex === -1
      ? []
      : synthesisSteps.map((task, index) => {
          let status = 'pending';
          if (pastAnalysis || index < currentIndex) {
            status = 'done';
          } else if (index === currentIndex) {
            status = 'active';
          }
          return { ...task, status };
        }));
  const completed = tasks.filter((task) => task.status === 'done').length;
  const remaining = tasks.length - completed;
  const lastSave = events.findLast((event) => {
    if (event.type !== 'analysis_progress') {
      return false;
    }
    try {
      const saved: unknown = JSON.parse(event.message);
      return (
        typeof saved === 'object' &&
        saved !== null &&
        'action' in saved &&
        saved.action === 'Saved analysis checkpoint.'
      );
    } catch {
      return false;
    }
  });
  const lastSavedAt = analysis.last_saved_at ?? lastSave?.created_at;
  return (
    <section className="analysis-progress" aria-label="Analysis progress">
      <div className="analysis-heading">
        <h3>
          <Boxes size={18} aria-hidden="true" /> Product model
        </h3>
        <span className="meta">
          {checkpointRevision > 0
            ? `Checkpoint ${checkpointRevision} saved`
            : 'No model checkpoint saved yet'}
        </span>
      </div>
      {tasks.length > 0 && (
        <>
          <p className="analysis-plan-summary" role="status">
            <strong>
              {completed} of {tasks.length} analysis steps completed
            </strong>
            <span>
              {remaining === 0 ? 'Analysis complete' : `${remaining} analysis steps remaining`}
            </span>
          </p>
          <ol className="analysis-steps" aria-label="Analysis steps">
            {tasks.map((task, index) => (
              <li
                key={task.id}
                data-state={task.status}
                aria-current={task.status === 'active' ? 'step' : undefined}
              >
                <span className="analysis-step-marker" aria-hidden="true">
                  {task.status === 'done' ? <Check size={14} /> : index + 1}
                </span>
                <div>
                  <strong>{task.title}</strong>
                  <span>{synthesisSteps.find((step) => step.id === task.id)?.output}</span>
                  <span>{taskStatus(task.status, running)}</span>
                </div>
              </li>
            ))}
          </ol>
        </>
      )}
      {lastSavedAt !== undefined && (
        <p className="analysis-note">Last checkpoint saved {duration(lastSavedAt, now)} ago.</p>
      )}
      <div className="analysis-focus" role="status">
        <span className="analysis-label">{running ? 'Current area' : 'Last area'}</span>
        <strong>{area === '' ? 'Preparing the product model' : area}</strong>
        <span className="analysis-label">
          {running ? 'Current action' : 'Last recorded action'}
        </span>
        <p>{action === '' ? 'Waiting for the next recorded action.' : generationMessage(action)}</p>
      </div>
      <dl className="analysis-counts" aria-label="Persisted model counts">
        {[
          { label: 'Objects', value: analysis.objects },
          { label: 'Claims', value: analysis.claims },
          { label: 'Relations', value: analysis.relations },
        ].map(({ label, value }) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value.toLocaleString('en-GB')}</dd>
          </div>
        ))}
      </dl>
      {analysis.validation_issues !== undefined && (
        <p className="analysis-note">
          {analysis.validation_issues} model checks to resolve before finalization.
        </p>
      )}
      <p className="analysis-note">
        Objects are saved output, not a completion percentage. Passport sections and diagrams follow
        analysis.
      </p>
    </section>
  );
}
