import {
  ArrowRight,
  BookOpenCheck,
  Bot,
  FileSearch,
  GraduationCap,
  KeyRound,
  ListChecks,
  MessagesSquare,
  ShieldCheck,
  Users,
  Wrench,
  type LucideIcon,
} from 'lucide-react';
import { useState } from 'react';
import { Link } from 'react-router-dom';

const route = [
  {
    title: 'Connect sources',
    detail:
      'A product lead adds the GitHub repositories and any useful product notes. Wayfinder reads the code at a fixed commit and never runs it.',
  },
  {
    title: 'Draft the passport',
    detail:
      'AI researches the code and drafts a product passport: purpose, architecture, end-to-end scenarios and decisions. Each claim links to the exact source lines.',
  },
  {
    title: 'Review and publish',
    detail:
      'The product lead checks the draft and publishes the version learners see. Drafts stay drafts until a person approves them.',
  },
  {
    title: 'Learn',
    detail:
      'Newcomers explore the published passport and take Foundations lessons on the shared engineering basics, with interactive quizzes.',
  },
  {
    title: 'Contribute',
    detail:
      'With the product and the basics understood, a newcomer is ready to pick a first real issue and open a pull request people can review.',
  },
];

const problems = [
  {
    title: 'A new role and new concepts',
    text: 'People moving from consulting into product work learn a product and an engineering way of working at the same time.',
  },
  {
    title: 'Slow, scattered onboarding',
    text: 'Knowledge lives in repositories, chats and people’s heads. Onboarding takes long and leaves confusion behind.',
  },
  {
    title: 'Everyone asks their own AI',
    text: 'Answers differ from person to person, can be wrong or too detailed, and nobody who owns the product has checked them.',
  },
];

const features: { icon: LucideIcon; title: string; text: string }[] = [
  {
    icon: FileSearch,
    title: 'Product passports with evidence',
    text: 'Purpose, architecture, scenarios and decisions, generated from the code. Every claim shows its basis and links to the source lines it came from.',
  },
  {
    icon: ShieldCheck,
    title: 'Reviewed before learners see it',
    text: 'AI drafts, people decide. Product leads review and publish the version their learners read, so everyone starts from the same checked facts.',
  },
  {
    icon: GraduationCap,
    title: 'Foundations for everyone',
    text: 'Short lessons on Git and pull requests, code quality, CI, testing, releases, APIs and AI assistants, with interactive quizzes, progress and achievements.',
  },
  {
    icon: Users,
    title: 'Access by role',
    text: 'Admins, product leads and learners. People see only the products they belong to, and leads manage their own teams.',
  },
];

const audiences = [
  {
    title: 'For product leads',
    items: [
      'Connect repositories and context once',
      'Get a source-backed draft instead of a blank page',
      'Review, correct and publish what newcomers learn',
      'Decide who can see each product',
    ],
  },
  {
    title: 'For newcomers',
    items: [
      'See how the product works, from purpose to architecture',
      'Follow real end-to-end scenarios step by step',
      'Check any claim against the code it came from',
      'Learn the shared basics with Foundations',
    ],
  },
];

const next: { icon: LucideIcon; title: string; text: string }[] = [
  {
    icon: Bot,
    title: 'Ask your AI assistant',
    text: 'An MCP server so Claude, Codex or Cursor answer from the published passport, with citations, instead of guessing.',
  },
  {
    icon: MessagesSquare,
    title: 'Built-in chat',
    text: 'Ask questions about a product directly in Wayfinder.',
  },
  {
    icon: KeyRound,
    title: 'Sign in with Google',
    text: 'Single sign-on with company Google accounts.',
  },
  {
    icon: Wrench,
    title: 'Practice on real issues',
    text: 'Exercises that guide a newcomer through solving a real issue from the repository.',
  },
];

