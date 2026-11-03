import { Link } from "react-router-dom";
import type { ReactNode } from "react";
import usePageTitle from "../scripts/usePageTitle.ts";

function MockCard({ children }: { children: ReactNode }) {
  return (
    <div className="w-full max-w-md rounded-xl border border-line-hover bg-surface p-4 shadow-xl shadow-panel/40">
      {children}
    </div>
  );
}

function MockDots() {
  return (
    <div className="flex gap-1.5">
      <span className="size-2.5 rounded-full bg-dim/40" />
      <span className="size-2.5 rounded-full bg-dim/40" />
      <span className="size-2.5 rounded-full bg-dim/40" />
    </div>
  );
}

function GitDiffMock() {
  return (
    <MockCard>
      <div className="flex items-center justify-between border-b border-line pb-3">
        <MockDots />
        <span className="font-mono text-xs text-muted">
          chapter-03/solution.ts
        </span>
      </div>
      <div className="mt-3 space-y-1 font-mono text-xs leading-relaxed">
        <p className="text-dim">
          function twoSum(nums: number[], target: number) &#123;
        </p>
        <p className="bg-red-500/10 text-red-400/90">
          - &nbsp;return bruteForce(nums, target);
        </p>
        <p className="bg-accent-light/10 text-accent-light">
          + &nbsp;const seen = new Map&lt;number, number&gt;();
        </p>
        <p className="bg-accent-light/10 text-accent-light">
          + &nbsp;for (let i = 0; i &lt; nums.length; i++) &#123;
        </p>
        <p className="pl-4 bg-accent-light/10 text-accent-light">
          + &nbsp;&nbsp;if (seen.has(target - nums[i])) return [seen.get(target
          - nums[i])!, i];
        </p>
        <p className="text-dim">&#125;</p>
      </div>
      <div className="mt-3 flex items-center gap-2 border-t border-line pt-3 font-mono text-xs text-muted">
        <span className="text-accent-light">$</span> git push origin chapter-03
      </div>
    </MockCard>
  );
}

function TestResultsMock() {
  const rows = [
    { label: "two_sum.test.ts", pass: true },
    { label: "edge_cases.test.ts", pass: true },
    { label: "large_input.test.ts", pass: true },
    { label: "duplicate_targets.test.ts", pass: false },
  ];
  return (
    <MockCard>
      <div className="flex items-center justify-between border-b border-line pb-3">
        <MockDots />
        <span className="font-mono text-xs text-muted">sandbox #4471</span>
      </div>
      <div className="mt-3 space-y-2 font-mono text-xs">
        <p className="text-muted">
          <span className="text-accent-light">$</span> run tests --chapter 03
        </p>
        {rows.map((row) => (
          <div key={row.label} className="flex items-center gap-2">
            <span
              className={row.pass ? "text-accent-light" : "text-red-400/90"}
            >
              {row.pass ? "✓" : "✗"}
            </span>
            <span className={row.pass ? "text-muted" : "text-red-400/90"}>
              {row.label}
            </span>
          </div>
        ))}
        <p className="mt-2 text-dim">3 passed, 1 failed · 480ms</p>
      </div>
    </MockCard>
  );
}

function ChapterListMock() {
  const chapters = [
    { title: "Arrays & hashing", tag: "Easy", done: true },
    { title: "Two pointers", tag: "Easy", done: true },
    { title: "Sliding window", tag: "Medium", done: true },
    { title: "Graph traversal", tag: "Hard", done: false },
  ];
  return (
    <MockCard>
      <div className="flex items-center justify-between border-b border-line pb-3">
        <MockDots />
        <span className="font-mono text-xs text-muted">
          public / interview-warmup
        </span>
      </div>
      <div className="mt-3 space-y-2">
        {chapters.map((chapter) => (
          <div
            key={chapter.title}
            className="flex items-center justify-between rounded-lg bg-panel px-3 py-2"
          >
            <div className="flex items-center gap-2">
              <span
                className={
                  chapter.done
                    ? "flex size-4 items-center justify-center rounded-full bg-accent-light/20 text-[10px] text-accent-light"
                    : "size-4 rounded-full border border-line-hover"
                }
              >
                {chapter.done ? "✓" : ""}
              </span>
              <span className="text-sm text-ink">{chapter.title}</span>
            </div>
            <span className="rounded-md bg-surface px-2 py-0.5 text-[10px] font-semibold text-muted">
              {chapter.tag}
            </span>
          </div>
        ))}
      </div>
    </MockCard>
  );
}

