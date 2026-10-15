import { AlertCircle, ArrowUpRight, Route, X } from 'lucide-react';
import { useEffect, useId, useRef, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { message } from '../lib/api';

export function Brand() {
  return (
    <Link to="/" className="brand" aria-label="Wayfinder home">
      <img src="/brand/wayfinder-mark-violet.svg" width={32} height={32} alt="" />
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
      <span>{message(error)}</span>
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
  const heading = useId();
  useEffect(() => {
    const dialog = ref.current;
    dialog?.showModal();
    return () => dialog?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className={`dialog ${wide ? 'wide' : ''}`}
      aria-labelledby={heading}
      onCancel={onClose}
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
