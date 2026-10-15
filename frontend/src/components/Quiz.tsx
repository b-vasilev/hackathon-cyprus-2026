import { ArrowRight, Check, RotateCcw, X } from 'lucide-react';
import { useState } from 'react';
import { BranchScenario } from './BranchScenario';
import { CardPick } from './CardPick';
import { FillGaps } from './FillGaps';
import { Inline } from './LessonContent';
import { PatchReview } from './PatchReview';
import { Feedback } from './QuizFeedback';
import type { ChoiceQuestion, Question } from '../foundations/types';

export const passRatio = 0.8;

const instructions: Record<Question['kind'], string> = {
  single: 'Pick one',
  multiple: 'Pick all that apply',
  cards: 'Click the card you think is right',
  fill: 'Drag the options into the gaps, or tap an option and then a gap',
  branch: 'Each decision changes what happens next',
  patch: 'Click every line you would comment on, then submit',
};

// Steps through questions one at a time. Each question view reports whether the
// learner got it right on the first attempt.
export function Quiz({
  questions,
  onFinish,
}: Readonly<{
  questions: Question[];
  onFinish?: (score: number, total: number) => void;
}>) {
  const [position, setPosition] = useState(0);
  const [answered, setAnswered] = useState(false);
  const [score, setScore] = useState(0);

  function restart() {
    setPosition(0);
    setAnswered(false);
    setScore(0);
  }

  if (position === questions.length) {
    const passed = score / questions.length >= passRatio;
    return (
      <div className="quiz-result" role="status">
        <span className="quiz-result-icon">
          {passed ? (
            <Check size={22} aria-hidden="true" />
          ) : (
            <RotateCcw size={22} aria-hidden="true" />
          )}
        </span>
        <h3>
          {score} of {questions.length} · {passed ? 'Passed' : 'Not passed yet'}
        </h3>
        <p>
          {passed
            ? 'Nice. The lessons stay here if you need to look something up later.'
            : `You need ${Math.ceil(questions.length * passRatio)} to pass. Have another look at the lessons and try again.`}
        </p>
        <button type="button" className="button secondary" onClick={restart}>
          <RotateCcw size={16} />
          Try again
        </button>
      </div>
    );
  }

  const question = questions[position];

  function answer(correct: boolean) {
    setAnswered(true);
    if (correct) {
      setScore((current) => current + 1);
    }
  }

  function next() {
    const nextPosition = position + 1;
    if (nextPosition === questions.length) {
      onFinish?.(score, questions.length);
    }
    setPosition(nextPosition);
    setAnswered(false);
  }

  return (
    <div className="quiz">
      <div className="quiz-progress">
        <span>
          Question {position + 1} of {questions.length}
        </span>
        <span>{instructions[question.kind]}</span>
      </div>
      {question.scenario !== undefined && (
        <div className="quiz-scenario">
          <p className="eyebrow">Scenario</p>
          <p>
            <Inline text={question.scenario} />
          </p>
        </div>
      )}
      <p className="quiz-prompt">
        <Inline text={question.prompt} />
      </p>
      {question.kind === 'cards' && (
        <CardPick key={question.id} question={question} onAnswered={answer} />
      )}
      {question.kind === 'fill' && (
        <FillGaps key={question.id} question={question} onAnswered={answer} />
      )}
      {question.kind === 'branch' && (
        <BranchScenario key={question.id} question={question} onAnswered={answer} />
      )}
      {question.kind === 'patch' && (
        <PatchReview key={question.id} question={question} onAnswered={answer} />
      )}
      {(question.kind === 'single' || question.kind === 'multiple') && (
        <Choice key={question.id} question={question} onAnswered={answer} />
      )}
      {answered && (
        <div className="form-actions">
          <button type="button" className="button primary" onClick={next}>
            {position + 1 === questions.length ? 'See result' : 'Next question'}
            <ArrowRight size={16} />
          </button>
        </div>
      )}
    </div>
  );
}

function Choice({
  question,
  onAnswered,
}: Readonly<{ question: ChoiceQuestion; onAnswered: (correct: boolean) => void }>) {
  const [selected, setSelected] = useState<number[]>([]);
  const [checked, setChecked] = useState(false);
  const correct =
    selected.length === question.correct.length &&
    question.correct.every((index) => selected.includes(index));

  function toggle(index: number) {
    if (question.kind === 'single') {
      setSelected([index]);
    } else {
      setSelected((current) =>
        current.includes(index) ? current.filter((item) => item !== index) : [...current, index]
      );
    }
  }

  function check() {
    setChecked(true);
    onAnswered(correct);
  }

  return (
    <>
      <fieldset className="quiz-options">
        <legend className="visually-hidden">Answers</legend>
        {question.options.map((option, index) => {
          const chosen = selected.includes(index);
          const expected = question.correct.includes(index);
          let state = '';
          if (checked && expected) {
            state = 'expected';
          } else if (checked && chosen) {
            state = 'wrong';
          }
          return (
            <label key={option} className={`quiz-option ${chosen ? 'chosen' : ''} ${state}`}>
              <input
                type={question.kind === 'single' ? 'radio' : 'checkbox'}
                name={question.id}
                checked={chosen}
                disabled={checked}
                onChange={() => toggle(index)}
              />
              <span>
                <Inline text={option} />
              </span>
              {state === 'expected' && (
                <span className="quiz-mark">
                  <Check size={15} aria-hidden="true" />
                  Correct answer
                </span>
              )}
              {state === 'wrong' && (
                <span className="quiz-mark">
                  <X size={15} aria-hidden="true" />
                  Your answer
                </span>
              )}
            </label>
          );
        })}
      </fieldset>
      {checked ? (
        <Feedback correct={correct} text={question.explanation} />
      ) : (
        <div className="form-actions">
          <button
            type="button"
            className="button primary"
            disabled={selected.length === 0}
            onClick={check}
          >
            Check answer
          </button>
        </div>
      )}
    </>
  );
}
