import { useState } from 'react';
import { ExternalLink } from './ui';
import type { Revision } from '../lib/api';
import './SourceCoverage.css';

type Snapshot = NonNullable<Revision['snapshots']>[number];

function fileState(cited: boolean, read: boolean) {
  if (cited) {
    return 'Cited';
  }
  if (read) {
    return 'Read';
  }
  return 'Indexed';
}

export function SourceCoverage({ snapshot }: Readonly<{ snapshot: Snapshot }>) {
  const [query, setQuery] = useState('');
  const [state, setState] = useState('all');
  const [page, setPage] = useState(0);
  const files = snapshot.coverage.files ?? [];
  const summary =
    files.length === 0
      ? `Source files · ${snapshot.coverage.file_count} selected in this earlier version`
      : `Source files · ${snapshot.coverage.file_count} indexed · ${snapshot.coverage.read_files} read · ${snapshot.coverage.cited_files} cited`;
  const matching = files.filter(
    (file) =>
      file.path.toLowerCase().includes(query.toLowerCase()) &&
      (state === 'all' ||
        (state === 'read' && (file.read_ranges?.length ?? 0) > 0) ||
        (state === 'cited' && file.cited))
  );
  const pageSize = 40;
  const start = Math.min(page * pageSize, Math.max(matching.length - 1, 0));
  return (
    <details className="source-coverage">
      <summary>{summary}</summary>
      {files.length === 0 ? (
        <p className="field-help">This earlier version has no file-level reading history.</p>
      ) : (
        <>
          <p className="field-help">
            Indexed files were available to the researcher. Read files returned source lines through
            the tools. Cited files support the saved passport.
          </p>
          <div className="source-coverage-filters">
            <label>
              Find a file
              <input
                type="search"
                value={query}
                onChange={(event) => {
                  setQuery(event.target.value);
                  setPage(0);
                }}
                placeholder="Path or filename"
              />
            </label>
            <label>
              Show
              <select
                value={state}
                onChange={(event) => {
                  setState(event.target.value);
                  setPage(0);
                }}
              >
                <option value="all">All indexed files</option>
                <option value="read">Read files</option>
                <option value="cited">Cited files</option>
              </select>
            </label>
          </div>
          <ul className="source-coverage-files">
            {matching.slice(start, start + pageSize).map((file) => (
              <li key={file.id}>
                <div>
                  {snapshot.locator.startsWith('https://github.com/') ? (
                    <ExternalLink
                      href={`${snapshot.locator.replace(/\.git$/, '')}/blob/${snapshot.revision}/${file.path
                        .split('/')
                        .map((part) => encodeURIComponent(part))
                        .join('/')}`}
                    >
                      {file.path}
                    </ExternalLink>
                  ) : (
                    <span>{file.path}</span>
                  )}
                  {(file.read_ranges?.length ?? 0) > 0 && (
                    <span className="meta">
                      Read lines{' '}
                      {file.read_ranges
                        ?.map((range) => `${range.start_line}–${range.end_line}`)
                        .join(', ')}
                    </span>
                  )}
                </div>
                <span className="meta">
                  {fileState(file.cited, (file.read_ranges?.length ?? 0) > 0)}
                </span>
              </li>
            ))}
          </ul>
          <div className="source-coverage-pages">
            <span className="meta">
              {matching.length === 0 ? 0 : start + 1}–{Math.min(start + pageSize, matching.length)}{' '}
              of {matching.length} files
            </span>
            <button
              type="button"
              className="button quiet"
              disabled={page === 0}
              onClick={() => setPage(page - 1)}
            >
              Previous
            </button>
            <button
              type="button"
              className="button quiet"
              disabled={start + pageSize >= matching.length}
              onClick={() => setPage(page + 1)}
            >
              Next
            </button>
          </div>
        </>
      )}
    </details>
  );
}
