export default function Alert({
  kind,
  text,
}: {
  kind: 'error' | 'notice';
  text: string;
}) {
  const tone =
    kind === 'error'
      ? 'border-danger/30 bg-danger/10 text-danger'
      : 'border-accent-light/30 bg-accent-light/10 text-accent-light';

  return (
    <div className={`rounded-md border px-3 py-2 text-sm ${tone}`}>{text}</div>
  );
}
