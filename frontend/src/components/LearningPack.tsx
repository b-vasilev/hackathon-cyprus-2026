import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  BookOpen,
  Check,
  CircleDot,
  EyeOff,
  Flag,
  GitPullRequest,
  ListChecks,
  RefreshCw,
  Send,
  Sparkles,
} from 'lucide-react';
import { useCallback, useState } from 'react';
import { loadProgress, markLessonDone, recordQuiz } from '../foundations';
import { withAchievements } from '../foundations/achievements';
import { api, type Chapter, type LearningPack } from '../lib/api';
import { packKey } from '../lib/learningPack';
import { Inline } from './Inline';
import { Quiz, passRatio } from './Quiz';
import { SafeMarkdown } from './SafeMarkdown';
import { Badge, ErrorNotice, ExternalLink, Loading } from './ui';
import type { ChoiceQuestion } from '../foundations/types';

type Pack = NonNullable<LearningPack['content']>;
type PackQuestion = NonNullable<Pack['quiz']>[number];

function questions(items: PackQuestion[] | null): ChoiceQuestion[] {
  return (items ?? []).map((item) => ({
    id: item.id,
    kind: item.kind === 'multiple' ? 'multiple' : 'single',
    prompt: item.prompt,
    options: item.options ?? [],
    correct: item.correct ?? [],
    explanation: item.explanation,
  }));
}

/** Leads generate, try and publish the practice for one passport version. */
export function PracticeStudio({
  revisionId,
  productId,
  chapters,
  onChapter,
}: Readonly<{
  revisionId: string;
  productId: string;
  chapters: Chapter[];
  onChapter: (id: string) => void;
}>) {
  const queryClient = useQueryClient();
  const key = [packKey, revisionId];
  const pack = useQuery({
    queryKey: key,
    queryFn: ({ signal }) =>
      api<LearningPack>(`/passport-revisions/${revisionId}/learning-pack`, {
        signal,
      }),
    refetchInterval: (query) => (query.state.data?.status === 'generating' ? 4000 : false),
  });
  const action = useMutation({
    mutationFn: (path: '' | '/publish' | '/unpublish') =>
      api<LearningPack>(`/passport-revisions/${revisionId}/learning-pack${path}`, {
        method: 'POST',
      }),
    onSuccess: (value) => {
      queryClient.setQueryData(key, value);
      void queryClient.invalidateQueries({
        queryKey: [packKey, 'published', productId],
      });
    },
  });
  if (pack.isPending) {
    return <Loading label="Loading practice…" />;
  }
  if (pack.isError) {
    return <ErrorNotice error={pack.error} />;
  }
  const view = pack.data;
  const generating = view.status === 'generating';
  const busy = generating || action.isPending;
  let state = 'Not generated';
  if (view.content) {
    state = view.published ? 'Published' : 'Draft';
  }
  return (
    <article aria-labelledby="practice-title" className="practice">
      <div className="section-title">
        <p className="eyebrow">Practice</p>
        <h2 id="practice-title" tabIndex={-1}>
          Quiz and first task
        </h2>
        <p className="section-summary">
          A short quiz on this passport and a guided first task built around a real closed issue
          from the repository. Learners see it in the published passport after you publish it.
        </p>
      </div>
      <div className="practice-controls">
        <Badge tone={view.published ? 'done' : ''}>
          {view.published && <Check size={13} aria-hidden="true" />}
          {state}
          {view.unpublished_changes && ' · unpublished changes'}
        </Badge>
        {view.content && view.status !== 'generating' && (
          <button
            type="button"
            className="button primary"
            disabled={busy}
            onClick={() => action.mutate('/publish')}
          >
            <Send size={16} />
            {view.published ? 'Publish changes' : 'Publish to learners'}
          </button>
        )}
        <button
          type="button"
          className={view.content ? 'button secondary' : 'button primary'}
          disabled={busy}
          onClick={() => action.mutate('')}
        >
          {generating ? <RefreshCw size={16} className="spin" /> : <Sparkles size={16} />}
          {generating && 'Generating…'}
          {!generating && (view.content ? 'Generate again' : 'Generate practice')}
        </button>
        {view.published && (
          <button
            type="button"
            className="text-button"
            disabled={busy}
            onClick={() => action.mutate('/unpublish')}
          >
            <EyeOff size={16} aria-hidden="true" />
            Unpublish
          </button>
        )}
      </div>
      {generating && (
        <p className="notice" role="status">
          <RefreshCw size={17} className="spin" aria-hidden="true" />
          Reading closed issues on GitHub and writing the quiz and task. This usually takes one to
          three minutes. You can keep reading the passport.
        </p>
      )}
      {view.status === 'failed' && (
        <p className="notice error" role="alert">
          Practice generation failed: {view.error}
        </p>
      )}
      {action.isError && <ErrorNotice error={action.error} />}
      {view.content && (
        <PracticeContent
          key={view.updated_at}
          pack={view.content}
          chapters={chapters}
          onChapter={onChapter}
        />
      )}
    </article>
  );
}

