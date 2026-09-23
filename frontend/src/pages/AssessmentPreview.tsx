import { useState } from "react";
import type { components } from "../api/schema.ts";
import usePageTitle from "../scripts/usePageTitle.ts";

type Assessment = components["schemas"]["Assessment"];
type Chapter = components["schemas"]["Chapter"];

type AssessmentPreviewData = Assessment & {
  visibility: "public" | "private";
};

const mockAssessment: AssessmentPreviewData = {
  id: 1,
  author: { id: 7, username: "ada_instructor" },
  chapters: [
    {
      id: 101,
      assessmentId: 1,
      title: "Two Sum",
      description: `Input:
An array of integers nums and an integer target.

Output:
The indices of the two numbers that add up to the target.

Rules:
Each input has exactly one solution. You cannot use the same array element twice. You can return the indices in any order.

Example
Input: nums = [2, 7, 11, 15], target = 9
Output: [0, 1]
Explanation: nums[0] + nums[1] equals 2 + 7 = 9, so we return indices 0 and 1.`,
      position: 1,
      timeLimitMinutes: 30,
      createdAt: "2026-08-01T09:00:00Z",
      updatedAt: "2026-08-14T15:32:00Z",
    },
  ],
  createdAt: "2026-08-01T09:00:00Z",
  description:
    "A single-chapter warm-up assessment covering array traversal and hash-map lookups.",
  difficulty: "easy",
  tags: [
    { id: 1, name: "arrays" },
    { id: 2, name: "hash-map" },
  ],
  template: {
    fileName: "solution.py",
    content: "def two_sum(nums, target):\n    pass\n",
  },
  tests: [
    {
      id: 501,
      name: "Two Sum",
      description:
        "Validates returned indices across small and large input arrays.",
    },
  ],
  timeLimitMinutes: 30,
  title: "Two Sum",
  updatedAt: "2026-08-14T15:32:00Z",
  visibility: "public",
};

const difficultyStyles: Record<Assessment["difficulty"], string> = {
  easy: "text-success",
  medium: "text-warning",
  hard: "text-danger",
};

const formatDate = (value: string) =>
  new Date(value).toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });

export default function AssessmentPreview() {
  const assessment = mockAssessment;
  usePageTitle(`Preview: ${assessment.title}`);
  const [activeChapter, setActiveChapter] = useState<Chapter | null>(null);

  const chapters = [...(assessment.chapters ?? [])].sort(
    (a, b) => a.position - b.position,
  );
  const activeLabel = activeChapter ? activeChapter.title : "README.md";
  const activeContent = activeChapter
    ? activeChapter.description
    : assessment.description;
  const activeTimeLimit = activeChapter
    ? activeChapter.timeLimitMinutes
    : assessment.timeLimitMinutes;

  return (
    <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
      <div className="flex w-full max-w-4xl justify-between items-center">
        <div className="flex flex-col">
          <div className="flex">
            <h1 className="text-2xl font-bold text-ink">
              {assessment.title} : &nbsp;
            </h1>
            <h1
              className={`text-2xl font-bold capitalize ${difficultyStyles[assessment.difficulty]}`}
            >
              {assessment.difficulty}
            </h1>
          </div>

          <p className="text-sm uppercase tracking-wide text-muted">
            {assessment.visibility}
          </p>
        </div>
        <div className="flex flex-row items-center gap-4">
          <button className="bg-accent-secondary hover:bg-accent-secondary-hover text-shell text-sm font-semibold px-6 py-3 rounded-lg">
            leaderboard & solutions
          </button>
          <button className="bg-accent-secondary hover:bg-accent-secondary-hover text-shell text-sm font-semibold px-6 py-3 rounded-lg">
            begin practice run
          </button>
        </div>
      </div>
      <div className="mt-8 flex w-full max-w-4xl gap-6 bg-panel rounded-xl p-6">
        <aside className="flex flex-col gap-3 w-1/3 border-r border-line pr-4">
          <p className="text-xs font-semibold uppercase text-accent-light">
            Files
          </p>
          <ul className="flex flex-col gap-1">
            <li>
              <button
                className={`w-full rounded-md px-2 py-1 text-left font-mono text-sm transition hover:bg-raised/40 ${
                  activeChapter === null
                    ? "font-semibold text-ink"
                    : "text-muted hover:text-accent-lighter"
                }`}
                type="button"
                onClick={() => setActiveChapter(null)}
              >
                README.md
              </button>
            </li>
            {chapters.map((chapter) => (
              <li key={chapter.id}>
                <button
                  className={`w-full rounded-md px-2 py-1 text-left font-mono text-sm transition hover:bg-raised/40 ${
                    activeChapter?.id === chapter.id
                      ? "font-semibold text-ink"
                      : "text-muted hover:text-accent-lighter"
                  }`}
                  type="button"
                  onClick={() => setActiveChapter(chapter)}
                >
                  {chapter.position}. {chapter.title}
                </button>
              </li>
            ))}
          </ul>

          <div className="flex flex-col gap-1 text-sm text-muted">
            <p>
              Author:{" "}
              <span className="text-ink">
                {assessment.author?.username ?? "unknown"}
              </span>
            </p>
            <p>
              Time limit:{" "}
              <span className="text-ink">
                {assessment.timeLimitMinutes} min
              </span>
            </p>
            <p>
              Chapters: <span className="text-ink">{chapters.length}</span>
            </p>
            <p>
              Updated:{" "}
              <span className="text-ink">
                {formatDate(assessment.updatedAt)}
              </span>
            </p>
            {assessment.tags && assessment.tags.length > 0 && (
              <p>
                Tags:{" "}
                <span className="text-ink">
                  {assessment.tags.map((tag) => tag.name).join(", ")}
                </span>
              </p>
            )}
          </div>
        </aside>

        <main className="flex w-2/3 flex-col gap-3">
          <p className="font-mono text-xs text-dim">
            {activeLabel} ({activeTimeLimit} min)
          </p>
          <pre className="max-h-[60vh] overflow-auto whitespace-pre-wrap rounded-lg border border-line-hover bg-surface p-4 font-mono text-sm leading-relaxed text-muted">
            {activeContent}
          </pre>
        </main>
      </div>
    </div>
  );
}
