import { registerHooks } from 'node:module';

// Server-rendered component tests do not need stylesheet contents.
registerHooks({
  load(url, context, nextLoad) {
    if (url.endsWith('.css')) {
      return { format: 'module', source: '', shortCircuit: true };
    }
    return nextLoad(url, context);
  },
});