/** What learners take: the product quiz, then the mission and its final check. */
export function PracticeLearner({
  pack,
  productId,
  chapters,
  onChapter,
}: Readonly<{
  pack: LearningPack;
  productId: string;
  chapters: Chapter[];
  onChapter: (id: string) => void;
}>) {
  return (
    <article aria-labelledby="practice-title" className="practice">
      <div className="section-title">
        <p className="eyebrow">Practice</p>
        <h2 id="practice-title" tabIndex={-1}>
          Quiz and first task
        </h2>
        <p className="section-summary">
          Check what you learned from this passport, then follow a real change through the code.
          Your progress is saved to your account.
        </p>
      </div>
      {pack.content && (
        <PracticeContent
          key={pack.published_at}
          pack={pack.content}
          chapters={chapters}
          onChapter={onChapter}
          progressKey={`product:${productId}`}
        />
      )}
    </article>
  );
}

// Progress is tracked only for learners; leads preview the same content without saving.
function PracticeContent({
  pack,
  chapters,
  onChapter,
  progressKey,
}: Readonly<{
  pack: Pack;
  chapters: Chapter[];
  onChapter: (id: string) => void;
  progressKey?: string;
}>) {
  const [progress, setProgress] = useState(loadProgress);
  const [preview, setPreview] = useState<{ steps: string[]; checked: boolean }>({
    steps: [],
    checked: false,
  });
  const apply = useCallback((change: () => void) => {
    withAchievements(change);
    setProgress(loadProgress());
  }, []);
  const quizKey = `${progressKey}:quiz`;
  const taskKey = `${progressKey}:task`;
  const quizPassed = progressKey !== undefined && progress[quizKey]?.quizPassed === true;
  const doneSteps = progressKey === undefined ? preview.steps : (progress[taskKey]?.lessons ?? []);
  const checkPassed =
    progressKey === undefined ? preview.checked : progress[taskKey]?.quizPassed === true;
  const steps = pack.mission.steps ?? [];
  const ready = steps.every((step) => doneSteps.includes(step.id)) && checkPassed;
  const chapterTitle = (id: string) => chapters.find((item) => item.id === id)?.title;

  function finishStep(id: string) {
    if (progressKey === undefined) {
      setPreview((current) => ({ ...current, steps: [...current.steps, id] }));
    } else {
      apply(() => markLessonDone(taskKey, id));
    }
  }

  function finishQuiz(key: string, score: number, total: number) {
    if (progressKey === undefined) {
      if (key === taskKey && score / total >= passRatio) {
        setPreview((current) => ({ ...current, checked: true }));
      }
      return;
    }
    apply(() => recordQuiz(key, score, total, passRatio));
  }

  const issue = pack.mission.issue;
  return (
    <>
      <section className="practice-section" aria-labelledby="practice-quiz">
        <div className="practice-heading">
          <ListChecks size={20} aria-hidden="true" />
          <h3 id="practice-quiz">Product quiz</h3>
          {quizPassed && (
            <Badge tone="done">
              <Check size={13} aria-hidden="true" />
              Passed
            </Badge>
          )}
        </div>
        <p className="practice-lead">
          {pack.quiz?.length ?? 0} questions about this passport. Pass with{' '}
          {Math.round(passRatio * 100)}% or more.
        </p>
        <Quiz
          questions={questions(pack.quiz)}
          onFinish={(score, total) => finishQuiz(quizKey, score, total)}
        />
      </section>

      <section className="practice-section" aria-labelledby="practice-mission">
        <div className="practice-heading">
          <Flag size={20} aria-hidden="true" />
          <h3 id="practice-mission">{pack.mission.title}</h3>
        </div>
        <p className="practice-lead">
          <Inline text={pack.mission.intro} />
        </p>
        {issue && (
          <div className="mission-issue">
            <p className="eyebrow">Your issue to study</p>
            <p className="mission-issue-title">
              <CircleDot size={17} aria-hidden="true" />
              <ExternalLink href={issue.url}>
                #{issue.number} {issue.title}
              </ExternalLink>
            </p>
            <p className="mission-issue-title">
              <GitPullRequest size={17} aria-hidden="true" />
              <ExternalLink href={issue.pull_url}>
                Pull request #{issue.pull_number} {issue.pull_title}
              </ExternalLink>
            </p>
            {issue.why !== '' && (
              <p className="mission-issue-why">
                <Inline text={issue.why} />
              </p>
            )}
            {(issue.files?.length ?? 0) > 0 && (
              <details>
                <summary>{issue.files?.length} changed files</summary>
                <ul className="mission-files">
                  {issue.files?.map((file) => (
                    <li key={file}>
                      <code>{file}</code>
                    </li>
                  ))}
                </ul>
              </details>
            )}
          </div>
        )}
        <ol className="mission-steps">
          {steps.map((step, index) => {
            const done = doneSteps.includes(step.id);
            const chapter = chapterTitle(step.chapter_id);
            return (
              <li key={step.id} className={done ? 'done' : ''}>
                <span className="mission-marker" aria-hidden="true">
                  {done ? <Check size={15} /> : index + 1}
                </span>
                <div className="mission-step">
                  <h4>
                    {step.title}
                    {done && <span className="visually-hidden"> (done)</span>}
                  </h4>
                  <div className="lesson-body">
                    <SafeMarkdown>{step.body}</SafeMarkdown>
                  </div>
                  {(step.commands?.length ?? 0) > 0 && (
                    <div className="lesson-code">
                      <pre>
                        <code>{step.commands?.join('\n')}</code>
                      </pre>
                    </div>
                  )}
                  <p className="mission-done-when">
                    <strong>Done when:</strong> <Inline text={step.done_when} />
                  </p>
                  <div className="mission-actions">
                    {!done && (
                      <button
                        type="button"
                        className="text-button"
                        onClick={() => finishStep(step.id)}
                      >
                        <Check size={16} />
                        Mark as done
                      </button>
                    )}
                    {chapter !== undefined && (
                      <button
                        type="button"
                        className="text-button"
                        onClick={() => onChapter(step.chapter_id)}
                      >
                        <BookOpen size={16} aria-hidden="true" />
                        Read: {chapter}
                      </button>
                    )}
                  </div>
                </div>
              </li>
            );
          })}
        </ol>
      </section>

      <section className="practice-section" aria-labelledby="practice-check">
        <div className="practice-heading">
          <ListChecks size={20} aria-hidden="true" />
          <h3 id="practice-check">Check your understanding of the change</h3>
          {checkPassed && (
            <Badge tone="done">
              <Check size={13} aria-hidden="true" />
              Passed
            </Badge>
          )}
        </div>
        <Quiz
          questions={questions(pack.mission.check)}
          onFinish={(score, total) => finishQuiz(taskKey, score, total)}
        />
      </section>

      <section
        className={`mission-finale${ready ? ' reached' : ''}`}
        aria-live="polite"
        aria-labelledby="practice-finale"
      >
        <Flag size={22} aria-hidden="true" />
        <div>
          <h3 id="practice-finale">{ready ? 'Mission complete' : 'Finish line'}</h3>
          <p>
            {ready
              ? pack.mission.finale
              : 'Mark every step as done and pass the check above to finish the mission.'}
          </p>
        </div>
      </section>
    </>
  );
}

/** Leads manage the practice for the version they review; learners take the published one. */
export function PracticePane({
  manage,
  pack,
  revisionId,
  productId,
  chapters,
  onChapter,
}: Readonly<{
  manage: boolean;
  pack: LearningPack | undefined;
  revisionId: string;
  productId: string;
  chapters: Chapter[];
  onChapter: (id: string) => void;
}>) {
  if (manage) {
    return (
      <PracticeStudio
        revisionId={revisionId}
        productId={productId}
        chapters={chapters}
        onChapter={onChapter}
      />
    );
  }
  if (pack === undefined) {
    return null;
  }
  return (
    <PracticeLearner pack={pack} productId={productId} chapters={chapters} onChapter={onChapter} />
  );
}
