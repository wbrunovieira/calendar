/**
 * Form Textarea Component
 * Reusable textarea for forms
 */

interface FormTextareaProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  rows?: number;
}

export default function FormTextarea({
  label,
  value,
  onChange,
  placeholder,
  rows = 3,
}: FormTextareaProps) {
  return (
    <div>
      <label className="block text-sm font-semibold text-white mb-2">{label}</label>
      <textarea
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white placeholder-white/50 rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent resize-none"
        rows={rows}
        placeholder={placeholder}
      />
    </div>
  );
}
