import type { components } from '../api.generated';

export type Connection = components['schemas']['Connection'];
export type ConnectionConfig = components['schemas']['ConnectionConfig'];
export type ModelConfig = components['schemas']['Model'];
export type AvailableModel = components['schemas']['AvailableModel'];
export type Product = components['schemas']['Product'];
export type ProductInput = components['schemas']['ProductInput'];
export type Run = components['schemas']['Run'];
export type Revision = components['schemas']['Revision'];
export type RevisionInfo = components['schemas']['RevisionInfo'];
export type Review = components['schemas']['Review'];
export type ReviewActivity = components['schemas']['ReviewActivity'];
export type ProductModel = components['schemas']['ProductModel'];
export type ProductObject = components['schemas']['ProductObject'];
export type Section = components['schemas']['Section'];
export type Document = NonNullable<Revision['content']['views']['document']>;
export type Chapter = NonNullable<Document['chapters']>[number];
export type DocumentBlock = NonNullable<Chapter['blocks']>[number];
export type Diagram = components['schemas']['Diagram'];
export type Claim = components['schemas']['Claim'];
export type Scenario = components['schemas']['Scenario'];
export type EvidenceView = components['schemas']['EvidenceView'];
export type Parameter = components['schemas']['Parameter'];
export type ProbeResult = components['schemas']['ProbeResult'];
export type FactoryStatus = components['schemas']['FactoryStatus'];
export type LearningPack = components['schemas']['LearningPack'];
export type SourceInput = components['schemas']['SourceInput'];
export type GitHubConnection = components['schemas']['GitHubConnection'];
export type OAuthURL = components['schemas']['OAuthURL'];
export type User = components['schemas']['User'];
export type Me = components['schemas']['Me'];
export type MCPSettings = components['schemas']['MCPSettings'];
export type MCPClient = components['schemas']['MCPClient'];
export type Member = components['schemas']['Member'];

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
  if (response.status === 401 && !path.startsWith('/auth/')) {
    // The session expired or was revoked; start again from sign-in.
    globalThis.location.assign('/login');
  }
  if (!response.ok) {
    const body = (await response.json().catch(() => ({}))) as ErrorBody;
    throw new ApiError(
      body.message ?? 'The request could not be completed.',
      response.status,
      body.request_id
    );
  }
  if (response.status === 204) {
    return undefined as T;
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
