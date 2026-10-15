import { aiAssistants } from './content/aiAssistants';
import { ciCd } from './content/ciCd';
import { codeQuality } from './content/codeQuality';
import { git } from './content/git';
import { http } from './content/http';
import { pullRequests } from './content/pullRequests';
import { releases } from './content/releases';
import { testing } from './content/testing';
import type { FoundationModule } from './types';

// Ordered as a suggested learning path; each module also stands on its own.
export const foundationModules: FoundationModule[] = [
  git,
  pullRequests,
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

const storageKey = 'of.foundations.passed';

// Progress is kept in this browser only until learner accounts exist.
export function passedModules(): string[] {
  try {
    const value: unknown = JSON.parse(localStorage.getItem(storageKey) ?? '[]');
    return Array.isArray(value) ? value.filter((item) => typeof item === 'string') : [];
  } catch {
    return [];
  }
}

export function markPassed(id: string) {
  try {
    const passed = new Set(passedModules()).add(id);
    localStorage.setItem(storageKey, JSON.stringify([...passed]));
  } catch {
    // Storage can be unavailable; progress is a convenience only.
  }
}
