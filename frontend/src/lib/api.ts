import type { components } from '../api.generated';

export type Connection = components['schemas']['Connection'];
export type ConnectionConfig = components['schemas']['ConnectionConfig'];
export type ModelConfig = components['schemas']['Model'];
export type Product = components['schemas']['Product'];
export type ProductInput = components['schemas']['ProductInput'];
export type Run = components['schemas']['Run'];
export type Revision = components['schemas']['Revision'];
export type RevisionInfo = components['schemas']['RevisionInfo'];
export type ProductModel = components['schemas']['ProductModel'];
export type ProductObject = components['schemas']['ProductObject'];
export type Section = components['schemas']['Section'];
export type Diagram = components['schemas']['Diagram'];
export type Claim = components['schemas']['Claim'];
export type Scenario = components['schemas']['Scenario'];
export type EvidenceView = components['schemas']['EvidenceView'];
export type Parameter = components['schemas']['Parameter'];
export type ProbeResult = components['schemas']['ProbeResult'];
export type FactoryStatus = components['schemas']['FactoryStatus'];
export type SourceInput = components['schemas']['SourceInput'];
export type GitHubConnection = components['schemas']['GitHubConnection'];
export type OAuthURL = components['schemas']['OAuthURL'];

interface ErrorBody {
  code?: string;
  message?: string;
  request_id?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
    readonly requestId = ''
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export async function api<T>(path: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers);
  headers.set('Content-Type', 'application/json');
  const response = await fetch(`/api${path}`, {
    ...options,
    headers,
  });
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorBody;
    throw new ApiError(
      body.message ?? 'The request could not be completed.',
      response.status,
      body.request_id
    );
  }
  return response.json() as Promise<T>;
}

export function sourceDescription(source: SourceInput): string {
  if (source.kind === 'github') {
    return source.ref === '' ? 'Default branch' : source.ref;
  }
  return `${source.text.length.toLocaleString()} characters`;
}

export function message(error: unknown): string {
  return error instanceof Error ? error.message : 'Something went wrong. Please try again.';
}

export const protocolLabels: Record<string, string> = {
  openai_responses: 'OpenAI Responses',
  openai_chat_completions: 'Chat Completions',
  anthropic_messages: 'Anthropic Messages',
};

export function date(value: string): string {
  return new Date(value).toLocaleString('en-GB', { dateStyle: 'medium', timeStyle: 'short' });
}
