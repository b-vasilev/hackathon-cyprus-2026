import ELK, { type ElkNode } from 'elkjs/lib/elk-api.js';
import workerUrl from 'elkjs/lib/elk-worker.min.js?url';

// Keep the layout engine off the main thread and release it after each layout.
export async function layoutGraph(graph: ElkNode): Promise<ElkNode> {
  const engine = new ELK({ workerUrl });
  try {
    return await engine.layout(graph);
  } finally {
    engine.terminateWorker();
  }
}
