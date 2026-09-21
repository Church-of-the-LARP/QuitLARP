export default function Pricing() {
  return (
    <div className="flex flex-col items-center min-h-screen bg-zinc-950 py-20 px-4 sm:px-6 lg:px-8">
      <h1 className="text-5xl font-bold mb-4 text-zinc-100">Pricing</h1>
      <p className="text-md text-zinc-400">
        Choose the plan that fits your needs.
      </p>

      <div className="mt-16 grid grid-cols-1 sm:grid-cols-2 gap-6 items-stretch w-full max-w-4xl">
        <div className="bg-zinc-800 border border-zinc-700 rounded-xl p-8 flex flex-col">
          <h2 className="text-2xl font-bold text-zinc-100">
            Monthly{" "}
            <span className="ml-1 text-base font-normal text-zinc-400">
              billed monthly
            </span>
          </h2>
          <p className="mt-4 text-sm text-zinc-400">
            Our monthly plan grants access to{" "}
            <span className="font-semibold text-zinc-100">
              all premium features
            </span>
            , the best plan for short-term subscribers.
          </p>

          <div className="mt-auto pt-12 flex items-baseline gap-2">
            <span className="text-3xl text-zinc-500 line-through">$39</span>
            <span className="text-4xl font-bold text-zinc-100">$35</span>
            <span className="text-zinc-400">/mo</span>
          </div>
          <p className="mt-2 text-sm text-zinc-500">Prices are marked in USD</p>

          <button className="mt-8 w-full bg-teal-500 hover:bg-teal-400 text-zinc-950 font-semibold px-4 py-3 rounded-lg">
            Subscribe
          </button>
        </div>

        <div className="relative bg-linear-to-br from-teal-400 to-teal-500 rounded-xl p-8 flex flex-col text-zinc-950 shadow-2xl shadow-teal-500/20">
          <span className="self-start rounded-md bg-zinc-950/15 px-3 py-1 text-sm font-semibold">
            🎉 Most popular
          </span>

          <h2 className="mt-5 text-2xl font-bold">
            Yearly{" "}
            <span className="ml-1 text-base font-normal text-zinc-950/70">
              billed yearly (
              <span className="font-semibold text-zinc-950">$159</span>)
            </span>
          </h2>
          <p className="mt-4 text-sm text-zinc-950/70">
            Our{" "}
            <span className="font-semibold text-zinc-950">most popular</span>{" "}
            plan previously sold for $299 and is now only{" "}
            <span className="font-semibold text-zinc-950">$13.25/month</span>.
            <br />
            This plan{" "}
            <span className="font-semibold text-zinc-950">
              saves you over 62%
            </span>{" "}
            in comparison to the monthly plan.
          </p>

          <div className="mt-auto pt-12 flex items-baseline gap-2">
            <span className="text-3xl text-zinc-950/50 line-through">
              $24.91
            </span>
            <span className="text-4xl font-bold">$13.25</span>
            <span className="text-zinc-950/70">/mo</span>
          </div>
          <p className="mt-2 text-sm text-zinc-950/60">
            Prices are marked in USD
          </p>

          <button className="mt-8 w-full bg-zinc-950 hover:bg-zinc-900 text-zinc-100 font-semibold px-4 py-3 rounded-lg">
            Subscribe
          </button>
        </div>
      </div>
    </div>
  );
}
