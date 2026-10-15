import { ArrowRight, Check, X } from 'lucide-react';
import { useState } from 'react';
import { Inline } from './LessonContent';
import type { BranchChoice, BranchQuestion } from '../foundations/types';

interface Taken {
  stepId: string;
  choice: BranchChoice;
}

export function BranchScenario({
  question,
  onAnswered,
}: Readonly<{ question: BranchQuestion; onAnswered: (correct: boolean) => void }>) {
  const [stepId, setStepId] = useState(question.steps[0].id);
  const [path, setPath] = useState<Taken[]>([]);
  const step = question.steps.find((item) => item.id === stepId) ?? question.steps[0];
  const current = path.find((item) => item.stepId === step.id);
  const finished = current !== undefined && current.choice.next === undefined;

  function choose(choice: BranchChoice) {
    const taken = [...path, { stepId: step.id, choice }];
    setPath(taken);
    if (choice.next === undefined) {
      onAnswered(taken.every((item) => item.choice.good));
    }
  }

  return (
    <div className="branch">
      <ol className="branch-trail">
        {path.map((item) => {
          const past = question.steps.find((candidate) => candidate.id === item.stepId);
          return (
            <li key={item.stepId} className={item.choice.good ? 'good' : 'bad'}>
              <span className="branch-dot" aria-hidden="true">
                {item.choice.good ? <Check size={12} /> : <X size={12} />}
              </span>
              <div>
                {past !== undefined && (
                  <p className="branch-past">
                    <Inline text={past.text} />
                  </p>
                )}
                <p className="branch-choice">
                  You: <Inline text={item.choice.label} />
                </p>
                <p className="branch-outcome">
                  <Inline text={item.choice.outcome} />
                </p>
              </div>
            </li>
          );
        })}
      </ol>
      {current === undefined && (
        <div className="branch-step">
          <p>
            <Inline text={step.text} />
          </p>
          <div className="branch-choices">
            {step.choices.map((choice) => (
              <button
                type="button"
                key={choice.label}
                className="button secondary"
                onClick={() => choose(choice)}
              >
                <Inline text={choice.label} />
              </button>
            ))}
          </div>
        </div>
      )}
      {current?.choice.next !== undefined && (
        <button
          type="button"
          className="text-button"
          onClick={() => setStepId(current.choice.next ?? step.id)}
        >
          What happens next
          <ArrowRight size={15} />
        </button>
      )}
      {finished && (
        <div
          className={`quiz-feedback ${path.every((item) => item.choice.good) ? '' : 'wrong'}`}
          role="status"
        >
          {path.every((item) => item.choice.good)
            ? 'Good calls all the way through.'
            : 'You got there, but not every call was a good one. The ✕ steps show where it went off track.'}
        </div>
      )}
    </div>
  );
}
