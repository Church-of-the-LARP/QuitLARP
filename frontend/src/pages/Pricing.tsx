import DesertScene from "../components/DesertScene";

export default function Pricing() {
  return (
    <div className="relative flex flex-col bg-panel items-center flex-1 py-20 px-4 sm:px-6 lg:px-8">
      <DesertScene />

      <h1 className="relative z-10 text-5xl font-bold mb-4 text-ink [text-shadow:0_0_6px_var(--color-panel),0_0_12px_var(--color-panel),0_0_24px_var(--color-panel),0_0_40px_var(--color-panel)]">
        Pricing
      </h1>
      <p className="relative z-10 text-md text-muted [text-shadow:0_0_4px_var(--color-panel),0_0_8px_var(--color-panel),0_0_16px_var(--color-panel),0_0_28px_var(--color-panel)]">
        Choose the plan that fits your needs.
      </p>

      <div className="relative z-10 mt-16 grid grid-cols-1 sm:grid-cols-2 gap-6 items-stretch w-full max-w-4xl">
        <div className="bg-surface rounded-xl p-8 flex flex-col">
          <h2 className="text-2xl font-bold text-ink">
            Monthly{" "}
            <span className="ml-1 text-base font-normal text-muted">
              billed monthly
            </span>
          </h2>
          <p className="mt-4 text-sm text-muted">
            Our monthly plan grants access to{" "}
            <span className="font-semibold text-ink">all premium features</span>
            , the best plan for short-term subscribers.
          </p>

          <div className="mt-auto pt-12 flex items-baseline gap-2">
            <span className="text-3xl text-dim line-through">$39</span>
            <span className="text-4xl font-bold text-ink">$35</span>
            <span className="text-muted">/mo</span>
          </div>
          <p className="mt-2 text-sm text-dim">Prices are marked in USD</p>

          <button className="mt-8 w-full bg-accent hover:bg-accent-light text-shell font-semibold px-4 py-3 rounded-lg">
            Subscribe
          </button>
        </div>

        <div className="relative bg-linear-to-br from-accent-light to-accent rounded-xl p-8 flex flex-col text-shell shadow-2xl shadow-accent/20">
          <span className="self-start rounded-md bg-shell/15 px-3 py-1 text-sm font-semibold">
            🎉 Most popular
          </span>

          <h2 className="mt-5 text-2xl font-bold">
            Yearly{" "}
            <span className="ml-1 text-base font-normal text-shell/70">
              billed yearly (
              <span className="font-semibold text-shell">$159</span>)
            </span>
          </h2>
          <p className="mt-4 text-sm text-shell/70">
            Our <span className="font-semibold text-shell">most popular</span>{" "}
            plan previously sold for $299 and is now only{" "}
            <span className="font-semibold text-shell">$13.25/month</span>.
            <br />
            This plan{" "}
            <span className="font-semibold text-shell">
              saves you over 62%
            </span>{" "}
            in comparison to the monthly plan. 
          </p>

          <div className="mt-auto pt-12 flex items-baseline gap-2">
            <span className="text-3xl text-shell/50 line-through">$24.91</span>
            <span className="text-4xl font-bold">$13.25</span>
            <span className="text-shell/70">/mo</span>
          </div>
          <p className="mt-2 text-sm text-shell/60">Prices are marked in USD</p>

          <button className="mt-8 w-full bg-shell hover:bg-panel text-ink font-semibold px-4 py-3 rounded-lg">
            Subscribe
          </button>
        </div>
      </div>
    </div>
  );
}
