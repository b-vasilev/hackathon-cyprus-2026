import { useQuery } from '@tanstack/react-query';
import { ArrowRight, ArrowUpRight, Check, Compass, Hammer } from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';
import { AchievementSummary } from '../components/Achievements';
import { ProductIdentity } from '../components/ProductIdentity';
import { ErrorNotice, Loading } from '../components/ui';
import { foundationModules, loadProgress, moduleStatus, type Progress } from '../foundations';
import { snapshot } from '../foundations/achievements';
import { api, type Product } from '../lib/api';
import { canUseStudio, useMe } from '../lib/auth';
import { productCardTheme } from '../lib/productCardTheme';
import type { FoundationModule } from '../foundations/types';

interface Mission {
  eyebrow: string;
  title: string;
  outcome: string;
  action: string;
  to: string;
  moduleId?: string;
}

// "tatiana.zimovets@…" becomes "Tatiana"; anything unusual falls back to a neutral greeting.
function firstName(email: string) {
  const name = email.split('@')[0]?.split(/[._-]/)[0] ?? '';
  return /^[a-z]{2,}$/i.test(name) ? name[0].toUpperCase() + name.slice(1).toLowerCase() : '';
}

// The next concrete step: the first unfinished lesson or quiz, then a published passport.
function nextMission(progress: Progress, published: Product[]): Mission {
  for (const module of foundationModules) {
    const status = moduleStatus(module, progress);
    if (status.state === 'completed') {
      continue;
    }
    const done = progress[module.id]?.lessons ?? [];
    const lesson = module.lessons.find((item) => !done.includes(item.id));
    if (lesson !== undefined) {
      return {
        eyebrow: status.state === 'not-started' ? 'Start here' : 'Continue where you stopped',
        title: `${module.title}: ${lesson.title}`,
        outcome: module.outcome,
        action: status.state === 'not-started' ? 'Start the lesson' : 'Continue the lesson',
        to: `/foundations/${module.id}?lesson=${lesson.id}`,
        moduleId: module.id,
      };
    }
    return {
      eyebrow: 'One step left in this module',
      title: `Take the ${module.title} quiz`,
      outcome: module.outcome,
      action: 'Take the quiz',
      to: `/foundations/${module.id}?step=quiz`,
      moduleId: module.id,
    };
  }
  const product = published.at(0);
  if (product !== undefined) {
    return {
      eyebrow: 'Foundations complete',
      title: `Explore the ${product.name} passport`,
      outcome: 'Learn what the product does, how it is built and how a request flows through it.',
      action: 'Open the passport',
      to: `/published/${product.id}`,
    };
  }
  return {
    eyebrow: 'Foundations complete',
    title: 'You finished every Foundations module',
    outcome: 'When a lead publishes a passport for your product, it will appear below.',
    action: 'Review Foundations',
    to: '/foundations',
  };
}

function ModuleRoute({ progress, current }: Readonly<{ progress: Progress; current?: string }>) {
  const [selected, setSelected] = useState(current ?? foundationModules[0].id);
  const module: FoundationModule =
    foundationModules.find((item) => item.id === selected) ?? foundationModules[0];
  const status = moduleStatus(module, progress);
  return (
    <div className="landing-route home-route">
      <p className="eyebrow">Your Foundations route</p>
      <ol className="route-steps">
        {foundationModules.map((item, index) => {
          const state = moduleStatus(item, progress).state;
          const label = {
            completed: 'completed',
            'in-progress': 'in progress',
            'not-started': 'not started',
          }[state];
          return (
            <li
              key={item.id}
              className={`${state === 'completed' ? 'reached' : ''} ${item.id === current ? 'current' : ''}`}
            >
              <button
                type="button"
                className={item.id === selected ? 'active' : ''}
                aria-pressed={item.id === selected}
                onClick={() => setSelected(item.id)}
              >
                <span className={`route-dot ${state}`} aria-hidden="true">
                  {state === 'completed' && <Check size={12} />}
                </span>
                <span className="route-label">
                  <span className="step-number">{String(index + 1).padStart(2, '0')}</span>
                  {item.title}
                </span>
                <span className="visually-hidden">, {label}</span>
              </button>
            </li>
          );
        })}
      </ol>
      <div className="route-detail" aria-live="polite">
        <strong>{module.title}</strong>
        <p>{module.summary}</p>
        <p className="home-route-meta">
          {status.done} of {status.total} steps ·{' '}
          <Link className="text-link" to={`/foundations/${module.id}`}>
            Open module
            <ArrowRight size={14} />
          </Link>
        </p>
      </div>
    </div>
  );
}

