import { Boxes } from 'lucide-react';
import { generationMessage } from '../lib/generationMessages';
import './AnalysisProgress.css';

export function AnalysisProgress({
  area,
  action,
  step,
  checkpointRevision,
  counts,
  running,
}: Readonly<{
  area: string;
  action: string;
  step: number;
  checkpointRevision: number;
  counts: { objects: number; claims: number; relations: number };
  running: boolean;
}>) {
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
          { label: 'Objects', value: counts.objects },
          { label: 'Claims', value: counts.claims },
          { label: 'Relations', value: counts.relations },
        ].map(({ label, value }) => (
          <div key={label}>
            <dt>{label}</dt>
            <dd>{value.toLocaleString('en-GB')}</dd>
          </div>
        ))}
      </dl>
      <p className="analysis-note">
        Counts reflect saved model parts, not full product coverage.
        {step > 0 && ` Recorded step ${step}.`}
      </p>
    </section>
  );
}
