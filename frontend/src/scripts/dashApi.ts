// Dashboard transport. Uses the generated OpenAPI client and schema types.
import type { components } from "../api/schema.ts";
import client, { getErrorText } from "./api.ts";

export type AssessmentSummary = components["schemas"]["AssessmentSummary"];
export type Difficulty = AssessmentSummary["difficulty"];

export type ListAssessmentsInput = {
  difficulty?: Difficulty;
  tag?: string;
  q?: string;
  author?: string;
  offset?: number;
  limit?: number;
};

export type ListAssessmentsOutput = {
  assessments: AssessmentSummary[];
  total: number;
};

export class DashApiError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "DashApiError";
  }
}

export async function getAssessments(
  input: ListAssessmentsInput = {},
): Promise<ListAssessmentsOutput> {
  const { data, error } = await client.GET("/api/v1/assessments", {
    params: { query: input },
  });

  if (error || !data) throw new DashApiError(getErrorText(error));

  return { assessments: data.assessments ?? [], total: data.total };
}

export type ScheduledAssessment = components["schemas"]["ScheduledAssessment"];

export async function getScheduled(limit?: number): Promise<ScheduledAssessment[]> {
  const { data, error } = await client.GET("/api/v1/scheduled-assessments", {
    params: { query: { limit } },
  });

  if (error || !data) throw new DashApiError(getErrorText(error));

  return data.scheduled ?? [];
}

export type TagCount = components["schemas"]["TagCount"];

export async function getTags(): Promise<TagCount[]> {
  const { data, error } = await client.GET("/api/v1/tags");

  if (error || !data) throw new DashApiError(getErrorText(error));

  return data.tags ?? [];
}
