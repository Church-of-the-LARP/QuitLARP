export default function Alert({
  kind,
  text,
}: {
  kind: 'error' | 'notice';
  text: string;
}) {
  const tone =
    kind === 'error'
      ? 'border-rose-400/30 bg-rose-400/10 text-rose-400'
      : 'border-teal-400/30 bg-teal-400/10 text-teal-400';

  return (
    <div className={`rounded-md border px-3 py-2 text-sm ${tone}`}>{text}</div>
  );
}
