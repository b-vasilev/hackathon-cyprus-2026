import { ArrowRight, Award, Check, Clock, GraduationCap, ListChecks } from 'lucide-react';
import { useCallback, useEffect, useState } from 'react';
import { Link, NavLink, useParams, useSearchParams } from 'react-router-dom';
import {
  AchievementGallery,
  AchievementSummary,
  BoldStrategyDialog,
} from '../components/Achievements';
import { LessonBlocks } from '../components/LessonContent';
import { Quiz, passRatio } from '../components/Quiz';
import { Badge, PageHeading } from '../components/ui';
import {
  findModule,
  foundationModules,
  loadProgress,
  markLessonDone,
  markVisited,
  moduleStatus,
  recordQuiz,
  type ModuleState,
} from '../foundations';
import {
  achievements,
  setFlag,
  snapshot,
  withAchievements,
  type Snapshot,
} from '../foundations/achievements';
import { NotFound } from './NotFound';

const quizStep = 'quiz';

const stateLabels: Record<ModuleState, string> = {
  'not-started': 'Not started',
  'in-progress': 'In progress',
  completed: 'Completed',
};

function ProgressBar({ done, total }: Readonly<{ done: number; total: number }>) {
  return (
    <span className="progress-bar">
      <span className="progress-track" aria-hidden="true">
        <span className="progress-fill" style={{ width: `${(done / total) * 100}%` }} />
      </span>
      <span className="progress-text">
        {done} of {total} steps
      </span>
    </span>
  );
}

function FoundationsTabs({ state }: Readonly<{ state: Snapshot }>) {
  const unlocked = achievements.filter((item) => item.unlocked(state)).length;
  return (
    <nav className="foundations-tabs" aria-label="Foundations sections">
      <NavLink to="/foundations" end>
        Modules
      </NavLink>
      <NavLink to="/foundations/achievements">
        Achievements
        <span className="foundations-tab-count">
          {unlocked}/{achievements.length}
        </span>
      </NavLink>
    </nav>
  );
}

function FoundationsHeading() {
  return (
    <PageHeading
      eyebrow="Foundations"
      title="Engineering basics for every product."
      description="Short lessons on how we work: Git and pull requests, code quality, CI, testing, releases, APIs and AI assistants. Each module ends with a quiz."
    />
  );
}

export function Foundations() {
  const progress = loadProgress();
  const completed = foundationModules.filter(
    (module) => moduleStatus(module, progress).state === 'completed'
  ).length;
  return (
    <>
      <FoundationsHeading />
      <FoundationsTabs state={snapshot()} />
      <div className="intro-strip foundations-strip">
        <GraduationCap size={19} aria-hidden="true" />
        <p>
          You have completed {completed} of {foundationModules.length} modules. A module is complete
          when you finish every lesson and pass its quiz. Progress is saved in this browser only.
        </p>
        <AchievementSummary state={snapshot()} />
      </div>
      <ol className="foundation-grid">
        {foundationModules.map((module, index) => {
          const status = moduleStatus(module, progress);
          return (
            <li key={module.id}>
              <Link className="product-card foundation-card" to={`/foundations/${module.id}`}>
                <div className="card-top">
                  <span className="step-number">{String(index + 1).padStart(2, '0')}</span>
                  <Badge tone={status.state === 'completed' ? 'done' : ''}>
                    {status.state === 'completed' && <Check size={13} aria-hidden="true" />}
                    {stateLabels[status.state]}
                  </Badge>
                </div>
                <h2>{module.title}</h2>
                <p className="foundation-summary">{module.summary}</p>
                <ProgressBar done={status.done} total={status.total} />
                <div className="card-bottom">
                  <span className="foundation-meta">
                    <Clock size={14} aria-hidden="true" />
                    {module.minutes} min · {module.lessons.length} lessons · {module.quiz.length}{' '}
                    questions
                  </span>
                  <ArrowRight size={18} aria-hidden="true" />
                </div>
              </Link>
            </li>
          );
        })}
      </ol>
    </>
  );
}

export function FoundationModulePage() {
  const { id } = useParams();
  return <ModuleView key={id} id={id ?? ''} />;
}

