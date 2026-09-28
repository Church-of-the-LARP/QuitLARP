import type { components } from '../api/schema.ts';

export type User = components['schemas']['User'];
export type Role = User['role'];
export type AuthStatus = 'loading' | 'anonymous' | 'authenticated';

export type AssessmentSummary = components['schemas']['AssessmentSummary'];
export type Difficulty = AssessmentSummary['difficulty'];

export type UserStats = components['schemas']['UserStatsOutputBody'];
export type ContinuingSession =
  components['schemas']['UserSessionsOutputBody']['sessions'][number];
export type Notification =
  components['schemas']['UserNotificationsOutputBody']['notifications'][number];

export interface ActionResult {
  error?: string;
  notice?: string;
}

export interface AlertMessage {
  kind: 'error' | 'notice';
  text: string;
}
