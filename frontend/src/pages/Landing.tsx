import { Link } from "react-router-dom";

const features = [
  {
    title: "Sandboxed execution",
    detail:
      "Every submission runs in its own isolated container, so candidates can build and run real projects without touching your infrastructure.",
  },
  {
    title: "Git-native workflow",
    detail:
      "Candidates clone, commit and push like they would on the job. You review a real history, not a pasted text box.",
  },
  {
    title: "Automatic judging",
    detail:
      "Attach test suites to each chapter and get pass/fail results the moment a submission lands.",
  },
  {
    title: "Chapters and tags",
    detail:
      "Split an assessment into ordered chapters, tag it by topic and difficulty, and reuse it across hiring rounds.",
  },
  {
    title: "Public and private tests",
    detail:
      "Publish assessments for anyone to attempt, or keep them private and invite only the people you're evaluating.",
  },
  {
    title: "Roles and teams",
    detail:
      "Admins manage users and assessments; authors create; candidates take. Nobody sees more than they need to.",
  },
];

const steps = [
  {
    step: "01",
    title: "Author an assessment",
    detail:
      "Write chapters, add starter files and define the tests that decide whether a solution passes.",
  },
  {
    step: "02",
    title: "Send the link",
    detail:
      "Candidates get a git remote and a sandbox. They work in their own editor, on their own terms.",
  },
  {
    step: "03",
    title: "Read the results",
    detail:
      "See test outcomes, commit history and timing per chapter. Decide based on what they built, not what they said.",
  },
];

export default function Landing() {
  return (
    <div className="mx-auto w-[75vw] py-20">
      <section className="flex flex-col items-center text-center">
        <p className="mb-4 text-xs font-semibold uppercase text-teal-400">
          Coding assessments that run real code
        </p>
        <h1 className="max-w-3xl text-5xl font-bold leading-tight text-zinc-100 sm:text-6xl">
          Stop LARPing. <span className="text-teal-400">Start shipping.</span>
        </h1>
        <p className="mt-6 max-w-2xl text-lg text-zinc-400">
          QuitLARP runs coding assessments the way real work happens: in a git
          repository, inside an isolated container, judged by tests you wrote.
          No whiteboards, no trivia, no pretending.
        </p>
        <div className="mt-10 flex flex-col gap-3 sm:flex-row">
          <Link
            to="/register"
            className="rounded-lg bg-teal-500 px-6 py-3 font-semibold text-zinc-950 hover:bg-teal-400"
          >
            Get started for free
          </Link>
          <Link
            to="/pricing"
            className="rounded-lg border border-zinc-700 bg-zinc-900 px-6 py-3 font-semibold text-zinc-300 hover:bg-zinc-800"
          >
            View pricing
          </Link>
        </div>
      </section>

      <section className="mt-32">
        <p className="mb-2 text-xs font-semibold uppercase text-teal-400">
          What you get
        </p>
        <h2 className="text-3xl font-bold text-zinc-100">
          Everything an assessment needs, nothing it doesn't
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {features.map((feature) => (
            <div
              key={feature.title}
              className="rounded-xl border border-zinc-700 bg-zinc-800 p-6"
            >
              <h3 className="text-lg font-semibold text-zinc-100">
                {feature.title}
              </h3>
              <p className="mt-3 text-sm text-zinc-400">{feature.detail}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-32">
        <p className="mb-2 text-xs font-semibold uppercase text-teal-400">
          How it works
        </p>
        <h2 className="text-3xl font-bold text-zinc-100">
          From blank repo to hiring decision
        </h2>
        <div className="mt-10 grid grid-cols-1 gap-6 sm:grid-cols-3">
          {steps.map((item) => (
            <div
              key={item.step}
              className="rounded-xl border border-zinc-700 bg-zinc-800 p-6"
            >
              <p className="text-sm font-semibold text-teal-400">{item.step}</p>
              <h3 className="mt-2 text-lg font-semibold text-zinc-100">
                {item.title}
              </h3>
              <p className="mt-3 text-sm text-zinc-400">{item.detail}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="mt-32 rounded-xl border border-teal-400/30 bg-teal-400/10 p-10 text-center">
        <h2 className="text-3xl font-bold text-zinc-100">
          Ready to see what candidates can actually build?
        </h2>
        <p className="mx-auto mt-4 max-w-xl text-zinc-400">
          Create an account, write your first assessment and send it out in
          under an hour.
        </p>
        <div className="mt-8 flex flex-col justify-center gap-3 sm:flex-row">
          <Link
            to="/register"
            className="rounded-lg bg-teal-500 px-6 py-3 font-semibold text-zinc-950 hover:bg-teal-400"
          >
            Create an account
          </Link>
          <Link
            to="/login"
            className="rounded-lg border border-zinc-700 bg-zinc-900 px-6 py-3 font-semibold text-zinc-300 hover:bg-zinc-800"
          >
            Log in
          </Link>
        </div>
      </section>
    </div>
  );
}
