import { Check, CircleHelp, Route } from 'lucide-react';
import {
  orderedResearchAreas,
  questionStatusLabels,
  researchCounts,
  researchFocusLabel,
  researchStatusLabels,
} from '../lib/researchProgress';
import type { ResearchArea, ResearchProgress as ResearchState } from '../lib/researchProgress';

function ResearchAreaRow({ area, running }: Readonly<{ area: ResearchArea; running: boolean }>) {
  return (
    <details className="research-area">
      <summary>
        <span className={`research-area-marker ${area.status}`} aria-hidden="true">
          {area.status === 'checked' ? <Check size={12} /> : <CircleHelp size={12} />}
        </span>
        <span className="research-area-name">{area.name}</span>
        <span className="research-area-status">
          {!running && area.status === 'investigating'
            ? 'Partly checked'
            : researchStatusLabels[area.status]}
        </span>
      </summary>
      <div className="research-area-body">
        <p className="meta">{area.repository}</p>
        <ol className="research-questions">
          {area.questions.map((question) => (
            <li key={question.id}>
              <strong>{question.question}</strong>
              <span className="research-question-status">
                {questionStatusLabels[question.status]}
              </span>
              {question.result !== '' && <p>{question.result}</p>}
              {question.evidence.length > 0 && (
                <ul className="research-evidence" aria-label="Source evidence">
                  {question.evidence.map((evidence) => (
                    <li key={`${evidence.document_id}/${evidence.start_line}/${evidence.end_line}`}>
                      {evidence.path}:L{evidence.start_line}–L{evidence.end_line}
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ol>
      </div>
    </details>
  );
}

function ResearchBudget({
  progress,
  running,
}: Readonly<{ progress: ResearchState; running: boolean }>) {
  const budget = progress.budget;
  const decision = budget.decisions.at(-1);
  const ended = !running || progress.status === 'complete' || progress.status === 'partial';
  return (
    <div className="research-budget">
      <div className="research-budget-numbers">
        <span>
          <strong>{budget.used}</strong> turns used
        </span>
        {!ended && <span>Next budget review at {budget.limit}</span>}
        <span>
          {budget.hard_limit > 0 ? `Hard cap ${budget.hard_limit} turns · ` : 'No turn cap · '}
          60 min
        </span>
      </div>
      {decision !== undefined && (
        <details className="research-budget-reason">
          <summary>
            {ended ? 'Why research stopped' : 'Why research continues'}
            {decision.kind === 'extended' && ` · allowance extended to ${decision.limit}`}
            {decision.kind === 'recovery' && ' · one recovery attempt'}
          </summary>
          <p className="meta">
            {decision.turn === 0 ? 'Initial allowance' : `Budget reviewed at turn ${decision.turn}`}
          </p>
          <p>
            {!running && progress.status === 'in_progress'
              ? 'The run ended before research coverage was accepted. The saved plan shows the remaining questions.'
              : decision.reason}
          </p>
          {budget.deadline !== '' && (
            <p className="meta">
              Attempt deadline{' '}
              <time dateTime={budget.deadline}>
                {new Date(budget.deadline).toLocaleTimeString('en-GB')}
              </time>
              . The runner reserves the last 30 seconds for saving results.
            </p>
          )}
        </details>
      )}
    </div>
  );
}

export function ResearchProgress({
  progress,
  running,
}: Readonly<{ progress: ResearchState; running: boolean }>) {
  const counts = researchCounts(progress.areas);
  const statuses = Object.keys(researchStatusLabels) as (keyof typeof researchStatusLabels)[];
  const countLabels = {
    not_started: 'to explore',
    investigating: 'started',
    checked: 'checked',
    blocked: 'source gaps',
    excluded: 'excluded',
  };
  const active = running && progress.status === 'in_progress';
  return (
    <section className="research-progress" aria-label="Research plan">
      <div className="research-heading">
        <h3>
          <Route size={18} aria-hidden="true" /> Research plan
        </h3>
        <span className="meta">
          {counts.answered}/{counts.questions} current questions resolved
        </span>
      </div>
      <div className="research-scope-summary" aria-label="Areas in the current plan">
        {statuses
          .filter((status) => counts[status] > 0)
          .map((status) => (
            <span key={status}>
              <strong>{counts[status]}</strong> {countLabels[status]}
            </span>
          ))}
      </div>
      {progress.parallel !== undefined && progress.parallel.limit > 0 && (
        <div className="research-scope-summary" aria-label="Research agents">
          <span>
            <strong>{active ? progress.parallel.active : 0}</strong> agents active
          </span>
          <span>
            <strong>{progress.parallel.queued}</strong> tasks queued
          </span>
          <span>Up to {progress.parallel.limit} parallel agents</span>
        </div>
      )}
      {active && progress.parallel !== undefined && progress.parallel.workers.length > 0 && (
        <details className="research-plan-details">
          <summary>View active agents ({progress.parallel.workers.length})</summary>
          <ol className="research-questions">
            {progress.parallel.workers.map((worker) => (
              <li key={worker.id}>
                <strong>{worker.task}</strong>
                {worker.focus.question !== '' && <p>{worker.focus.question}</p>}
                {worker.activity !== '' && <span className="meta">{worker.activity}</span>}
              </li>
            ))}
          </ol>
        </details>
      )}
      {active && (
        <div className="research-focus" role="status">
          <span className="research-label">
            {progress.focus.area === '' ? 'Getting oriented' : researchFocusLabel(progress)}
          </span>
          <p>
            {progress.focus.question === ''
              ? 'Mapping the supplied source areas and choosing the first question.'
              : progress.focus.question}
          </p>
          {progress.focus.reason !== '' && <span className="meta">{progress.focus.reason}</span>}
        </div>
      )}
      {progress.last_result !== '' && (
        <div className="research-last-result">
          <span className="research-label">Latest saved result</span>
          <p>{progress.last_result.replaceAll(/\[[A-Z0-9]{20,}:L\d+-L\d+\]/g, '').trim()}</p>
        </div>
      )}
      <details className="research-plan-details">
        <summary>View areas, questions and results ({progress.areas.length})</summary>
        <div className="research-plan-body">
          <p className="research-scope-note">
            The plan can grow during discovery. Checked means read and cited source evidence, not
            exhaustive coverage or executed tests.
          </p>
          <div className="research-area-list" role="region" aria-label="Research areas">
            {orderedResearchAreas(progress).map((area) => (
              <ResearchAreaRow area={area} running={active} key={area.id} />
            ))}
            {progress.areas.length === 0 && <p className="meta">No source areas indexed yet.</p>}
          </div>
        </div>
      </details>
      <ResearchBudget progress={progress} running={running} />
    </section>
  );
}
