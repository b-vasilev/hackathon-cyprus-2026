import { CircleAlert } from 'lucide-react';
import { generationMessage } from '../lib/generationMessages';
import type { Run } from '../lib/api';

export function PartialResearchNotice({
  events,
  label,
}: Readonly<{ events: Run['events'] | undefined; label: string }>) {
  const partial = (events ?? []).filter((event) => event.type === 'research_partial');
  if (partial.length === 0) {
    return null;
  }
  const messages = [
    ...new Set(partial.map((event) => generationMessage(event.message).trim())),
  ].filter((message) => message !== '');
  return (
    <div className="notice warning" role="status">
      <CircleAlert size={20} aria-hidden="true" />
      <div>
        <strong>{label}</strong>
        <p>Some source areas still need investigation. Treat this draft as partial.</p>
        {messages.map((message) => (
          <p key={message}>{message}</p>
        ))}
      </div>
    </div>
  );
}
