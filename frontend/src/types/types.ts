import type { components } from '../api/schema.ts';

export type User = components['schemas']['User'];
export type Role = 'user' | 'moderator' | 'admin';
export type AuthStatus = 'loading' | 'anonymous' | 'authenticated';

export interface ActionResult {
  error?: string;
  notice?: string;
}

export interface AlertMessage {
  kind: 'error' | 'notice';
  text: string;
}
