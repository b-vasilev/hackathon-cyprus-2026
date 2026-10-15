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
}

export type Progress = Partial<Record<string, ModuleProgress>>;

function strings(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item) => typeof item === 'string') : [];
}

// Progress is kept in this browser only until learner accounts exist.
export function loadProgress(): Progress {
  const progress: Progress = {};
  try {
    const stored: unknown = JSON.parse(localStorage.getItem(storageKey) ?? '{}');
    if (typeof stored === 'object' && stored !== null) {
      for (const [id, value] of Object.entries(stored as Record<string, unknown>)) {
        const item = (value ?? {}) as Partial<Record<keyof ModuleProgress, unknown>>;
        progress[id] = { lessons: strings(item.lessons), quizPassed: item.quizPassed === true };
      }
    }
    for (const id of strings(JSON.parse(localStorage.getItem(legacyPassedKey) ?? '[]'))) {
      progress[id] = { lessons: progress[id]?.lessons ?? [], quizPassed: true };
    }
  } catch {
    // Storage can be unavailable or corrupted; start empty.
  }
  return progress;
}

function save(progress: Progress) {
  try {
    localStorage.setItem(storageKey, JSON.stringify(progress));
  } catch {
    // Storage can be unavailable; progress is a convenience only.
  }
}

export function markLessonDone(moduleId: string, lessonId: string): Progress {
  const progress = loadProgress();
  const current = progress[moduleId] ?? { lessons: [], quizPassed: false };
  if (!current.lessons.includes(lessonId)) {
    progress[moduleId] = { ...current, lessons: [...current.lessons, lessonId] };
    save(progress);
  }
  return progress;
}

export function markQuizPassed(moduleId: string): Progress {
  const progress = loadProgress();
  progress[moduleId] = { lessons: progress[moduleId]?.lessons ?? [], quizPassed: true };
  save(progress);
  return progress;
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
