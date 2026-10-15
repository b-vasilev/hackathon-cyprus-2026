import Markdown, { type Components } from 'react-markdown';
import remarkGfm from 'remark-gfm';

// Only absolute web/email URLs are useful here. Relative source paths are inspected
// through evidence, rather than accidentally navigating inside the portal.
function safeURL(value: string): string {
  try {
    const url = new URL(value);
    return ['https:', 'http:', 'mailto:'].includes(url.protocol) ? url.href : '';
  } catch {
    return '';
  }
}

const components: Components = {
  // The page owns h1 and the chapter owns h2; prose headings start at h3.
  h1: 'h3',
  h2: 'h3',
  h3: 'h4',
  h4: 'h5',
  h5: 'h6',
  h6: 'h6',
  table: ({ children }) => (
    <div className="passport-table-scroll">
      <table className="passport-table">{children}</table>
    </div>
  ),
  a: ({ href, children }) =>
    href !== undefined && href !== '' ? (
      <a href={href} target="_blank" rel="noopener noreferrer">
        {children}
      </a>
    ) : (
      <span>{children}</span>
    ),
  // Generated prose cannot load remote media or execute authored components.
  img: ({ alt }) => (alt !== undefined && alt !== '' ? <span>{alt}</span> : null),
};

export function SafeMarkdown({ children }: Readonly<{ children: string }>) {
  return (
    <div className="document-prose">
      <Markdown skipHtml remarkPlugins={[remarkGfm]} urlTransform={safeURL} components={components}>
        {children}
      </Markdown>
    </div>
  );
}
