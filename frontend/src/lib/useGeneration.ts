import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { api, type Run } from './api';
import { activeRun, mergeRun } from './generation';

export function useGeneration(runId: string) {
  const queryClient = useQueryClient();
  const [connection, setConnection] = useState({ runId: '', state: 'connecting' });
  const streamState = connection.runId === runId ? connection.state : 'connecting';
  const run = useQuery({
    queryKey: ['run', runId],
    queryFn: ({ signal }) => api<Run>(`/generations/${runId}`, { signal }),
    enabled: runId !== '',
    structuralSharing: (previous, incoming) =>
      mergeRun(previous as Run | undefined, incoming as Run),
    refetchInterval: (query) => {
      const needsUpdates =
        !(query.state.status === 'error' && query.state.data === undefined) &&
        (query.state.data === undefined || activeRun(query.state.data.status));
      return needsUpdates && streamState !== 'live' ? 5000 : false;
    },
  });
  const finished = run.data !== undefined && !activeRun(run.data.status);

  useEffect(() => {
    if (runId === '' || finished) {
      return;
    }
    let source: EventSource;
    let retry: ReturnType<typeof globalThis.setTimeout> | undefined;
    let stopped = false;
    let cursor = queryClient.getQueryData<Run>(['run', runId])?.events.at(-1)?.id ?? 0;
    const connect = () => {
      source = new EventSource(`/api/generations/${runId}/events?after=${cursor}`);
      source.addEventListener('open', () => setConnection({ runId, state: 'live' }));
      source.addEventListener('error', () => {
        if (stopped) {
          return;
        }
        setConnection({ runId, state: 'reconnecting' });
        // Network drops reconnect natively. HTTP errors can close EventSource
        // permanently, so recreate it with the saved cursor in that case.
        if (source.readyState === EventSource.CLOSED) {
          retry = globalThis.setTimeout(connect, 2000);
        }
      });
      const receive = (message: MessageEvent<string>) => {
        const incoming = JSON.parse(message.data) as Run;
        if (stopped || incoming.id !== runId) {
          return;
        }
        cursor = Number(message.lastEventId);
        queryClient.setQueryData<Run>(['run', runId], (previous) => mergeRun(previous, incoming));
        if (!activeRun(incoming.status)) {
          source.close();
          setConnection({ runId, state: 'finished' });
        }
      };
      source.addEventListener('snapshot', receive);
      source.addEventListener('update', receive);
    };
    connect();
    return () => {
      stopped = true;
      globalThis.clearTimeout(retry);
      source.close();
    };
  }, [runId, queryClient, finished]);

  return { run, streamState };
}
