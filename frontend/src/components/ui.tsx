import { AlertCircle, ArrowUpRight, ChevronDown, Route, X } from 'lucide-react';
import { useEffect, useId, useRef, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { message } from '../lib/api';
import { generationMessage } from '../lib/generationMessages';

export function Brand() {
  return (
    <Link to="/" className="brand" aria-label="Wayfinder home">
      <img src="/brand/wayfinder-mark-violet.svg" width={38} height={38} alt="" />
      <span>
        <strong>Wayfinder</strong>
      </span>
    </Link>
  );
}

export function PageHeading({
  eyebrow,
  title,
  description,
  action,
}: Readonly<{
  eyebrow: string;
  title: string;
  description: string;
  action?: ReactNode;
}>) {
  return (
    <div className="page-heading">
      <div>
        <p className="eyebrow">{eyebrow}</p>
        <h1>{title}</h1>
        <p className="page-description">{description}</p>
      </div>
      {action}
    </div>
  );
}

export function ErrorNotice({ error }: Readonly<{ error: unknown }>) {
  return (
    <div className="notice error" role="alert">
      <AlertCircle size={18} aria-hidden="true" />
      <span>{generationMessage(message(error))}</span>
    </div>
  );
}

export function EmptyState({
  title,
  children,
  action,
}: Readonly<{
  title: string;
  children: ReactNode;
  action?: ReactNode;
}>) {
  return (
    <div className="empty-state">
      <div className="empty-icon">
        <Route size={28} aria-hidden="true" />
      </div>
      <h2>{title}</h2>
      <p>{children}</p>
      {action}
    </div>
  );
}

export function Badge({ children, tone = '' }: Readonly<{ children: ReactNode; tone?: string }>) {
  return <span className={`badge ${tone}`}>{children}</span>;
}

export function Loading({ label = 'Loading…' }: Readonly<{ label?: string }>) {
  return (
    <div className="loading" role="status">
      <span className="spinner" />
      {label}
    </div>
  );
}

export function Dialog({
  title,
  children,
  onClose,
  wide = false,
}: Readonly<{
  title: string;
  children: ReactNode;
  onClose: () => void;
  wide?: boolean;
}>) {
  const ref = useRef<HTMLDialogElement>(null);
  const mounted = useRef(false);
  const heading = useId();
  useEffect(() => {
    const dialog = ref.current;
    mounted.current = true;
    dialog?.showModal();
    return () => {
      mounted.current = false;
      dialog?.close();
    };
  }, []);
  return (
    <dialog
      ref={ref}
      className={`dialog ${wide ? 'wide' : ''}`}
      aria-labelledby={heading}
      onCancel={(event) => {
        // Escape must go through onClose, which can refuse while an action is running;
        // a natively closed dialog would stay mounted but invisible.
        event.preventDefault();
        onClose();
      }}
      onClose={() => {
        // Browsers may close the dialog natively anyway (for example a repeated Escape).
        // If the owner keeps it mounted, show it again instead of leaving it invisible.
        onClose();
        requestAnimationFrame(() => {
          if (mounted.current && ref.current?.open === false) {
            ref.current.showModal();
          }
        });
      }}
    >
      <div className="dialog-heading">
        <h2 id={heading}>{title}</h2>
        <button type="button" className="icon-button" onClick={onClose} aria-label="Close dialog">
          <X size={20} />
        </button>
      </div>
      <div className="dialog-body">{children}</div>
    </dialog>
  );
}

export function ExternalLink({ href, children }: Readonly<{ href: string; children: ReactNode }>) {
  return (
    <a href={href} target="_blank" rel="noopener noreferrer" className="external-link">
      {children}
      <ArrowUpRight size={14} aria-hidden="true" />
    </a>
  );
}

/** A visible expand cue for a <details> summary. CSS shows the label for the current state. */
export function DisclosureHint({ show, hide }: Readonly<{ show: string; hide: string }>) {
  // The summary already announces its expanded state, so the cue is visual only.
  return (
    <span className="disclosure-hint" aria-hidden="true">
      <span className="disclosure-show">{show}</span>
      <span className="disclosure-hide">{hide}</span>
      <ChevronDown size={16} className="disclosure-chevron" />
    </span>
  );
}
