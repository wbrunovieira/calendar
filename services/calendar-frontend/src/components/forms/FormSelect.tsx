/**
 * Form Select Component
 * Reusable select input for forms
 */

interface FormSelectProps {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: Array<{ value: string; label: string }>;
  required?: boolean;
  disabled?: boolean;
  placeholder?: string;
}

export default function FormSelect({
  label,
  value,
  onChange,
  options,
  required = false,
  disabled = false,
  placeholder = 'Selecione uma opção',
}: FormSelectProps) {
  return (
    <div>
      <label className="block text-sm font-semibold text-white mb-2">
        {label} {required && <span className="text-red-400">*</span>}
      </label>
      <select
        value={value}
        onChange={e => onChange(e.target.value)}
        className="w-full px-4 py-2 bg-white/10 border border-white/20 text-white rounded-lg focus:ring-2 focus:ring-[#792990] focus:border-transparent disabled:opacity-50"
        required={required}
        disabled={disabled}
      >
        <option value="" className="bg-[#350545] text-white">
          {placeholder}
        </option>
        {options.map(option => (
          <option key={option.value} value={option.value} className="bg-[#350545] text-white">
            {option.label}
          </option>
        ))}
      </select>
    </div>
  );
}
