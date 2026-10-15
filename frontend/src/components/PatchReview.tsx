import { Flag, MessageSquare } from 'lucide-react';
import { useState } from 'react';
import type { PatchQuestion } from '../foundations/types';

const signs = { add: '+', remove: '-', context: ' ' };

export function PatchReview({
  question,
  onAnswered,
}: Readonly<{ question: PatchQuestion; onAnswered: (correct: boolean) => void }>) {
  const [flagged, setFlagged] = useState<number[]>([]);
  const [submitted, setSubmitted] = useState(false);
  const issues = question.lines.flatMap((line, index) => (line.issue === undefined ? [] : [index]));
  const found = issues.filter((index) => flagged.includes(index)).length;
  const extra = flagged.filter((index) => !issues.includes(index)).length;
  const extraLines = extra === 1 ? '1 line that was' : `${extra} lines that were`;
  const extraNote = extra > 0 ? ` and flagged ${extraLines} fine` : '';

  function toggle(index: number) {
    setFlagged((current) =>
      current.includes(index) ? current.filter((item) => item !== index) : [...current, index]
    );
  }

  function submit() {
    setSubmitted(true);
    onAnswered(found === issues.length && extra === 0);
  }

  return (
    <>
      <div className="patch">
        <div className="patch-file">{question.file}</div>
        {question.lines.map((line, index) => {
          const isFlagged = flagged.includes(index);
          let state = '';
          if (submitted && line.issue !== undefined) {
            state = isFlagged ? 'found' : 'missed';
          } else if (submitted && isFlagged) {
            state = 'extra';
          }
          return (
            // eslint-disable-next-line react/no-array-index-key -- diff lines are positional
            <div key={index} className={`patch-row ${line.type} ${state}`}>
              <button
                type="button"
                className={`patch-line ${isFlagged ? 'flagged' : ''}`}
                disabled={submitted}
                aria-pressed={isFlagged}
                aria-label={`Line ${index + 1}${isFlagged ? ', flagged' : ''}`}
                onClick={() => toggle(index)}
              >
                <span className="patch-flag" aria-hidden="true">
                  {isFlagged && <Flag size={13} />}
                </span>
                <code>
                  {signs[line.type]} {line.text}
                </code>
              </button>
              {submitted && line.issue !== undefined && (
                <p className="patch-comment">
                  <MessageSquare size={14} aria-hidden="true" />
                  <span>
                    <strong>{isFlagged ? 'You caught it.' : 'Missed.'}</strong> {line.issue}
                  </span>
                </p>
              )}
              {state === 'extra' && (
                <p className="patch-comment">
                  <MessageSquare size={14} aria-hidden="true" />
                  <span>
                    <strong>This line is fine.</strong>
                  </span>
                </p>
              )}
            </div>
          );
        })}
      </div>
      {submitted ? (
        <div
          className={`quiz-feedback ${found === issues.length && extra === 0 ? '' : 'wrong'}`}
          role="status"
        >
          You found {found} of {issues.length} problems
          {extraNote}.
        </div>
      ) : (
        <div className="form-actions">
          <span className="patch-count">{flagged.length} flagged</span>
          <button
            type="button"
            className="button primary"
            disabled={flagged.length === 0}
            onClick={submit}
          >
            Submit review
          </button>
        </div>
      )}
    </>
  );
}