function ModuleView({ id }: Readonly<{ id: string }>) {
  const module = findModule(id);
  const [searchParams] = useSearchParams();
  // Links such as the home page's next mission can open a lesson or the quiz directly.
  const [step, setStep] = useState(() => {
    const wanted = searchParams.get('step') === quizStep ? quizStep : searchParams.get('lesson');
    if (wanted === quizStep || module?.lessons.some((lesson) => lesson.id === wanted) === true) {
      return wanted ?? quizStep;
    }
    return module?.lessons.at(0)?.id ?? quizStep;
  });
  const [progress, setProgress] = useState(loadProgress);
  const [zeroQuiz, setZeroQuiz] = useState<number>();

  // Saves progress; any achievement it unlocks is announced app-wide.
  const apply = useCallback((change: () => void) => {
    withAchievements(change);
    setProgress(loadProgress());
  }, []);

  useEffect(() => {
    if (findModule(id) !== undefined) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- syncs from browser storage once per module visit
      apply(() => markVisited(id));
    }
  }, [id, apply]);

  if (module === undefined) {
    return <NotFound />;
  }

  const status = moduleStatus(module, progress);
  const doneLessons = progress[module.id]?.lessons ?? [];
  const lesson = module.lessons.find((item) => item.id === step);
  const next = module.lessons.at(module.lessons.findIndex((item) => item.id === step) + 1);

  function go(target: string) {
    setStep(target);
    window.scrollTo({ top: 0 });
  }

  function finishLesson(lessonId: string) {
    if (module !== undefined) {
      apply(() => markLessonDone(module.id, lessonId));
    }
    go(next?.id ?? quizStep);
  }

  function finishQuiz(score: number, total: number) {
    if (module !== undefined) {
      apply(() => {
        recordQuiz(module.id, score, total, passRatio);
        if (score === 0) {
          setFlag('bold-strategy');
        }
      });
    }
    if (score === 0) {
      setZeroQuiz(total);
    }
  }

  return (
    <>
      <Link to="/foundations" className="back-link">
        ← Foundations
      </Link>
      <PageHeading
        eyebrow={`Foundations · ${module.minutes} min`}
        title={module.title}
        description={module.outcome}
        action={
          <div className="module-progress">
            <Badge tone={status.state === 'completed' ? 'done' : ''}>
              {status.state === 'completed' && <Check size={13} aria-hidden="true" />}
              {stateLabels[status.state]}
            </Badge>
            <ProgressBar done={status.done} total={status.total} />
          </div>
        }
      />
      <div className="passport-layout">
        <nav className="section-nav" aria-label="Module lessons">
          {module.lessons.map((item, index) => {
            const done = doneLessons.includes(item.id);
            return (
              <button
                type="button"
                key={item.id}
                className={`${item.id === step ? 'active' : ''} ${done ? 'done' : ''}`}
                aria-current={item.id === step ? 'step' : undefined}
                onClick={() => go(item.id)}
              >
                <span>{String(index + 1).padStart(2, '0')}</span>
                {item.title}
                {done && <Check size={15} className="nav-check" aria-label="Done" role="img" />}
              </button>
            );
          })}
          <button
            type="button"
            className={`${step === quizStep ? 'active' : ''} ${status.quizPassed ? 'done' : ''}`}
            aria-current={step === quizStep ? 'step' : undefined}
            onClick={() => go(quizStep)}
          >
            <ListChecks size={15} aria-hidden="true" />
            Check yourself
            {status.quizPassed && (
              <Check size={15} className="nav-check" aria-label="Passed" role="img" />
            )}
          </button>
        </nav>
        {lesson === undefined ? (
          <section aria-labelledby="quiz-title">
            <div className="section-title">
              <h2 id="quiz-title">Check yourself</h2>
              <p className="section-summary">
                {module.quiz.length} questions. You get feedback after each answer. Pass with{' '}
                {Math.round(passRatio * 100)}% or more.
              </p>
            </div>
            <Quiz key={module.id} questions={module.quiz} onFinish={finishQuiz} />
          </section>
        ) : (
          <article aria-labelledby="lesson-title">
            <div className="section-title">
              <h2 id="lesson-title">{lesson.title}</h2>
              <p className="section-summary">{lesson.summary}</p>
            </div>
            <LessonBlocks key={lesson.id} blocks={lesson.blocks} />
            <div className="form-actions lesson-actions">
              <button
                type="button"
                className="button primary"
                onClick={() => finishLesson(lesson.id)}
              >
                <Check size={16} />
                {next === undefined ? 'Done. Take the quiz' : `Done. Next: ${next.title}`}
              </button>
            </div>
          </article>
        )}
      </div>
      {zeroQuiz !== undefined && (
        <BoldStrategyDialog total={zeroQuiz} onClose={() => setZeroQuiz(undefined)} />
      )}
    </>
  );
}

export function FoundationAchievements() {
  const state = snapshot();
  const unlocked = achievements.filter((item) => item.unlocked(state)).length;
  return (
    <>
      <FoundationsHeading />
      <FoundationsTabs state={snapshot()} />
      <div className="intro-strip">
        <Award size={19} aria-hidden="true" />
        <p>
          You have unlocked {unlocked} of {achievements.length} achievements. Each one shows its
          picture when you unlock it. Achievements are saved in this browser only.
        </p>
      </div>
      <AchievementGallery state={state} />
    </>
  );
}
