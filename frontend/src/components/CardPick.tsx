import {
  Check,
  FileCode2,
  GitBranch,
  GitCommitHorizontal,
  GitPullRequest,
  MessageSquare,
  X,
} from 'lucide-react';
import { useState } from 'react';
import { Inline } from './LessonContent';
import type { Card, CardQuestion } from '../foundations/types';

const icons = {
  'pull-request': GitPullRequest,
  commit: GitCommitHorizontal,
  branch: GitBranch,
  file: FileCode2,
};

// Wraps the first occurrence of flaw in a mark once the card has been rejected.
function Flawed({ text, flaw }: Readonly<{ text: string; flaw?: string }>) {
  const at = flaw === undefined ? -1 : text.indexOf(flaw);
  if (flaw === undefined || at === -1) {
    return <Inline text={text} />;
  }
  return (
    <>
      <Inline text={text.slice(0, at)} />
      <mark className="card-flaw">{flaw}</mark>
      <Inline text={text.slice(at + flaw.length)} />
    </>
  );
}

export function CardPick({
  question,
  onAnswered,
}: Readonly<{ question: CardQuestion; onAnswered: (correct: boolean) => void }>) {
  const [picked, setPicked] = useState<number[]>([]);
  const solved = picked.some((index) => question.cards[index].correct === true);
  const Icon = icons[question.look];

  function pick(index: number, card: Card) {
    if (solved || picked.includes(index)) {
      return;
    }
    setPicked((current) => [...current, index]);
    if (card.correct === true) {
      onAnswered(picked.length === 0);
    }
  }

  return (
    <div className={`card-pick ${question.look}`}>
      {question.cards.map((card, index) => {
        const revealed = picked.includes(index);
        const flaw = revealed && card.correct !== true ? card.flaw : undefined;
        let state = '';
        if (revealed) {
          state = card.correct === true ? 'right' : 'wrong';
        }
        return (
          <button
            type="button"
            // eslint-disable-next-line react/no-array-index-key -- cards are static and may share a title
            key={index}
            className={`mock-card ${state}`}
            disabled={solved && !revealed}
            aria-pressed={revealed}
            onClick={() => pick(index, card)}
          >
            <span className="mock-card-head">
              <Icon size={16} aria-hidden="true" />
              <span className="mock-card-title">
                <Flawed text={card.title} flaw={flaw} />
              </span>
            </span>
            {card.meta !== undefined && (
              <span className="mock-card-meta">
                <Flawed text={card.meta} flaw={flaw} />
              </span>
            )}
            {card.body !== undefined && (
              <span className="mock-card-body">
                <Flawed text={card.body} flaw={flaw} />
              </span>
            )}
            {revealed && (
              <span className="mock-card-comment" role="status">
                <span className="mock-card-verdict">
                  {card.correct === true ? (
                    <>
                      <Check size={15} aria-hidden="true" /> That&apos;s the one
                    </>
                  ) : (
                    <>
                      <X size={15} aria-hidden="true" /> Not this one
                    </>
                  )}
                </span>
                <span>
                  <MessageSquare size={14} aria-hidden="true" />
                  <span>
                    <Inline text={card.feedback} />
                  </span>
                </span>
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