function RouteExplorer() {
  const [active, setActive] = useState(1);
  const step = route[active];
  return (
    <div className="landing-route">
      <p className="eyebrow">How it works</p>
      <ol className="route-steps">
        {route.map((item, index) => (
          <li key={item.title} className={index <= active ? 'reached' : ''}>
            <button
              type="button"
              className={index === active ? 'active' : ''}
              aria-pressed={index === active}
              onClick={() => setActive(index)}
            >
              <span className="route-dot" aria-hidden="true">
                {index + 1}
              </span>
              <span className="route-label">{item.title}</span>
            </button>
          </li>
        ))}
      </ol>
      <div className="route-detail" aria-live="polite">
        <strong>{step.title}</strong>
        <p>{step.detail}</p>
      </div>
    </div>
  );
}

export function Landing() {
  return (
    <div className="landing">
      <section className="landing-hero" aria-labelledby="landing-title">
        <div className="landing-hero-copy">
          <p className="eyebrow">Product onboarding, reviewed by the people who own the product</p>
          <h1 id="landing-title">Know the product. Find your way.</h1>
          <p className="landing-lead">
            Wayfinder turns a product&apos;s repositories into a reviewed product passport, adds
            shared engineering Foundations, and gives every newcomer the same trusted path to a
            first contribution.
          </p>
          <div className="landing-actions">
            <Link className="button primary" to="/login">
              Sign in
              <ArrowRight size={17} />
            </Link>
            <Link className="button secondary" to="/register">
              Create an account
            </Link>
          </div>
        </div>
        <RouteExplorer />
      </section>

      <section className="landing-section" aria-labelledby="problem-title">
        <p className="eyebrow">Why Wayfinder</p>
        <h2 id="problem-title">Onboarding should not depend on who you ask.</h2>
        <ul className="landing-problems">
          {problems.map((item) => (
            <li key={item.title}>
              <strong>{item.title}</strong>
              <p>{item.text}</p>
            </li>
          ))}
        </ul>
        <p className="landing-answer">
          <BookOpenCheck size={20} aria-hidden="true" />
          Wayfinder gives product owners control over what newcomers learn, and gives newcomers one
          trusted place to learn it.
        </p>
      </section>

      <section className="landing-section" aria-labelledby="features-title">
        <p className="eyebrow">What is inside</p>
        <h2 id="features-title">Curated knowledge, connected to the code.</h2>
        <ul className="landing-features">
          {features.map((item) => (
            <li key={item.title}>
              <span className="landing-icon" aria-hidden="true">
                <item.icon size={20} />
              </span>
              <strong>{item.title}</strong>
              <p>{item.text}</p>
            </li>
          ))}
        </ul>
      </section>

      <section className="landing-section" aria-labelledby="audience-title">
        <p className="eyebrow">Two sides, one source</p>
        <h2 id="audience-title">Built for the people who teach and the people who learn.</h2>
        <div className="landing-audiences">
          {audiences.map((group) => (
            <div key={group.title}>
              <h3>{group.title}</h3>
              <ul>
                {group.items.map((item) => (
                  <li key={item}>
                    <ListChecks size={16} aria-hidden="true" />
                    {item}
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
      </section>

      <section className="landing-section" aria-labelledby="next-title">
        <p className="eyebrow">Roadmap</p>
        <h2 id="next-title">Coming next.</h2>
        <p className="landing-note">These are plans, not features you can use today.</p>
        <ul className="landing-next">
          {next.map((item) => (
            <li key={item.title}>
              <item.icon size={18} aria-hidden="true" />
              <span>
                <strong>{item.title}</strong>
                {item.text}
              </span>
            </li>
          ))}
        </ul>
      </section>

      <section className="landing-cta" aria-labelledby="cta-title">
        <h2 id="cta-title">Ready to find your way?</h2>
        <p>
          Sign in to see the products you have access to, or create an account and ask a lead to add
          you.
        </p>
        <div className="landing-actions">
          <Link className="button primary" to="/login">
            Sign in
            <ArrowRight size={17} />
          </Link>
          <Link className="button secondary" to="/register">
            Create an account
          </Link>
        </div>
      </section>
    </div>
  );
}
