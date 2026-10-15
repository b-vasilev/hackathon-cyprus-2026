import { foundationModules, loadProgress, moduleStatus, type Progress } from '.';

// One line of the small terminal-style picture shown for an unlocked achievement.
export interface PictureLine {
  text: string;
  tone?: 'prompt' | 'ok' | 'fail' | 'muted';
}

// Easter eggs that progress alone cannot show.
export type Flag = 'off-route' | 'perfectionist' | 'bold-strategy' | 'curious';

export interface Snapshot {
  progress: Progress;
  flags: Flag[];
}

export interface Achievement {
  id: string;
  title: string;
  // Hidden achievements show "???" until unlocked.
  hidden?: boolean;
  // Developer-meme style picture and caption, revealed when unlocked.
  picture: PictureLine[];
  // Optional image instead of the terminal picture.
  image?: { src: string; alt: string };
  joke: string;
  // What it is for, shown when unlocked.
  description: string;
  // How to get it (or a vague clue for hidden ones), shown while locked.
  hint: string;
  icon:
    | 'first-step'
    | 'explorer'
    | 'perfect'
    | 'comeback'
    | 'ai'
    | 'complete'
    | 'off-route'
    | 'perfectionist'
    | 'bold'
    | 'curious';
  unlocked: (state: Snapshot) => boolean;
}

const items = (progress: Progress) => foundationModules.map((module) => progress[module.id]);

const completed = (progress: Progress) =>
  foundationModules.filter((module) => moduleStatus(module, progress).state === 'completed');

export const scaredMouse = {
  src: '/easter/scared-mouse.jpg',
  alt: 'A small white mouse looking very worried',
};

export const achievements: Achievement[] = [
  {
    id: 'first-step',
    title: 'First step',
    icon: 'first-step',
    picture: [
      { text: '$ git commit -m "first lesson"', tone: 'prompt' },
      { text: '[main 1a2b3c4] first lesson' },
      { text: ' 1 file changed, 1 brain updated', tone: 'ok' },
    ],
    joke: 'One small commit for you. One giant leap for your onboarding.',
    description: 'You finished your first lesson.',
    hint: 'Finish any lesson.',
    unlocked: ({ progress }) => items(progress).some((item) => (item?.lessons.length ?? 0) > 0),
  },
  {
    id: 'explorer',
    title: 'Explorer',
    icon: 'explorer',
    picture: [
      { text: '$ ls foundations/', tone: 'prompt' },
      { text: 'git/  code-quality/  ci-cd/  testing/' },
      { text: 'releases/  http/  ai-assistants/' },
      { text: '7 modules opened, 0 left unseen', tone: 'ok' },
    ],
    joke: 'You opened every door. Reading what is behind them is the next step.',
    description: 'You opened every Foundations module.',
    hint: 'Open every module at least once.',
    unlocked: ({ progress }) => items(progress).every((item) => item?.visited === true),
  },
  {
    id: 'perfect',
    title: 'Perfect score',
    icon: 'perfect',
    picture: [
      { text: '$ npm test -- quiz', tone: 'prompt' },
      { text: 'Tests: 5 passed, 0 failed', tone: 'ok' },
      { text: 'Linter: no warnings', tone: 'ok' },
      { text: 'Reviewer: no comments', tone: 'ok' },
    ],
    joke: '100%. Even the linter has nothing to say.',
    description: 'You answered every question in a quiz correctly.',
    hint: 'Get 100% in any quiz.',
    unlocked: ({ progress }) => items(progress).some((item) => (item?.bestScore ?? 0) >= 1),
  },
  {
    id: 'comeback',
    title: 'Comeback',
    icon: 'comeback',
    picture: [
      { text: 'Run #1   quiz   ✕ failed', tone: 'fail' },
      { text: 'Run #2   quiz   ✓ passed', tone: 'ok' },
      { text: '# read the lesson again, pushed a fix', tone: 'muted' },
    ],
    joke: 'Failed once, passed anyway. That is basically how CI works.',
    description: 'You passed a quiz after a failed attempt. Mistakes are part of learning.',
    hint: 'Pass a quiz after an attempt that did not pass.',
    unlocked: ({ progress }) => items(progress).some((item) => item?.comeback === true),
  },
  {
    id: 'ai',
    title: 'AI tech',
    icon: 'ai',
    picture: [
      { text: 'AI: Here is your code. It is perfect.', tone: 'muted' },
      { text: 'You: *reads every line*', tone: 'prompt' },
      { text: 'You: *runs the tests*', tone: 'prompt' },
      { text: '1 bug found before production', tone: 'ok' },
    ],
    joke: 'You trust the AI like a new teammate: you review its code.',
    description: 'You completed Working with AI assistants.',
    hint: 'Complete the AI assistants module.',
    unlocked: ({ progress }) => completed(progress).some((module) => module.id === 'ai-assistants'),
  },
  {
    id: 'complete',
    title: 'Foundations complete',
    icon: 'complete',
    picture: [
      { text: '$ wayfinder status --all', tone: 'prompt' },
      { text: 'Foundations: 7/7 completed ✓', tone: 'ok' },
      { text: 'Ready for: real tickets', tone: 'ok' },
      { text: 'Not ready for: Friday deploys', tone: 'muted' },
    ],
    joke: 'Foundations complete. Time to pick your first real issue.',
    description: 'You completed every Foundations module.',
    hint: 'Complete all modules.',
    unlocked: ({ progress }) => completed(progress).length === foundationModules.length,
  },
  {
    id: 'off-route',
    title: 'Off the route',
    hidden: true,
    icon: 'off-route',
    picture: [
      { text: '$ curl wayfinder.local/somewhere', tone: 'prompt' },
      { text: 'HTTP/1.1 404 Not Found', tone: 'fail' },
      { text: '# here be dragons', tone: 'muted' },
    ],
    joke: 'You found the edge of the map. Real explorers always do.',
    description: 'You opened a page that does not exist.',
    hint: 'Some pages are not on any map.',
    unlocked: ({ flags }) => flags.includes('off-route'),
  },
  {
    id: 'perfectionist',
    title: 'Perfectionist',
    hidden: true,
    icon: 'perfectionist',
    picture: [
      { text: '$ reset && reset && reset && reset && reset', tone: 'prompt' },
      { text: 'Formatter: I already fixed it.', tone: 'muted' },
      { text: 'Formatter: Five times.', tone: 'muted' },
    ],
    joke: 'Five resets. The code was clean after the first one.',
    description: 'You restarted the code playground five times.',
    hint: 'Some people press reset more than once.',
    unlocked: ({ flags }) => flags.includes('perfectionist'),
  },
  {
    id: 'bold-strategy',
    title: 'Bold strategy',
    hidden: true,
    icon: 'bold',
    picture: [],
    image: scaredMouse,
    joke: 'Zero correct answers. Statistically, that is almost a skill.',
    description: 'You answered no questions correctly in a quiz. Comeback is waiting for you.',
    hint: 'Not every quiz strategy is a good one.',
    unlocked: ({ flags }) => flags.includes('bold-strategy'),
  },
  {
    id: 'curious',
    title: 'Curious engineer',
    hidden: true,
    icon: 'curious',
    picture: [
      { text: '> wayfinder.explore()', tone: 'prompt' },
      { text: 'Achievement unlocked: Curious engineer', tone: 'ok' },
      { text: '# you read the console. Most people never do.', tone: 'muted' },
    ],
    joke: 'You opened DevTools without being asked twice.',
    description: 'You found the message in the browser console and followed it.',
    hint: 'Developers know where to look.',
    unlocked: ({ flags }) => flags.includes('curious'),
  },
];

