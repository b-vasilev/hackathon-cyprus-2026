import { useQuery } from '@tanstack/react-query';
import { ApiError, api, type LearningPack } from './api';

interface Entry {
  id: string;
  title: string;
}

export const packKey = 'learning-pack';
export const practiceId = 'practice';
export const practiceTitle = 'Practice: quiz and first task';

/** The published pack for learners; a missing pack is a normal state, not an error. */
export function usePublishedPack(productId: string, revisionId: string) {
  return useQuery({
    queryKey: [packKey, 'published', productId, revisionId],
    queryFn: async ({ signal }) => {
      try {
        return await api<LearningPack>(`/products/${productId}/learning-pack`, {
          signal,
        });
      } catch (error) {
        if (error instanceof ApiError && error.status === 404) {
          return null;
        }
        throw error;
      }
    },
    enabled: revisionId !== '',
    select: (pack) => (pack?.revision_id === revisionId ? pack : null),
  });
}

/** Practice follows the chapters of a current passport when the viewer can use it. */
export function withPractice(
  chapters: Entry[] | undefined,
  sections: Entry[],
  visible: boolean
): Entry[] {
  if (chapters === undefined) {
    return sections;
  }
  return visible ? [...chapters, { id: practiceId, title: practiceTitle }] : chapters;
}
