import { Trash2 } from 'lucide-react';
import { Dialog, ErrorNotice } from './ui';
import type { ReactNode } from 'react';

export function DeleteConfirmation({
  title,
  label,
  pending,
  disabled = false,
  error,
  onClose,
  onConfirm,
  children,
}: Readonly<{
  title: string;
  label: string;
  pending: boolean;
  disabled?: boolean;
  error: Error | null;
  onClose: () => void;
  onConfirm: () => void;
  children: ReactNode;
}>) {
  return (
    <Dialog
      title={title}
      onClose={() => {
        if (!pending) {
          onClose();
        }
      }}
    >
      <p>{children}</p>
      {error !== null && <ErrorNotice error={error} />}
      <div className="form-actions">
        <button type="button" className="button secondary" disabled={pending} onClick={onClose}>
          Cancel
        </button>
        <button
          type="button"
          className="text-button destructive"
          disabled={pending || disabled}
          onClick={onConfirm}
        >
          <Trash2 size={16} aria-hidden="true" />
          {pending ? 'Deleting…' : label}
        </button>
      </div>
    </Dialog>
  );
}