// ── Storage ───────────────────────────────────────────────────────────────────

const flagsKey = 'of.foundations.flags';
const unlockedKey = 'of.foundations.achievements';

export function loadFlags(): Flag[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(flagsKey) ?? '[]');
    return Array.isArray(value) ? (value.filter((item) => typeof item === 'string') as Flag[]) : [];
  } catch {
    return [];
  }
}

export function setFlag(flag: Flag) {
  const flags = loadFlags();
  if (!flags.includes(flag)) {
    try {
      localStorage.setItem(flagsKey, JSON.stringify([...flags, flag]));
    } catch {
      // Storage can be unavailable; easter eggs are a convenience only.
    }
  }
}

export function snapshot(): Snapshot {
  return { progress: loadProgress(), flags: loadFlags() };
}

export function unlockedIds(state: Snapshot) {
  return achievements.filter((item) => item.unlocked(state)).map((item) => item.id);
}

// When each achievement was first seen as unlocked, as ISO dates.
export function unlockDates(): Partial<Record<string, string>> {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(unlockedKey) ?? '{}');
    return typeof value === 'object' && value !== null ? value : {};
  } catch {
    return {};
  }
}

// Records a date for unlocked achievements that do not have one yet.
export function syncUnlockDates(state: Snapshot) {
  const dates = unlockDates();
  let changed = false;
  for (const id of unlockedIds(state)) {
    if (dates[id] === undefined) {
      dates[id] = new Date().toISOString();
      changed = true;
    }
  }
  if (changed) {
    try {
      localStorage.setItem(unlockedKey, JSON.stringify(dates));
    } catch {
      // Storage can be unavailable; dates are a convenience only.
    }
  }
  return dates;
}

// The most recently unlocked achievement, if any.
export function latestUnlocked(state: Snapshot) {
  const dates = syncUnlockDates(state);
  return achievements
    .filter((item) => item.unlocked(state))
    .reduce<Achievement | undefined>((latest, item) => {
      if (latest === undefined) {
        return item;
      }
      return (dates[item.id] ?? '') >= (dates[latest.id] ?? '') ? item : latest;
    }, undefined);
}

// ── Announcing unlocks ────────────────────────────────────────────────────────

export const unlockEvent = 'wayfinder:achievements';

// Runs a change, then announces achievements it unlocked to the app-wide notice.
export function withAchievements<T>(change: () => T): T {
  const before = new Set(unlockedIds(snapshot()));
  const result = change();
  const after = snapshot();
  syncUnlockDates(after);
  const fresh = unlockedIds(after).filter((id) => !before.has(id));
  if (fresh.length > 0) {
    globalThis.dispatchEvent(new CustomEvent(unlockEvent, { detail: fresh }));
  }
  return result;
}
