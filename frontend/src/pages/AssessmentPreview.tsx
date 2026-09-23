import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import client, { getErrorText } from "../scripts/api";
import useAuth from "../scripts/useAuth.tsx";
import type { components } from "../api/schema.ts";

type Assessment = components["schemas"]["Assessment"];
type Chapter = components["schemas"]["Chapter"];

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
  const params = useParams();
  const id = Number(params.id);
  const { user } = useAuth();

  const [assessment, setAssessment] = useState<Assessment | null>(null);
  const [activeChapter, setActiveChapter] = useState<Chapter | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!Number.isInteger(id) || id <= 0) {
      setError("This assessment id is not valid");
      setLoading(false);
      return;
    }
    let active = true;
    setLoading(true);
    setError(null);
    setAssessment(null);
    setActiveChapter(null);
    client
      .GET("/api/v1/assessments/{id}", { params: { path: { id } } })
      .then((res) => {
        if (!active) return;
        if (res.error) {
          setError(
            res.response.status === 404
              ? "This assessment does not exist"
              : getErrorText(res.error, "Could not load the assessment"),
          );
          return;
        }
        const loaded = res.data?.assessment;
        if (!loaded) {
          setError("Could not load the assessment");
          return;
        }
        setAssessment(loaded);
      })
      .catch(() => {
        if (active) setError("Could not load the assessment");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [id]);

  if (loading) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl justify-between items-center">
          <p className="text-sm text-muted">Loading...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl justify-between items-center">
          <p className="text-sm text-danger">{error}</p>
        </div>
      </div>
    );
  }

  if (!assessment) return null;

  const chapters = [...(assessment.chapters ?? [])].sort(
    (a, b) => a.position - b.position,
  );
  const activeLabel = activeChapter
    ? `chapters/${activeChapter.position}_${activeChapter.title}`
    : "README.md";
  const activeContent = activeChapter
    ? activeChapter.description
    : assessment.description;
  const activeTimeLimit = activeChapter
    ? activeChapter.timeLimitMinutes
    : assessment.timeLimitMinutes;
  const isAdmin = user?.role === "admin" || user?.role === "superadmin";
  const isAuthor = user !== null && assessment.author?.id === user.id;
  const showRepositoryLink = isAdmin || isAuthor;

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
        </div>
        <div className="flex flex-col items-end gap-2">
          <div className="flex flex-row items-center gap-4">
            <button className="bg-accent-secondary hover:bg-accent-secondary-hover text-shell text-sm font-semibold px-6 py-3 rounded-lg">
              leaderboard & solutions
            </button>
            <button className="bg-accent-secondary hover:bg-accent-secondary-hover text-shell text-sm font-semibold px-6 py-3 rounded-lg">
              begin practice run
            </button>
          </div>
          {showRepositoryLink && (
            <Link
              to={`/assessments/${id}/git`}
              className="text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
            >
              View repository
            </Link>
          )}
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
                  {`chapters/${chapter.position}_${chapter.title}`}
                  {activeChapter?.id === chapter.id && (
                    <span className="block font-normal text-xs text-dim">
                      {chapter.startMode === "clean"
                        ? "starts clean"
                        : "continues the previous chapter"}
                    </span>
                  )}
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
