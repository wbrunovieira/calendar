/**
 * Form Input Component
 * Reusable input for forms (text, date, time)
 */

interface FormInputProps {
  label: string;
  type: 'text' | 'date' | 'time';
  value: string;
  onChange: (value: string) => void;
  required?: boolean;
  placeholder?: string;
}

export default function FormInput({
  label,
  type,
  value,
  onChange,
  required = false,
  placeholder,
}: FormInputProps) {
  return (
    <div>
      <label className="block text-sm font-semibold text-white mb-2">
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      <input
        type={type}
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white placeholder-white/50 rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent"
        placeholder={placeholder}
        required={required}
      />
    </div>
  );
}