export function Home() {
  const me = useMe();
  const studio = canUseStudio(me);
  const progress = loadProgress();
  const products = useQuery({
    queryKey: ['products'],
    queryFn: ({ signal }) => api<Product[]>('/products', { signal }),
  });
  const published = (products.data ?? []).filter((item) => item.published_revision_id !== '');
  const mission = nextMission(progress, published);
  const missionModule = foundationModules.find((item) => item.id === mission.moduleId);
  const completed = foundationModules.filter(
    (module) => moduleStatus(module, progress).state === 'completed'
  ).length;
  const name = firstName(me.user.email);

  return (
    <div className="home">
      <header className="home-heading">
        <p className="eyebrow">Your onboarding</p>
        <h1>{name === '' ? 'Welcome back.' : `Welcome back, ${name}.`}</h1>
      </header>

      <section className="home-hero" aria-labelledby="mission-title">
        <div className="home-mission">
          <p className="eyebrow">{mission.eyebrow}</p>
          <h2 id="mission-title">{mission.title}</h2>
          <p className="home-outcome">
            <Compass size={18} aria-hidden="true" />
            <span>
              <strong>You will be able to:</strong> {mission.outcome}
            </span>
          </p>
          <Link className="button primary" to={mission.to}>
            {mission.action}
            <ArrowRight size={17} />
          </Link>
          {missionModule !== undefined && (
            <div className="home-lessons">
              <p className="eyebrow">In this module</p>
              <ol>
                {missionModule.lessons.map((lesson) => {
                  const done = (progress[missionModule.id]?.lessons ?? []).includes(lesson.id);
                  return (
                    <li key={lesson.id} className={done ? 'done' : ''}>
                      <span className="home-lesson-mark" aria-hidden="true">
                        {done && <Check size={12} />}
                      </span>
                      <Link to={`/foundations/${missionModule.id}?lesson=${lesson.id}`}>
                        {lesson.title}
                      </Link>
                      {done && <span className="visually-hidden"> (done)</span>}
                    </li>
                  );
                })}
              </ol>
            </div>
          )}
        </div>
        <ModuleRoute progress={progress} current={mission.moduleId} />
      </section>

      <section className="home-section" aria-labelledby="passports-title">
        <div className="home-section-heading">
          <h2 id="passports-title">Product passports</h2>
          <Link className="text-link" to="/published">
            All published passports
            <ArrowRight size={14} />
          </Link>
        </div>
        {products.isPending && <Loading />}
        {products.isError && <ErrorNotice error={products.error} />}
        {products.isSuccess && published.length === 0 && (
          <p className="home-empty">
            No published passports for your products yet. When a product lead publishes one, it
            appears here.
          </p>
        )}
        <div className="product-grid">
          {published.slice(0, 3).map((product) => (
            <article
              className={`product-card published-card${product.card_theme ? ' product-branded' : ''}`}
              style={productCardTheme(product.card_theme)}
              key={product.id}
            >
              <Link className="product-card-content" to={`/published/${product.id}`}>
                <ProductIdentity
                  name={product.name}
                  theme={product.card_theme}
                  icon={product.icon_url}
                />
                <h2>{product.name}</h2>
                <ArrowUpRight size={20} aria-hidden="true" />
              </Link>
            </article>
          ))}
        </div>
      </section>

      <section className="home-section home-columns" aria-label="Foundations and studio">
        <div className="home-panel">
          <h2>Foundations</h2>
          <p>
            You have completed {completed} of {foundationModules.length} modules.
          </p>
          <AchievementSummary state={snapshot()} />
          <Link className="text-link" to="/foundations">
            All modules
            <ArrowRight size={14} />
          </Link>
        </div>
        {studio && (
          <div className="home-panel">
            <h2>Product studio</h2>
            <p>
              Connect repositories, generate passport drafts, review them and publish what learners
              see.
            </p>
            <Link className="button secondary" to="/products">
              <Hammer size={16} />
              Open the studio
            </Link>
          </div>
        )}
      </section>
    </div>
  );
}
