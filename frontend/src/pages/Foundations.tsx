import { ArrowRight, Check, Clock, GraduationCap, ListChecks } from 'lucide-react';
import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { LessonBlocks } from '../components/LessonContent';
import { Quiz, passRatio } from '../components/Quiz';
import { Badge, EmptyState, PageHeading } from '../components/ui';
import { findModule, foundationModules, markPassed, passedModules } from '../foundations';

const quizStep = 'quiz';

export function Foundations() {
  const passed = passedModules();
  return (
    <>
      <PageHeading
        eyebrow="Foundations"
        title="Engineering basics for every product."
        description="Short lessons on how we work: Git, pull requests, CI, code quality, testing, releases, APIs and AI assistants. Each module ends with a quiz."
      />
      <div className="intro-strip">
        <GraduationCap size={19} aria-hidden="true" />
        <p>
          Foundations are not tied to one product. Take them in order or jump to the topic you need.
          Quiz results are saved in this browser only.
        </p>
      </div>
      <ol className="foundation-grid">
        {foundationModules.map((module, index) => (
          <li key={module.id}>
            <Link className="product-card foundation-card" to={`/foundations/${module.id}`}>
              <div className="card-top">
                <span className="step-number">{String(index + 1).padStart(2, '0')}</span>
                {passed.includes(module.id) ? (
                  <Badge>
                    <Check size={13} aria-hidden="true" />
                    Quiz passed
                  </Badge>
                ) : (
                  <Badge>Not started</Badge>
                )}
              </div>
              <h2>{module.title}</h2>
              <p className="foundation-summary">{module.summary}</p>
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
        ))}
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
  const [step, setStep] = useState(module?.lessons.at(0)?.id ?? quizStep);
  const [passed, setPassed] = useState(() => passedModules().includes(id));

  if (module === undefined) {
    return (
      <EmptyState
        title="Module not found"
        action={
          <Link className="button primary" to="/foundations">
            All foundations
          </Link>
        }
      >
        This Foundations module does not exist.
      </EmptyState>
    );
  }

  const lesson = module.lessons.find((item) => item.id === step);
  const next = module.lessons.at(module.lessons.findIndex((item) => item.id === step) + 1);

  function go(target: string) {
    setStep(target);
    window.scrollTo({ top: 0 });
  }

  function finish(score: number, total: number) {
    if (module !== undefined && score / total >= passRatio) {
      markPassed(module.id);
      setPassed(true);
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
          passed ? (
            <Badge>
              <Check size={13} aria-hidden="true" />
              Quiz passed
            </Badge>
          ) : undefined
        }
      />
      <div className="passport-layout">
        <nav className="section-nav" aria-label="Module lessons">
          {module.lessons.map((item, index) => (
            <button
              type="button"
              key={item.id}
              className={item.id === step ? 'active' : ''}
              aria-current={item.id === step ? 'step' : undefined}
              onClick={() => go(item.id)}
            >
              <span>{String(index + 1).padStart(2, '0')}</span>
              {item.title}
            </button>
          ))}
          <button
            type="button"
            className={step === quizStep ? 'active' : ''}
            aria-current={step === quizStep ? 'step' : undefined}
            onClick={() => go(quizStep)}
          >
            <ListChecks size={15} aria-hidden="true" />
            Check yourself
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
            <Quiz key={module.id} questions={module.quiz} onFinish={finish} />
          </section>
        ) : (
          <article aria-labelledby="lesson-title">
            <div className="section-title">
              <h2 id="lesson-title">{lesson.title}</h2>
              <p className="section-summary">{lesson.summary}</p>
            </div>
            <LessonBlocks blocks={lesson.blocks} />
            <div className="form-actions lesson-actions">
              <button
                type="button"
                className="button primary"
                onClick={() => go(next?.id ?? quizStep)}
              >
                {next === undefined ? 'Take the quiz' : `Next: ${next.title}`}
                <ArrowRight size={16} />
              </button>
            </div>
          </article>
        )}
      </div>
    </>
  );
}
