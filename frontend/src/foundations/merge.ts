import type { components } from '../api.generated';

export type LearningState = components['schemas']['LearningState'];

// Combines two copies without losing anything either side has learned.
export function mergeStates(a: LearningState, b: LearningState): LearningState {
  const modules: Partial<LearningState['modules']> = { ...a.modules };
  for (const [id, item] of Object.entries(b.modules)) {
    const current = modules[id];
    modules[id] =
      current === undefined
        ? item
        : {
            lessons: [...new Set([...(current.lessons ?? []), ...(item.lessons ?? [])])],
            quiz_passed: current.quiz_passed || item.quiz_passed,
            visited: current.visited || item.visited,
            best_score: Math.max(current.best_score, item.best_score),
            failed_quiz: current.failed_quiz || item.failed_quiz,
            comeback: current.comeback || item.comeback,
          };
  }
  const unlocked: Partial<LearningState['unlocked']> = { ...a.unlocked };
  for (const [id, date] of Object.entries(b.unlocked)) {
    const current = unlocked[id];
    unlocked[id] = current === undefined || date < current ? date : current;
  }
  return {
    modules: modules as LearningState['modules'],
    flags: [...new Set([...(a.flags ?? []), ...(b.flags ?? [])])],
    unlocked: unlocked as LearningState['unlocked'],
  };
}
