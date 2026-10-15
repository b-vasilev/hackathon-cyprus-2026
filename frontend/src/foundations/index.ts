import { aiAssistants } from './content/aiAssistants';
import { ciCd } from './content/ciCd';
import { codeQuality } from './content/codeQuality';
import { git } from './content/git';
import { http } from './content/http';
import { releases } from './content/releases';
import { testing } from './content/testing';
import type { FoundationModule } from './types';

// Ordered as a suggested learning path; each module also stands on its own.
export const foundationModules: FoundationModule[] = [
  git,
  codeQuality,
  ciCd,
  testing,
  releases,
  http,
  aiAssistants,
];

export function findModule(id: string | undefined) {
  return foundationModules.find((module) => module.id === id);
}

const storageKey = 'of.foundations.progress';
const legacyPassedKey = 'of.foundations.passed';

export interface ModuleProgress {
  lessons: string[];
  quizPassed: boolean;
  visited: boolean;
  // Best quiz result as a share of correct answers, 0 to 1.
  bestScore: number;
  // A quiz attempt below the pass mark happened.
  failedQuiz: boolean;
  // The quiz was passed after an earlier failed attempt.
  comeback: boolean;
}

export type Progress = Partial<Record<string, ModuleProgress>>;

const empty: ModuleProgress = {
  lessons: [],
  quizPassed: false,
  visited: false,
  bestScore: 0,
  failedQuiz: false,
  comeback: false,
};

function strings(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item) => typeof item === 'string') : [];
}

// The browser keeps a copy for instant rendering; foundations/sync.ts saves it to the account.
export function loadProgress(): Progress {
  const progress: Progress = {};
  try {
    const stored: unknown = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
    if (typeof stored === 'object' && stored !== null) {
      for (const [id, value] of Object.entries(stored as Record<string, unknown>)) {
        const item = (value ?? {}) as Partial<Record<keyof ModuleProgress, unknown>>;
        progress[id] = {
          lessons: strings(item.lessons),
          quizPassed: item.quizPassed === true,
          visited: item.visited === true,
          bestScore: typeof item.bestScore === 'number' ? item.bestScore : 0,
          failedQuiz: item.failedQuiz === true,
          comeback: item.comeback === true,
        };
      }
    }
    for (const id of strings(JSON.parse(localStorage.getItem(legacyPassedKey) ?? '[]'))) {
      progress[id] = { ...empty, ...progress[id], quizPassed: true };
    }
  } catch {
    // Storage can be unavailable or corrupted; start empty.
  }
  return progress;
}

// Announced after every local change so the account copy can be updated.
export const learningChanged = 'wayfinder:learning-changed';

function write(progress: Progress) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(progress));
    localStorage.removeItem(legacyPassedKey);
  } catch {
    // Storage can be unavailable; the account copy is the durable one.
  }
}

function save(progress: Progress) {
  write(progress);
  globalThis.dispatchEvent(new Event(learningChanged));
}

// Replaces the local copy with progress loaded from the account, without announcing a change.
export function replaceProgress(progress: Progress) {
  write(progress);
}

function update(moduleId: string, change: (current: ModuleProgress) => ModuleProgress): Progress {
  const progress = loadProgress();
  progress[moduleId] = change(progress[moduleId] ?? empty);
  save(progress);
  return progress;
}

export function markVisited(moduleId: string): Progress {
  return update(moduleId, (current) => ({ ...current, visited: true }));
}

export function markLessonDone(moduleId: string, lessonId: string): Progress {
  return update(moduleId, (current) =>
    current.lessons.includes(lessonId)
      ? current
      : { ...current, lessons: [...current.lessons, lessonId] }
  );
}

export function recordQuiz(moduleId: string, score: number, total: number, passRatio: number) {
  const share = total === 0 ? 0 : score / total;
  const passed = share >= passRatio;
  return update(moduleId, (current) => ({
    ...current,
    quizPassed: current.quizPassed || passed,
    bestScore: Math.max(current.bestScore, share),
    failedQuiz: current.failedQuiz || !passed,
    comeback: current.comeback || (passed && current.failedQuiz),
  }));
}

export type ModuleState = 'not-started' | 'in-progress' | 'completed';

// Each lesson and the quiz count as one step.
export function moduleStatus(module: FoundationModule, progress: Progress) {
  const item = progress[module.id];
  const lessons = module.lessons.filter((lesson) => item?.lessons.includes(lesson.id)).length;
  const quizPassed = item?.quizPassed ?? false;
  const done = lessons + (quizPassed ? 1 : 0);
  const total = module.lessons.length + 1;
  let state: ModuleState = 'in-progress';
  if (done === 0) {
    state = 'not-started';
  } else if (done === total) {
    state = 'completed';
  }
  return { lessons, quizPassed, done, total, state };
}
