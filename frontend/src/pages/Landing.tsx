import { Link } from "react-router-dom";
import usePageTitle from "../scripts/usePageTitle.ts";

const features = [
  {
    title: "Real interview practice",
    detail:
      "Work through multi-step coding problems with starter files and hidden tests, the same shape as a real technical interview, not trivia questions.",
  },
  {
    title: "Git-native workflow",
    detail:
      "Clone, commit and push like you would on the job. Build the exact muscle memory interviewers expect, instead of typing into a text box.",
  },
  {
    title: "Sandboxed execution",
    detail:
      "Every submission runs in its own isolated container, so you can build and run real projects without setting up anything locally.",
  },
  {
    title: "Instant, automatic feedback",
    detail:
      "Attach test suites to each chapter and get pass/fail results the moment you submit, so you know exactly what to fix before the real thing.",
  },
  {
    title: "Chapters and difficulty tags",
    detail:
      "Work through ordered chapters tagged by topic and difficulty, from easy warmups to hard, interview-grade problems.",
  },
  {
    title: "Public practice library",
    detail:
      "Attempt assessments other people have published, or keep your own private for a focused, distraction-free mock interview.",
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
          Stop LARPing. <span className="text-accent-light">Start shipping.</span>
        </h1>
        <p className="mt-6 max-w-2xl text-lg text-muted">
          QuitLARP prepares you for technical interviews the way real work
          happens: in a git repository, inside an isolated container, judged
          by tests instead of a stranger's gut feeling. No whiteboards, no
          trivia, no pretending.
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

      <section className="mt-32">
        <p className="mb-2 text-xs font-semibold uppercase text-accent-light">
          What you get
        </p>
        <h2 className="text-3xl font-bold text-ink">
          Everything interview prep needs, nothing it doesn't
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="rounded-xl border border-line-hover bg-surface p-6"
            >
              <h3 className="text-lg font-semibold text-ink">
                {feature.title}
              </h3>
              <p className="mt-3 text-sm text-muted">{feature.detail}</p>
            </div>
          ))}
        </div>
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
              <p className="text-sm font-semibold text-accent-light">{item.step}</p>
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
          Create an account, pick your first assessment and get real feedback
          in minutes.
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