const highlights: {
  eyebrow: string;
  title: string;
  detail: string;
  visual: ReactNode;
}[] = [
  {
    eyebrow: "Real interview practice",
    title: "Solve it the same way you would on the job",
    detail:
      "Multi-step coding problems with starter files and hidden tests, the same shape as a real technical interview. Clone the repo, commit and push like it's a real codebase, instead of typing into a text box.",
    visual: <GitDiffMock />,
  },
  {
    eyebrow: "Sandboxed execution",
    title: "Instant feedback, zero setup",
    detail:
      "Every submission runs in its own isolated container and gets checked against a real test suite the moment you push, so you know exactly what to fix before the real thing.",
    visual: <TestResultsMock />,
  },
  {
    eyebrow: "Chapters & public library",
    title: "Structured practice, your way",
    detail:
      "Work through ordered chapters tagged by topic and difficulty, from easy warmups to hard, interview-grade problems. Attempt assessments others have published, or keep your own private for a focused mock interview.",
    visual: <ChapterListMock />,
  },
];

const steps = [
  {
    step: "01",
    title: "Pick or write an assessment",
    detail:
      "Browse the public library for practice, or author your own chapters and tests to drill the exact skills you're weak on.",
  },
  {
    step: "02",
    title: "Clone it and solve it",
    detail:
      "Get a git remote and a sandbox. Work in your own editor, on your own terms, the same way you would in a real interview.",
  },
  {
    step: "03",
    title: "Read the results",
    detail:
      "See test outcomes, commit history and timing per chapter, so you always know how close you are to interview-ready.",
  },
];

export default function Landing() {
  usePageTitle();

  return (
    <div className="mx-auto w-[75vw] py-20">
      <section className="flex flex-col items-center text-center">
        <p className="mb-4 text-xs font-semibold uppercase text-accent-light">
          Practice technical interviews with real code
        </p>
        <h1 className="max-w-3xl text-5xl font-bold leading-tight text-ink sm:text-6xl">
          Stop LARPing.{" "}
          <span className="text-accent-light">Start shipping.</span>
        </h1>
        <p className="mt-6 max-w-2xl text-lg text-muted">
          QuitLARP prepares you for technical interviews the way real work
          happens: in a git repository, inside an isolated container, judged by
          tests instead of a stranger's gut feeling. No whiteboards, no trivia,
          no pretending.
        </p>
        <div className="mt-10 flex flex-col gap-3 sm:flex-row">
          <Link
            to="/register"
            className="rounded-lg bg-accent px-6 py-3 font-semibold text-shell hover:bg-accent-light"
          >
            Start practicing for free
          </Link>
          <Link
            to="/pricing"
            className="rounded-lg border border-accent-secondary/40 bg-panel px-6 py-3 font-semibold text-accent-secondary hover:bg-accent-secondary/10"
          >
            View pricing
          </Link>
        </div>
      </section>

      <section className="mt-32 flex flex-col gap-24">
        {highlights.map((item, index) => (
          <div
            key={item.title}
            className={`flex flex-col items-center gap-10 md:gap-16 ${
              index % 2 === 1 ? "md:flex-row-reverse" : "md:flex-row"
            }`}
          >
            <div className="flex w-full justify-center md:w-1/2">
              {item.visual}
            </div>
            <div className="w-full md:w-1/2">
              <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
                {item.eyebrow}
              </p>
              <h2 className="text-3xl font-bold text-ink">{item.title}</h2>
              <p className="mt-4 text-muted">{item.detail}</p>
            </div>
          </div>
        ))}
      </section>

      <section className="mt-32">
        <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
          How it works
        </p>
        <h2 className="text-3xl font-bold text-ink">
          From blank repo to interview-ready
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3">
          {steps.map((item) => (
            <div
              key={item.step}
              className="rounded-xl border border-line-hover bg-surface p-6"
            >
              <p className="text-sm font-semibold text-accent-light">
                {item.step}
              </p>
              <h3 className="mt-2 text-lg font-semibold text-ink">
                {item.title}
              </h3>
              <p className="mt-3 text-sm text-muted">{item.detail}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-32 rounded-xl border border-accent-light/30 bg-accent-light/10 p-10 text-center">
        <h2 className="text-3xl font-bold text-ink">
          Ready to see what you can actually build?
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-muted">
          Create an account, pick your first assessment and get real feedback in
          minutes.
        </p>
        <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <Link
            to="/register"
            className="rounded-lg bg-accent px-6 py-3 font-semibold text-shell hover:bg-accent-light"
          >
            Create an account
          </Link>
          <Link
            to="/login"
            className="rounded-lg border border-accent-secondary/40 bg-panel px-6 py-3 font-semibold text-accent-secondary hover:bg-accent-secondary/10"
          >
            Log in
          </Link>
        </div>
      </section>
    </div>
  );
}
