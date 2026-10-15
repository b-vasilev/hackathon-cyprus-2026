import { Inline } from './LessonContent';

export function Feedback({ correct, text }: Readonly<{ correct: boolean; text: string }>) {
  return (
    <div className={`quiz-feedback ${correct ? '' : 'wrong'}`} role="status">
      <strong>{correct ? 'Right.' : 'Not quite.'}</strong> <Inline text={text} />
    </div>
  );
}
