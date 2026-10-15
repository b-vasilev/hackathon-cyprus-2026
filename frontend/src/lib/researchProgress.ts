import type { Run } from './api';

export type ResearchProgress = NonNullable<Run['research']>;
export type ResearchArea = ResearchProgress['areas'][number];

// Keep source IDs for lookups; replace them only in user-visible text,
// including progress saved before readable task labels were introduced.
export function researchTextFormatter(progress: ResearchProgress): (text: string) => string {
  const repositories = new Map<string, string>();
  for (const area of progress.areas) {
    if (!area.id.startsWith('scope/') && !area.id.startsWith('responsibility/')) {
      continue;
    }
    const id = area.id.split('/').at(1);
    if (id === undefined || repositories.has(id)) {
      continue;
    }
    let name = area.repository;
    try {
      const url = new URL(name);
      name = url.pathname.split('/').findLast((segment) => segment !== '') ?? url.hostname;
    } catch {
      // Context sources may have a readable label instead of a URL.
    }
    repositories.set(id, name.replace(/\.git$/, ''));
  }
  return (text) => {
    let result = text;
    for (const [id, name] of repositories) {
      result = result === id ? name : result.replaceAll(`${id}/`, `${name}/`);
    }
    return result;
  };
}

export const researchStatusLabels = {
  not_started: 'To explore',
  investigating: 'Investigating',
  checked: 'Evidence checked',
  blocked: 'Source gap',
  excluded: 'Outside scope',
} as const;

export const questionStatusLabels = {
  pending: 'Open',
  answered: 'Answered with sources',
  not_applicable: 'Not applicable',
  blocked: 'No match in recorded searches',
  excluded: 'Excluded with reason',
} as const;

export function researchFocusLabel(progress: ResearchProgress): string {
  const area = progress.areas.find(
    (area) => area.id === `scope/${progress.focus.area}` || area.id === progress.focus.area
  );
  return area === undefined ? progress.focus.area : `${area.repository} · ${area.name}`;
}

export function researchCounts(areas: ResearchArea[]) {
  const counts = { checked: 0, investigating: 0, not_started: 0, blocked: 0, excluded: 0 };
  let answered = 0;
  let questions = 0;
  for (const area of areas) {
    counts[area.status]++;
    for (const question of area.questions) {
      if (question.status !== 'excluded') {
        questions++;
      }
      if (question.status === 'answered' || question.status === 'not_applicable') {
        answered++;
      }
    }
  }
  return { ...counts, answered, questions };
}

export function orderedResearchAreas(progress: ResearchProgress): ResearchArea[] {
  const priority = { investigating: 0, not_started: 1, blocked: 2, checked: 3, excluded: 4 };
  const focused = (area: ResearchArea) =>
    area.name === progress.focus.area ||
    area.id === progress.focus.area ||
    area.id === `scope/${progress.focus.area}`;
  return progress.areas.toSorted((left, right) => {
    if (focused(left) !== focused(right)) {
      return focused(left) ? -1 : 1;
    }
    const difference = priority[left.status] - priority[right.status];
    return difference === 0 ? left.name.localeCompare(right.name) : difference;
  });
}
