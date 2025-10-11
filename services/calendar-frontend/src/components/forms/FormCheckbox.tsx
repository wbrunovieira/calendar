/**
 * Form Checkbox Component
 * Reusable checkbox input with label
 */

interface FormCheckboxProps {
  label: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  colorClass?: string;
}

export default function FormCheckbox({
  label,
  checked,
  onChange,
  colorClass = 'text-[#792990]',
}: FormCheckboxProps) {
  return (
    <label className="flex items-center gap-2 cursor-pointer">
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        className={`w-4 h-4 ${colorClass} rounded focus:ring-2 focus:ring-${colorClass.split('-')[1]}`}
      />
      <span className="text-sm font-medium text-white">{label}</span>
    </label>
  );
}
