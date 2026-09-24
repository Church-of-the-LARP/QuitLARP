import { useEffect, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import Markdown from "react-markdown";
import { Link, useParams } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";
import {
  SolveApiError,
  abandonSession,
  getAssessment,
  getCurrentSession,
  runTests,
  saveFile,
  startSession,
  submitSolution,
  type SolveAssessment,
  type SolveResult,
  type SolveSession,
  type SolveSolution,
} from "../scripts/solveApi.ts";

const languageByExtension: Record<string, string> = {
  py: "python",
  ml: "ocaml",
  js: "javascript",
  ts: "typescript",
  tsx: "typescript",
  go: "go",
  rs: "rust",
  java: "java",
  cpp: "cpp",
  c: "c",
  rb: "ruby",
  sh: "shell",
  json: "json",
  md: "markdown",
};

const languageForFile = (fileName: string) => {
  const extension = fileName.toLowerCase().split(".").pop() ?? "";
  return languageByExtension[extension] ?? "plaintext";
};

const formatRemaining = (milliseconds: number) => {
  const total = Math.max(0, Math.floor(milliseconds / 1000));
  const hours = Math.floor(total / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const seconds = total % 60;
  const pad = (value: number) => String(value).padStart(2, "0");
  return hours > 0
    ? `${hours}:${pad(minutes)}:${pad(seconds)}`
    : `${pad(minutes)}:${pad(seconds)}`;
};

const errorText = (err: unknown, fallback = "Something went wrong") => {
  if (err instanceof SolveApiError) return err.message;
  if (err instanceof Error && err.message) return err.message;
  return fallback;
};

export default function AssessmentSolveSingle() {
  const params = useParams();
  const id = Number(params.id);

  const [assessment, setAssessment] = useState<SolveAssessment | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);

  const [session, setSession] = useState<SolveSession | null>(null);
  const [sessionState, setSessionState] = useState<
    "idle" | "preparing" | "running"
  >("idle");
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const [content, setContent] = useState("");
  const [results, setResults] = useState<SolveResult[] | null>(null);
  const [raw, setRaw] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [isPublic, setIsPublic] = useState(false);
  const [solution, setSolution] = useState<SolveSolution | null>(null);
  const [tab, setTab] = useState<"desc" | "runner">("desc");
  const [now, setNow] = useState(() => Date.now());

  const [asideWidth, setAsideWidth] = useState(460);
  const containerRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);
  const startRef = useRef<Promise<SolveSession> | null>(null);

  usePageTitle(assessment ? `Solve: ${assessment.title}` : undefined);

  // StrictMode runs mount effects twice; reuse the in-flight start so only one
  // session is created per assessment.
  const startSessionOnce = () => {
    if (startRef.current === null) {
      startRef.current = startSession(id).finally(() => {
        startRef.current = null;
      });
    }
    return startRef.current;
  };

  useEffect(() => {
    let active = true;

    const applySession = (next: SolveSession) => {
      setSession(next);
      setContent(next.content ?? "");
      setSessionState("running");
    };

    const loadSession = async () => {
      setSessionState("preparing");
      setSessionError(null);
      try {
        const current = await getCurrentSession(id);
        if (active) applySession(current);
      } catch (err) {
        if (!active) return;
        if (err instanceof SolveApiError && err.status === 404) {
          try {
            const started = await startSessionOnce();
            if (active && started) applySession(started);
          } catch (startError) {
            if (!active) return;
            setSessionState("idle");
            setSessionError(errorText(startError, "Could not start a session"));
          }
          return;
        }
        setSessionState("idle");
        setSessionError(errorText(err, "Could not load the session"));
      }
    };

    const load = async () => {
      if (!Number.isInteger(id) || id <= 0) {
        setLoadError("This assessment id is not valid");
        setLoading(false);
        return;
      }

      setLoading(true);
      setLoadError(null);
      setAssessment(null);
      setSession(null);
      setSessionState("idle");
      setSessionError(null);
      setNotice(null);
      setActionError(null);
      setSolution(null);
      setResults(null);
      setRaw("");
      setTab("desc");

      try {
        const loaded = await getAssessment(id);
        if (!active) return;
        setAssessment(loaded);
        setLoading(false);
        const solvable =
          loaded.kind === "leetcode" &&
          (loaded.chapters ?? []).some((chapter) => chapter.taskFile !== "");
        if (solvable) await loadSession();
      } catch (err) {
        if (!active) return;
        setLoadError(
          err instanceof SolveApiError && err.status === 404
            ? "This assessment does not exist"
            : errorText(err, "Could not load the assessment"),
        );
        setLoading(false);
      }
    };

    load();

    return () => {
      active = false;
    };
  }, [id]);

  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), 1000);
    return () => window.clearInterval(timer);
  }, []);

  const handleDragStart = () => {
    isDragging.current = true;
    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isDragging.current || !containerRef.current) return;
      const containerLeft = containerRef.current.getBoundingClientRect().left;
      const newWidth = e.clientX - containerLeft;
      const min = 220;
      const max = containerRef.current.clientWidth * 0.7;
      setAsideWidth(Math.min(Math.max(newWidth, min), max));
    };
    const handleMouseUp = () => {
      isDragging.current = false;
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };
    window.addEventListener("mousemove", handleMouseMove);
    window.addEventListener("mouseup", handleMouseUp);
    return () => {
      window.removeEventListener("mousemove", handleMouseMove);
      window.removeEventListener("mouseup", handleMouseUp);
    };
  }, []);

  const chapters = assessment
    ? [...(assessment.chapters ?? [])].sort((a, b) => a.position - b.position)
    : [];
  const solvable =
    assessment !== null &&
    assessment.kind === "leetcode" &&
    chapters.some((chapter) => chapter.taskFile !== "");
  const activeChapter =
    session !== null
      ? (chapters.find((chapter) => chapter.id === session.chapterId) ?? null)
      : null;
  const description =
    activeChapter?.description || assessment?.description || "";

  const deadline = session !== null ? Date.parse(session.deadline) : Number.NaN;
  const hasDeadline = Number.isFinite(deadline);
  const expired = session !== null && hasDeadline && deadline <= now;

  const fileName = session !== null ? session.fileName || session.taskFile : "";
  const language = languageForFile(fileName);
  const busy = running || submitting;
  const controlsDisabled = busy || expired || solution !== null;
  const timeChipLabel = !hasDeadline
    ? "no time limit"
    : expired
      ? "session expired"
      : `⏱ ${formatRemaining(deadline - now)}`;
  const timeChipClass = !hasDeadline
    ? "border-line bg-surface text-dim"
    : expired
      ? "border-danger/40 bg-danger/15 text-danger"
      : "border-warning/40 bg-warning/15 text-warning-light";

  const handleSessionGone = (err: unknown) => {
    if (
      err instanceof SolveApiError &&
      (err.status === 404 || err.status === 410)
    ) {
      setSession(null);
      setSessionState("idle");
      setResults(null);
      setRaw("");
      setNotice(
        "This session is no longer available. Start a new one to keep practicing.",
      );
      return true;
    }
    return false;
  };

  const handleRunTests = async () => {
    if (session === null || expired) return;
    setRunning(true);
    setActionError(null);
    setTab("runner");
    try {
      const saved = await saveFile(id, content);
      if (saved) setSession(saved);
      const run = await runTests(id);
      setResults(run.results);
      setRaw(run.raw);
    } catch (err) {
      if (!handleSessionGone(err)) {
        setActionError(errorText(err, "Could not run the tests"));
      }
    } finally {
      setRunning(false);
    }
  };

  const handleSubmit = async () => {
    if (session === null || expired || submitting) return;
    const message = isPublic
      ? "Submit this solution and make it public? Other people will be able to see it."
      : "Submit this solution?";
    if (!window.confirm(message)) return;
    setSubmitting(true);
    setActionError(null);
    try {
      const saved = await saveFile(id, content);
      if (saved) setSession(saved);
      const created = await submitSolution(id, isPublic);
      if (created) setSolution(created);
    } catch (err) {
      if (!handleSessionGone(err)) {
        setActionError(errorText(err, "Could not submit the solution"));
      }
    } finally {
      setSubmitting(false);
    }
  };

  const handleLeave = async () => {
    if (session === null) return;
    if (
      !window.confirm(
        "Leave this session? Unsubmitted work in this session will be discarded.",
      )
    ) {
      return;
    }
    setActionError(null);
    try {
      await abandonSession(id);
      setSession(null);
      setSessionState("idle");
      setContent("");
      setResults(null);
      setRaw("");
      setTab("desc");
      setIsPublic(false);
      setNotice("You left the session. Start a new one whenever you are ready.");
    } catch (err) {
      if (!handleSessionGone(err)) {
        setActionError(errorText(err, "Could not leave the session"));
      }
    }
  };

  const handleStart = async () => {
    if (loading || assessment === null || !solvable) return;
    setSessionError(null);
    setActionError(null);
    setNotice(null);
    setSolution(null);
    setSessionState("preparing");
    try {
      const started = await startSessionOnce();
      if (started) {
        setSession(started);
        setContent(started.content ?? "");
        setResults(null);
        setRaw("");
        setTab("desc");
        setSessionState("running");
      }
    } catch (err) {
      setSessionState(session === null ? "idle" : "running");
      const message =
        err instanceof SolveApiError && err.status === 409
          ? "A session is already running here. Leave it first, then start a new one."
          : errorText(err, "Could not start a session");
      setSessionError(message);
      // The session view stays on screen when a session is still active, so the
      // error has to land in the banner it actually renders.
      if (session !== null) setActionError(message);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl justify-between items-center">
          <p className="text-sm text-muted">Loading...</p>
        </div>
      </div>
    );
  }

  if (loadError) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl justify-between items-center">
          <p className="text-sm text-danger">{loadError}</p>
        </div>
      </div>
    );
  }

  if (!assessment) return null;

  if (!solvable) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl flex-col gap-3">
          <h1 className="text-2xl font-bold text-ink">{assessment.title}</h1>
          <p className="text-sm text-muted">
            This assessment is not solvable yet. It has no runnable task file, so
            there is nothing to practice here.
          </p>
          <Link
            to={`/assessments/${id}/preview`}
            className="w-fit text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
          >
            Back to the assessment preview
          </Link>
        </div>
      </div>
    );
  }

  if (sessionState === "preparing") {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl flex-col gap-2">
          <p className="text-sm text-ink">Preparing your environment...</p>
          <p className="text-xs text-muted">
            The first run can take a few minutes while the assessment image is
            built. Keep this tab open.
          </p>
        </div>
      </div>
    );
  }

  if (session === null) {
    return (
      <div className="flex flex-col items-center flex-1 py-20 px-4 sm:px-6 lg:px-8 w-screen">
        <div className="flex w-full max-w-4xl flex-col gap-4">
          <h1 className="text-2xl font-bold text-ink">{assessment.title}</h1>
          {notice && <p className="text-sm text-muted">{notice}</p>}
          {sessionError && (
            <p
              role="alert"
              className="rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger"
            >
              {sessionError}
            </p>
          )}
          <button
            type="button"
            onClick={handleStart}
            className="bg-accent hover:bg-accent-light text-shell font-semibold px-4 py-2 rounded-lg w-fit shadow-md shadow-accent/20 transition"
          >
            Start a practice session
          </button>
          <Link
            to={`/assessments/${id}/preview`}
            className="w-fit text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
          >
            Back to the assessment preview
          </Link>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto w-[95vw] py-10 overflow-hidden">
      <div
        ref={containerRef}
        className="flex flex-col w-full bg-panel rounded-xl p-6"
      >
        <div className="flex flex-row items-center justify-between gap-4 mb-4">
          <h1 className="flex items-center gap-3 text-2xl font-bold text-ink">
            <span className="h-6 w-1.5 rounded-full bg-accent" />
            {assessment.title}
          </h1>
          <div className="flex flex-row items-center gap-3">
            <label className="flex items-center gap-2 text-xs text-muted">
              <input
                type="checkbox"
                checked={isPublic}
                onChange={(event) => setIsPublic(event.target.checked)}
                disabled={controlsDisabled}
                className="h-3.5 w-3.5 accent-accent"
              />
              make my solution public
            </label>
            <button
              type="button"
              onClick={handleRunTests}
              disabled={controlsDisabled}
              className="bg-accent hover:bg-accent-light text-shell font-semibold px-4 py-2 rounded-lg w-fit shadow-md shadow-accent/20 transition disabled:cursor-not-allowed disabled:opacity-50"
            >
              {running ? "Running..." : "Run tests"}
            </button>
            <button
              type="button"
              onClick={handleSubmit}
              disabled={controlsDisabled}
              className="bg-accent-secondary hover:bg-accent-secondary-hover text-shell text-sm font-semibold px-6 py-3 rounded-lg transition disabled:cursor-not-allowed disabled:opacity-50"
            >
              {submitting ? "Submitting..." : "Submit"}
            </button>
            <button
              type="button"
              onClick={handleLeave}
              className="rounded-lg border border-line px-4 py-2 text-sm font-semibold text-muted transition hover:text-accent-lighter"
            >
              Leave session
            </button>
          </div>
        </div>

        {notice && <p className="mb-3 text-sm text-muted">{notice}</p>}

        {actionError && (
          <p
            role="alert"
            className="mb-3 rounded-lg border border-danger/40 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            {actionError}
          </p>
        )}

        {expired && (
          <div className="mb-3 flex flex-wrap items-center gap-3 rounded-lg border border-warning/40 bg-warning/10 px-3 py-2 text-sm text-warning-light">
            <span>
              The session timer has run out. Running tests and submitting are
              disabled.
            </span>
            <button
              type="button"
              onClick={handleStart}
              className="font-semibold text-accent-lighter transition hover:text-accent-light"
            >
              Start a new session
            </button>
          </div>
        )}

        <div className="mt-4 flex flex-row gap-2 w-full min-h-[60vh]">
          <aside
            style={{ width: asideWidth }}
            className="flex flex-col gap-3 shrink-0 overflow-auto pr-4"
          >
            <p className="text-xs font-semibold uppercase text-accent-light">
              {session.chapterTitle || `chapter ${session.chapterId}`}
            </p>

            <div className="flex gap-4 border-b border-line pb-2 text-sm">
              <button
                type="button"
                className={`border-b-2 pb-2 -mb-2.5 transition ${
                  tab === "desc"
                    ? "border-accent font-semibold text-accent-lighter"
                    : "border-transparent text-muted hover:text-accent-lighter"
                }`}
                onClick={() => setTab("desc")}
              >
                Chapter desc
              </button>
              <button
                type="button"
                className={`border-b-2 pb-2 -mb-2.5 transition ${
                  tab === "runner"
                    ? "border-accent font-semibold text-accent-lighter"
                    : "border-transparent text-muted hover:text-accent-lighter"
                }`}
                onClick={() => setTab("runner")}
              >
                Runner
              </button>
            </div>

            {tab === "desc" ? (
              <div className="text-ink **:[all:revert]">
                <Markdown>{description}</Markdown>
              </div>
            ) : (
              <div className="flex flex-col gap-3 text-sm">
                {results === null ? (
                  <p className="text-muted">
                    Run the tests to see the results here.
                  </p>
                ) : results.length === 0 ? (
                  <p className="text-muted">The runner did not report a test.</p>
                ) : (
                  <ul className="flex flex-col gap-1.5">
                    {results.map((result, index) => (
                      <li
                        key={`${result.name}-${index}`}
                        className={`flex flex-col gap-1 rounded-md border-l-2 px-2.5 py-1.5 ${
                          result.passed
                            ? "border-success bg-success/10"
                            : "border-danger bg-danger/10"
                        }`}
                      >
                        <span className="flex items-center gap-2 text-ink">
                          <span
                            className={
                              result.passed ? "text-success" : "text-danger"
                            }
                          >
                            {result.passed ? "✓" : "✗"}
                          </span>
                          {result.name}
                        </span>
                        {!result.passed && result.detail && (
                          <pre className="max-h-40 overflow-auto whitespace-pre-wrap font-mono text-xs text-danger">
                            {result.detail}
                          </pre>
                        )}
                      </li>
                    ))}
                  </ul>
                )}

                {raw && (
                  <pre className="max-h-48 overflow-auto whitespace-pre-wrap rounded-lg border border-line-hover bg-surface p-3 font-mono text-xs text-muted">
                    {raw}
                  </pre>
                )}
              </div>
            )}
          </aside>

          <div
            onMouseDown={handleDragStart}
            className="w-1.5 shrink-0 cursor-col-resize hover:bg-accent-lighter/50 transition mr-3 rounded border-r border-line"
          />

          <main className="flex min-w-0 flex-1 flex-col gap-3">
            <div className="flex flex-row items-center justify-between w-full gap-4 text-sm text-muted">
              <span className="inline-flex items-center gap-1.5 rounded-full border border-accent-secondary/40 bg-accent-secondary/15 px-3 py-1 font-mono text-xs text-accent-secondary-hover w-fit">
                <span className="h-1.5 w-1.5 rounded-full bg-accent-secondary" />
                {language}
              </span>
              <span className="rounded-full border border-line bg-surface px-3 py-1 font-mono text-xs text-dim">
                {fileName}
              </span>
              <span
                className={`rounded-full border px-3 py-1 font-mono text-xs w-fit ${timeChipClass}`}
              >
                {timeChipLabel}
              </span>
            </div>

            {solution ? (
              <div className="flex flex-col gap-3 rounded-lg border border-success/40 bg-success/10 p-6">
                <p className="text-sm font-semibold text-success">
                  Solution submitted
                </p>
                <p className="text-sm text-muted">
                  Your {solution.public ? "public" : "private"} solution was
                  committed as{" "}
                  <span className="font-mono text-xs text-ink">
                    {solution.commitSha}
                  </span>
                </p>
                <a
                  href={solution.url}
                  target="_blank"
                  rel="noreferrer"
                  className="w-fit text-sm font-semibold text-accent-light transition hover:text-accent-lighter"
                >
                  {solution.url}
                </a>
                <p className="text-xs text-dim">
                  Submitted{" "}
                  {new Date(solution.createdAt).toLocaleString()}. Leave the
                  session to start a new one.
                </p>
              </div>
            ) : (
              <Editor
                height="70vh"
                language={language}
                value={content}
                onChange={(value) => setContent(value ?? "")}
                theme="vs-dark"
                options={{ automaticLayout: true, readOnly: expired }}
                className="rounded-lg overflow-hidden"
              />
            )}
          </main>
        </div>
      </div>
    </div>
  );
}
