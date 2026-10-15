import {
  ArrowRight,
  Award,
  CircleHelp,
  Compass,
  Dices,
  Footprints,
  Lock,
  MapPinOff,
  RotateCcw,
  ShieldCheck,
  Sparkles,
  Target,
  Terminal,
  X,
  type LucideIcon,
} from 'lucide-react';
import { useCallback, useEffect, useLayoutEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import {
  achievements,
  latestUnlocked,
  scaredMouse,
  syncUnlockDates,
  unlockEvent,
  type Achievement,
  type Snapshot,
} from '../foundations/achievements';
import { Dialog } from './ui';

const icons: Record<Achievement['icon'], LucideIcon> = {
  'first-step': Footprints,
  explorer: Compass,
  perfect: Target,
  comeback: RotateCcw,
  ai: ShieldCheck,
  complete: Award,
  'off-route': MapPinOff,
  perfectionist: Sparkles,
  bold: Dices,
  curious: Terminal,
};

const dateFormat = new Intl.DateTimeFormat('en', { day: 'numeric', month: 'short' });

function BadgeIcon({ item, size = 20 }: Readonly<{ item: Achievement; size?: number }>) {
  const Icon = icons[item.icon];
  return (
    <span className="achievement-icon" aria-hidden="true">
      <Icon size={size} />
    </span>
  );
}

function LockedIcon({ hidden, size }: Readonly<{ hidden: boolean; size: number }>) {
  const Icon = hidden ? CircleHelp : Lock;
  return (
    <span className="achievement-icon locked" aria-hidden="true">
      <Icon size={size} />
    </span>
  );
}

// Latest badge and a link to all achievements, for the top of the Foundations page.
export function AchievementSummary({ state }: Readonly<{ state: Snapshot }>) {
  const latest = latestUnlocked(state);
  const unlocked = achievements.filter((item) => item.unlocked(state)).length;
  return (
    <Link to="/foundations/achievements" className="achievement-summary">
      {latest === undefined ? (
        <LockedIcon hidden={false} size={18} />
      ) : (
        <BadgeIcon item={latest} size={18} />
      )}
      <span>
        <span className="achievement-summary-label">
          {latest === undefined ? 'No achievements yet' : 'Latest achievement'}
        </span>
        <strong>{latest === undefined ? 'Finish a lesson to get your first' : latest.title}</strong>
      </span>
      <span className="achievement-summary-count">
        {unlocked} of {achievements.length}
        <ArrowRight size={16} aria-hidden="true" />
      </span>
    </Link>
  );
}

function Picture({ item }: Readonly<{ item: Achievement }>) {
  if (item.image !== undefined) {
    return (
      <div className="achievement-picture image">
        <img src={item.image.src} alt={item.image.alt} loading="lazy" />
      </div>
    );
  }
  return (
    <pre className="achievement-picture">
      {item.picture.map((line) => (
        <span key={line.text} className={`picture-line ${line.tone ?? ''}`}>
          {line.text}
        </span>
      ))}
    </pre>
  );
}

// All achievements with their pictures; locked ones keep the picture hidden.
export function AchievementGallery({ state }: Readonly<{ state: Snapshot }>) {
  const dates = syncUnlockDates(state);
  return (
    <ul className="achievement-gallery">
      {achievements.map((item) => {
        const open = item.unlocked(state);
        const secret = item.hidden === true && !open;
        const date = dates[item.id];
        return (
          <li key={item.id} className={`achievement-card ${open ? 'unlocked' : 'locked'}`}>
            {open ? (
              <Picture item={item} />
            ) : (
              <div className="achievement-picture hidden" aria-hidden="true">
                {secret ? <CircleHelp size={22} /> : <Lock size={22} />}
                <span>???</span>
              </div>
            )}
            <div className="achievement-card-body">
              <div className="achievement-card-title">
                {open ? (
                  <BadgeIcon item={item} size={16} />
                ) : (
                  <LockedIcon hidden={secret} size={16} />
                )}
                <strong>{secret ? 'Hidden achievement' : item.title}</strong>
              </div>
              {open ? (
                <>
                  <p className="achievement-joke">{item.joke}</p>
                  <p className="achievement-meta">
                    {item.description}
                    {date !== undefined && ` Unlocked ${dateFormat.format(new Date(date))}.`}
                  </p>
                </>
              ) : (
                <p className="achievement-meta">
                  <span className="visually-hidden">Locked. </span>
                  {secret ? `Clue: ${item.hint}` : item.hint}
                </p>
              )}
            </div>
          </li>
        );
      })}
    </ul>
  );
}

// App-wide notice for achievements announced through `withAchievements`.
export function AchievementCenter() {
  const [items, setItems] = useState<Achievement[]>([]);
  const close = useCallback(() => setItems([]), []);

  // Layout effects run before page effects, so unlocks on first render are not missed.
  useLayoutEffect(() => {
    function listen(event: Event) {
      const ids = (event as CustomEvent<string[]>).detail;
      setItems(achievements.filter((item) => ids.includes(item.id)));
    }
    globalThis.addEventListener(unlockEvent, listen);
    return () => globalThis.removeEventListener(unlockEvent, listen);
  }, []);

  useEffect(() => {
    if (items.length === 0) {
      return undefined;
    }
    const timer = globalThis.setTimeout(close, 6000);
    return () => globalThis.clearTimeout(timer);
  }, [items, close]);

  if (items.length === 0) {
    return null;
  }
  return (
    <div className="achievement-toast" role="status">
      <div className="achievement-toast-list">
        {items.map((item) => (
          <div key={item.id} className="achievement-toast-item">
            <BadgeIcon item={item} size={18} />
            <span>
              <span className="achievement-toast-label">
                {item.hidden === true ? 'Hidden achievement unlocked' : 'Achievement unlocked'}
              </span>
              <strong>{item.title}</strong>
            </span>
          </div>
        ))}
        <Link
          to="/foundations/achievements"
          className="text-link achievement-toast-link"
          onClick={close}
        >
          View achievements
          <ArrowRight size={14} />
        </Link>
      </div>
      <button type="button" className="icon-button" onClick={close} aria-label="Close">
        <X size={16} />
      </button>
    </div>
  );
}

// Shown when a quiz ends with zero correct answers.
export function BoldStrategyDialog({
  total,
  onClose,
}: Readonly<{ total: number; onClose: () => void }>) {
  return (
    <Dialog title={`0 of ${total}. Bold strategy.`} onClose={onClose}>
      <figure className="meme">
        <img src={scaredMouse.src} alt={scaredMouse.alt} />
        <figcaption>Me, looking at my quiz result.</figcaption>
      </figure>
      <p className="meme-text">
        It happens to everyone once. Read the lessons again and retry the quiz. If you pass, you get
        the Comeback achievement.
      </p>
      <div className="form-actions">
        <button type="button" className="button primary" onClick={onClose}>
          Back to the quiz
        </button>
      </div>
    </Dialog>
  );
}
