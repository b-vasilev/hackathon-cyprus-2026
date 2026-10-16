import { api } from '../lib/api';
import { loadFlags, replaceFlags, replaceUnlockDates, unlockDates } from './achievements';
import { mergeStates, type LearningState } from './merge';
import {
  learningChanged,
  loadProgress,
  replaceProgress,
  type ModuleProgress,
  type Progress,
} from '.';

type LearningModule = LearningState['modules'][string];

// The browser keeps a copy for instant rendering; it belongs to one account at a time.
const ownerKey = 'of.foundations.owner';
const uploadDelay = 800;

let owner: string | undefined;
let timer: ReturnType<typeof setTimeout> | undefined;

function readOwner(): string | null {
  try {
    return localStorage.getItem(ownerKey);
  } catch {
    return null;
  }
}

function toModule(item: ModuleProgress): LearningModule {
  return {
    lessons: item.lessons,
    quiz_passed: item.quizPassed,
    visited: item.visited,
    best_score: item.bestScore,
    failed_quiz: item.failedQuiz,
    comeback: item.comeback,
  };
}

function fromModule(item: LearningModule): ModuleProgress {
  return {
    lessons: item.lessons ?? [],
    quizPassed: item.quiz_passed,
    visited: item.visited,
    bestScore: item.best_score,
    failedQuiz: item.failed_quiz,
    comeback: item.comeback,
  };
}

function localState(): LearningState {
  const modules: LearningState['modules'] = {};
  for (const [id, item] of Object.entries(loadProgress())) {
    if (item !== undefined) {
      modules[id] = toModule(item);
    }
  }
  const unlocked: LearningState['unlocked'] = {};
  for (const [id, date] of Object.entries(unlockDates())) {
    if (date !== undefined) {
      unlocked[id] = date;
    }
  }
  return { modules, flags: loadFlags(), unlocked };
}

const emptyState: LearningState = { modules: {}, flags: [], unlocked: {} };

function applyState(state: LearningState) {
  const progress: Progress = {};
  for (const [id, item] of Object.entries(state.modules)) {
    progress[id] = fromModule(item);
  }
  replaceProgress(progress);
  replaceFlags(state.flags ?? []);
  replaceUnlockDates(state.unlocked);
}

// The server merges each save into the account copy, so the response also carries
// progress made on other devices; it is merged into the browser copy.
function upload(keepalive = false): Promise<void> {
  timer = undefined;
  const sender = owner;
  if (sender === undefined) {
    return Promise.resolve();
  }
  return api<LearningState>('/me/learning', {
    method: 'PUT',
    body: JSON.stringify(localState()),
    keepalive,
  })
    .then((saved) => {
      if (owner === sender) {
        applyState(mergeStates(saved, localState()));
      }
    })
    .catch(() => {
      // The next change or visit retries; the local copy keeps the progress meanwhile.
    });
}

/** Saves a pending change now, for example before signing out. */
export function flushLearning(): Promise<void> {
  if (timer === undefined) {
    return Promise.resolve();
  }
  clearTimeout(timer);
  return upload();
}

function scheduleUpload() {
  if (owner === undefined) {
    return;
  }
  if (timer !== undefined) {
    clearTimeout(timer);
  }
  timer = setTimeout(() => void upload(), uploadDelay);
}

globalThis.addEventListener(learningChanged, scheduleUpload);
globalThis.addEventListener('pagehide', () => {
  if (timer !== undefined) {
    clearTimeout(timer);
    void upload(true);
  }
});

/**
 * Loads the account's progress and merges it with the browser copy. Progress saved before
 * accounts existed (no owner yet) is adopted by the first account that signs in; a copy that
 * belongs to another account is replaced.
 */
export async function syncLearning(userId: string): Promise<true> {
  owner = undefined;
  const saved = await api<LearningState>('/me/learning');
  const previous = readOwner();
  const local = previous === null || previous === userId ? localState() : emptyState;
  const merged = mergeStates(saved, local);
  applyState(merged);
  try {
    localStorage.setItem(ownerKey, userId);
  } catch {
    // Storage can be unavailable; the account copy still works for this visit.
  }
  owner = userId;
  if (JSON.stringify(merged) !== JSON.stringify(saved)) {
    void upload();
  }
  return true;
}
