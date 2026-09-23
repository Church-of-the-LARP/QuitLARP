import { useEffect, useRef, useState } from "react";
import Editor from "@monaco-editor/react";
import Markdown from "react-markdown";
import usePageTitle from "../scripts/usePageTitle.ts";

type TestResult = {
  name: string;
  passed: boolean;
};

type AssessmentPreviewData = {
  title: string;
  language: string;
  timeLimitMinutes: number | null;
  fileName: string;
  description: string;
  testResults: TestResult[];
};

const mockAssessment: AssessmentPreviewData = {
  title: "Two Sum",
  language: "C++",
  timeLimitMinutes: null,
  fileName: "App.js",
  description: `## Input
An array of integers nums and an integer target.

## Output
The indices of the two numbers that add up to the target.

## Rules
Each input has exactly one solution. You cannot use the same array element twice. You can return the indices in any order.

## Example
Input: nums = [2, 7, 11, 15], target = 9
Output: [0, 1]
Explanation: nums[0] + nums[1] equals 2 + 7 = 9, so we return indices 0 and 1.`,
  testResults: [
    { name: "testing output 1st func", passed: true },
    { name: "speedtest cache func", passed: true },
  ],
};

export default function AssessmentSolveSingle() {
  const assessment = mockAssessment;
  usePageTitle(`Solve: ${assessment.title}`);
  const [btn, setBtn] = useState("Chapter desc");
  const [asideWidth, setAsideWidth] = useState(460);
  const containerRef = useRef<HTMLDivElement>(null);
  const isDragging = useRef(false);

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

  return (
    <div className="mx-auto w-[95vw] py-10 overflow-hidden">
      <div
        ref={containerRef}
        className="flex flex-col w-full bg-panel rounded-xl p-6"
      >
        <div className="flex flex-row items-center justify-between mb-4">
          <h1 className="flex items-center gap-3 text-2xl font-bold text-ink mb-4">
            <span className="h-6 w-1.5 rounded-full bg-accent" />
            {assessment.title}
          </h1>
          <button className="bg-accent hover:bg-accent-light text-shell font-semibold px-4 py-2 rounded-lg w-fit shadow-md shadow-accent/20 transition">
            Run tests
          </button>
        </div>
        <div className="mt-4 flex flex-row gap-2 w-full min-h-[60vh]">
          <aside
            style={{ width: asideWidth }}
            className="flex flex-col gap-3 shrink-0 overflow-auto pr-4"
          >
            <div className="flex gap-4 border-b border-line pb-2 text-sm">
              <button
                type="button"
                className={`border-b-2 pb-2 -mb-2.5 transition ${
                  btn === "Chapter desc"
                    ? "border-accent font-semibold text-accent-lighter"
                    : "border-transparent text-muted hover:text-accent-lighter"
                }`}
                onClick={() => setBtn("Chapter desc")}
              >
                Chapter desc
              </button>
              <button
                type="button"
                className={`border-b-2 pb-2 -mb-2.5 transition ${
                  btn === "Runner"
                    ? "border-accent font-semibold text-accent-lighter"
                    : "border-transparent text-muted hover:text-accent-lighter"
                }`}
                onClick={() => setBtn("Runner")}
              >
                Runner
              </button>
            </div>
            {btn === "Chapter desc" ? (
              <div className="text-ink **:[all:revert]">
                <Markdown>{assessment.description}</Markdown>
              </div>
            ) : (
              assessment.testResults.length > 0 && (
                <ul className="flex flex-col gap-1.5 text-sm">
                  {assessment.testResults.map((result) => (
                    <li
                      key={result.name}
                      className={`flex items-center gap-2 rounded-md border-l-2 px-2.5 py-1.5 ${
                        result.passed
                          ? "border-success bg-success/10 text-ink"
                          : "border-danger bg-danger/10 text-ink"
                      }`}
                    >
                      <span
                        className={
                          result.passed ? "text-success" : "text-danger"
                        }
                      >
                        {result.passed ? "✓" : "✗"}
                      </span>
                      {result.name}
                    </li>
                  ))}
                </ul>
              )
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
                {assessment.language}
              </span>
              <span className="rounded-full border border-line bg-surface px-3 py-1 font-mono text-xs text-dim">
                {assessment.fileName}
              </span>
              <span
                className={`rounded-full border px-3 py-1 font-mono text-xs w-fit ${
                  assessment.timeLimitMinutes !== null
                    ? "border-warning/40 bg-warning/15 text-warning-light"
                    : "border-line bg-surface text-dim"
                }`}
              >
                {assessment.timeLimitMinutes !== null
                  ? `⏱ ${assessment.timeLimitMinutes} min`
                  : "no time limit"}
              </span>
            </div>

            <Editor
              height="70vh"
              defaultLanguage={assessment.language.toLowerCase()}
              defaultValue="// some comment"
              theme="vs-dark"
              options={{ automaticLayout: true }}
              className="rounded-lg overflow-hidden"
            />
          </main>
        </div>
      </div>
    </div>
  );
}
