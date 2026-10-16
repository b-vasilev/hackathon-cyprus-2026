import { Fragment, useMemo, useState, type DragEvent } from 'react';
import { Inline } from './Inline';
import { Feedback } from './QuizFeedback';
import type { FillQuestion } from '../foundations/types';

interface Chip {
  id: number;
  text: string;
}

// Stable order that does not reveal which chips are answers.
function shuffled(question: FillQuestion): Chip[] {
  const chips = [...question.answers, ...question.distractors].map((text, id) => ({ id, text }));
  const weight = (chip: Chip) => {
    const seed = `${question.id}${chip.text}${chip.id}`;
    let hash = 2_166_136_261;
    for (let index = 0; index < seed.length; index++) {
      hash = Math.imul(hash ^ (seed.codePointAt(index) ?? 0), 16_777_619);
    }
    return hash;
  };
  return chips.toSorted((a, b) => weight(a) - weight(b));
}

export function FillGaps({
  question,
  onAnswered,
}: Readonly<{ question: FillQuestion; onAnswered: (correct: boolean) => void }>) {
  const chips = useMemo(() => shuffled(question), [question]);
  const [placed, setPlaced] = useState<(Chip | undefined)[]>(() =>
    question.answers.map(() => undefined)
  );
  const [selected, setSelected] = useState<Chip>();
  const [checked, setChecked] = useState(false);
  const used = new Set(placed.map((chip) => chip?.id));
  const results = placed.map(
    (chip, gap) => chip?.text.toLowerCase() === question.answers[gap].toLowerCase()
  );
  const allRight = results.every(Boolean);

  function put(gap: number, chip: Chip) {
    setPlaced((current) =>
      current.map((item, index) => {
        if (index === gap) {
          return chip;
        }
        return item?.id === chip.id ? undefined : item;
      })
    );
    setSelected(undefined);
  }

  function clickGap(gap: number) {
    if (checked) {
      return;
    }
    if (selected !== undefined) {
      put(gap, selected);
    } else if (placed[gap] !== undefined) {
      setPlaced((current) => current.map((item, index) => (index === gap ? undefined : item)));
    }
  }

  function drop(event: DragEvent, gap: number) {
    event.preventDefault();
    const chip = chips.find((item) => String(item.id) === event.dataTransfer.getData('text/plain'));
    if (chip !== undefined && !checked) {
      put(gap, chip);
    }
  }

  function check() {
    setChecked(true);
    onAnswered(allRight);
  }

  const parts = question.template.split(/\{\{(\d+)\}\}/);
  const content = parts.map((part, index) => {
    if (index % 2 === 0) {
      return (
        // eslint-disable-next-line react/no-array-index-key -- template parts are positional
        <Fragment key={index}>{question.code === true ? part : <Inline text={part} />}</Fragment>
      );
    }
    const gap = Number(part);
    const chip = placed[gap];
    let state = chip === undefined ? 'empty' : 'filled';
    if (checked) {
      state = results[gap] ? 'right' : 'wrong';
    }
    return (
      // eslint-disable-next-line react/no-array-index-key -- template parts are positional
      <span key={index} className="gap-wrap">
        <button
          type="button"
          className={`gap ${state}`}
          aria-label={gapLabel(gap, chip?.text, checked ? results[gap] : undefined)}
          disabled={checked}
          onClick={() => clickGap(gap)}
          onDragOver={(event) => event.preventDefault()}
          onDrop={(event) => drop(event, gap)}
        >
          {chip === undefined ? ' ' : <Inline text={chip.text} />}
        </button>
        {checked && !results[gap] && (
          <span className="gap-fix">
            <Inline text={question.answers[gap]} />
          </span>
        )}
      </span>
    );
  });

  return (
    <>
      {question.code === true ? (
        <pre className="fill-template code">{content}</pre>
      ) : (
        <div className="fill-template">{content}</div>
      )}
      {!checked && (
        <div className="chip-tray" role="group" aria-label="Options">
          {chips.map((chip) => (
            <button
              type="button"
              key={chip.id}
              className={`chip ${selected?.id === chip.id ? 'selected' : ''}`}
              draggable={!used.has(chip.id)}
              disabled={used.has(chip.id)}
              aria-pressed={selected?.id === chip.id}
              onClick={() => setSelected(selected?.id === chip.id ? undefined : chip)}
              onDragStart={(event) => event.dataTransfer.setData('text/plain', String(chip.id))}
            >
              <Inline text={chip.text} />
            </button>
          ))}
        </div>
      )}
      {checked ? (
        <Feedback correct={allRight} text={question.explanation} />
      ) : (
        <div className="form-actions">
          <button
            type="button"
            className="button primary"
            disabled={used.has(undefined)}
            onClick={check}
          >
            Check answer
          </button>
        </div>
      )}
    </>
  );
}

// Screen readers hear the result too, not only the colour of the gap.
function gapLabel(gap: number, text: string | undefined, correct: boolean | undefined) {
  const content = text === undefined ? `Gap ${gap + 1}, empty` : `Gap ${gap + 1}: ${text}`;
  if (correct === undefined) {
    return content;
  }
  return `${content}, ${correct ? 'correct' : 'incorrect'}`;
}
