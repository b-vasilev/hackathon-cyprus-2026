-- Run status is polled while research is active. Keep counts and token totals
-- scoped to that run instead of scanning source and call history for all runs.
CREATE INDEX IF NOT EXISTS source_snapshots_run_id ON source_snapshots(run_id, id);
CREATE INDEX IF NOT EXISTS llm_calls_run_tokens ON llm_calls(run_id, input_tokens, output_tokens);
