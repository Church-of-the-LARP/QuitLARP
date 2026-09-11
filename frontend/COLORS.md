# Color system

QuitLARP uses a dark zinc/teal palette. This is the single source of truth for
which Tailwind class to reach for — don't introduce new colors outside this
list without updating this doc.

## Surfaces (layering, darkest → lightest)

| Role              | Class                              | Where                                                  |
| ------------------ | ----------------------------------- | ------------------------------------------------------- |
| Shell               | `bg-zinc-950`                       | `<body>`/root background, page-level wrapper            |
| Panel / Editor      | `bg-zinc-900`                       | Nav header, auth card container, input fields           |
| Card / Sidebar      | `bg-zinc-800`                       | Content cards, list rows, form sections                 |
| Hover / Border      | `bg-zinc-700` · `border-zinc-800` / `border-zinc-700` | Hover states, dividers, outlines of the layer above |

Set once globally in [src/index.css](src/index.css) on `html, body, #root` so
every page gets the shell background even without an explicit wrapper.

Rule of thumb: each nested surface should be one step lighter than its
parent (shell → panel → card), and borders use the zinc tone one step
darker/lighter than the surface they outline.

## Text

| Role          | Class            | Use                                               |
| -------------- | ----------------- | --------------------------------------------------- |
| Text primary   | `text-zinc-100`   | Headings, body copy, primary labels                 |
| Text muted     | `text-zinc-400`   | Secondary copy, helper text, timestamps             |
| Text disabled  | `text-zinc-500`   | Placeholder-like/disabled/least-important text      |

## Accent (teal)

| Role            | Class                          | Use                                         |
| ---------------- | -------------------------------- | ---------------------------------------------- |
| Accent bg        | `bg-teal-500` `hover:bg-teal-400` | Primary buttons                              |
| Accent text      | `text-teal-400`                  | Section eyebrows, links, active/accent labels |
| Accent on-btn    | `text-zinc-950`                  | Text color sitting on top of an accent button |
| Focus ring        | `ring-teal-500` (used as `focus:ring-2 focus:ring-teal-500/30 focus:border-teal-500`) | Inputs/selects on focus |

## Semantic status colors

These are reserved for status/severity indicators — not general UI chrome.

| Meaning            | Class            | Currently used for                          |
| -------------------- | ----------------- | ---------------------------------------------- |
| Easy / Beginner       | `text-emerald-400` | Difficulty badges (e.g. public test cards)   |
| Medium / Intermediate | `text-amber-400`   | Difficulty badges; also reused for warning banners (verify-email) and the `superadmin` role pill |
| Hard / Advanced       | `text-rose-400`    | Difficulty badges; destructive actions (Remove chapter) |
| Accepted              | `text-green-400`   | Reserved for future test-run/judge results   |
| Wrong                 | `text-red-400`     | Reserved for future test-run/judge results   |
| Error                 | `text-orange-400`  | Reserved for future test-run/judge results   |
| Running               | `text-sky-400`     | Reserved for future test-run/judge results   |

The judge-run statuses (Accepted/Wrong/Error/Running) aren't wired into any
screen yet — no submission-result UI exists. When that's built, use these
classes directly rather than inventing new ones.

## Applying colors: quick recipes

- **Page shell**: `bg-zinc-950` wrapping `min-h-screen`.
- **Nav / header**: `bg-zinc-900 border-b border-zinc-800`.
- **Card**: `bg-zinc-800 border border-zinc-700 rounded-xl`.
- **Input**: `bg-zinc-900 border border-zinc-700 text-zinc-100 placeholder:text-zinc-500 focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30` (use `bg-zinc-800` instead when the input sits inside a card that's already `bg-zinc-900`, so it stays one step lighter than its parent).
- **Primary button**: `bg-teal-500 hover:bg-teal-400 text-zinc-950 font-semibold`.
- **Secondary/outline button**: `border border-zinc-700 bg-zinc-900 text-zinc-300 hover:bg-zinc-800` (or `hover:bg-zinc-700` on a card background).
- **Destructive/link action**: `text-rose-400 hover:text-rose-300`.
- **Inline alert**: tint the semantic color at low opacity so it reads on dark surfaces, e.g. `border-rose-400/30 bg-rose-400/10 text-rose-400` for errors, `border-teal-400/30 bg-teal-400/10 text-teal-400` for notices, `border-amber-400/30 bg-amber-400/10 text-amber-400` for warnings.
- **Role/status pill**: tint at ~20% opacity, e.g. `bg-teal-400/20 text-teal-400`.

## Where it's implemented

- [src/index.css](src/index.css) — global shell background/text.
- [src/components/Layout.tsx](src/components/Layout.tsx) — nav shell, role pills.
- [src/components/Alert.tsx](src/components/Alert.tsx) — error/notice tones.
- [src/pages/Dashboard.tsx](src/pages/Dashboard.tsx) — cards, buttons, difficulty colors.
- [src/pages/Create.tsx](src/pages/Create.tsx) — form sections and inputs.
- [src/pages/Login.tsx](src/pages/Login.tsx), [src/pages/Register.tsx](src/pages/Register.tsx), [src/pages/ForgotPassword.tsx](src/pages/ForgotPassword.tsx), [src/pages/ResetPassword.tsx](src/pages/ResetPassword.tsx) — auth card pattern.
- [src/pages/Users.tsx](src/pages/Users.tsx) — list rows, role pills, admin actions.
