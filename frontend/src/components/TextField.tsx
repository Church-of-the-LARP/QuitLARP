import type { AnyFieldApi } from '@tanstack/react-form';
import FieldErrors from './FieldErrors.tsx';

export default function TextField({
  field,
  label,
  type = 'text',
  autoComplete,
  placeholder,
}: {
  field: AnyFieldApi;
  label: string;
  type?: string;
  autoComplete?: string;
  placeholder?: string;
}) {
  return (
    <label className="block">
      <span className="mb-1 block text-xs font-medium uppercase tracking-wide text-zinc-400">
        {label}
      </span>
      <input
        id={field.name}
        name={field.name}
        type={type}
        autoComplete={autoComplete}
        placeholder={placeholder}
        value={field.state.value}
        onBlur={field.handleBlur}
        onChange={(e) => field.handleChange(e.target.value)}
        className="w-full rounded-md border border-zinc-700 bg-zinc-800 px-3 py-2 text-sm text-zinc-100 outline-none focus:border-teal-500 focus:ring-2 focus:ring-teal-500/30"
      />
      <FieldErrors field={field} />
    </label>
  );
}
