/**
 * Event Form Fields Component
 * All basic fields for event creation/editing
 */

import { CategoryType } from '@/types/calendar';
import FormSelect from '../forms/FormSelect';
import FormInput from '../forms/FormInput';
import FormTextarea from '../forms/FormTextarea';

interface EventFormFieldsProps {
  formData: {
    calendarId: string;
    categoryId: string;
    categoryTypeId: string;
    title: string;
    description: string;
    startDate: string;
    endDate: string;
    startTime: string;
    endTime: string;
  };
  calendarOptions: Array<{ value: string; label: string }>;
  categoryOptions: Array<{ value: string; label: string }>;
  categoryTypeOptions: Array<{ value: string; label: string }>;
  availableTypes: CategoryType[]; // Tipos disponíveis para a categoria selecionada
  onCalendarChange: (value: string) => void;
  onCategoryChange: (value: string) => void;
  onCategoryTypeChange: (value: string) => void;
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onStartDateChange: (value: string) => void;
  onEndDateChange: (value: string) => void;
  onStartTimeChange: (value: string) => void;
  onEndTimeChange: (value: string) => void;
}

export default function EventFormFields({
  formData,
  calendarOptions,
  categoryOptions,
  categoryTypeOptions,
  availableTypes,
  onCalendarChange,
  onCategoryChange,
  onCategoryTypeChange,
  onTitleChange,
  onDescriptionChange,
  onStartDateChange,
  onEndDateChange,
  onStartTimeChange,
  onEndTimeChange,
}: EventFormFieldsProps) {
  return (
    <>
      {/* Calendário */}
      <FormSelect
        label="Calendário"
        value={formData.calendarId}
        onChange={onCalendarChange}
        options={calendarOptions}
        placeholder="Selecione um calendário"
        required
      />

      {/* Tipo (Opcional) */}
      <FormSelect
        label="Tipo (opcional)"
        value={formData.categoryTypeId}
        onChange={onCategoryTypeChange}
        options={categoryTypeOptions}
        placeholder="Selecione o tipo (a categoria será selecionada automaticamente)"
        disabled={!formData.calendarId}
      />

      {/* Categoria */}
      <FormSelect
        label="Categoria (opcional)"
        value={formData.categoryId}
        onChange={onCategoryChange}
        options={categoryOptions}
        placeholder="Selecione a categoria"
        disabled={!formData.calendarId || !!formData.categoryTypeId}
      />

      {/* Se categoria foi escolhida e tem múltiplos tipos, mostrar seleção de tipo */}
      {formData.categoryId && !formData.categoryTypeId && availableTypes.length > 0 && (
        <div className="p-4 bg-white/5 rounded-lg border border-white/10">
          <label className="block text-white font-semibold mb-3">
            Selecione o tipo desta atividade:
          </label>
          <div className="grid grid-cols-1 gap-2">
            {availableTypes.map((type) => (
              <button
                key={type.id}
                type="button"
                onClick={() => onCategoryTypeChange(type.id)}
                className={`flex items-center gap-3 p-3 rounded-lg border transition-all ${
                  formData.categoryTypeId === type.id
                    ? 'bg-white/20 border-white/30'
                    : 'bg-white/10 border-white/10 hover:bg-white/15'
                }`}
              >
                {type.icon && <span className="text-xl">{type.icon}</span>}
                <span className="text-white font-medium flex-1 text-left">{type.name}</span>
                <div
                  className="w-4 h-4 rounded flex-shrink-0"
                  style={{ backgroundColor: type.color }}
                />
              </button>
            ))}
          </div>
        </div>
      )}

      {/* Título */}
      <FormInput
        label="Título"
        type="text"
        value={formData.title}
        onChange={onTitleChange}
        placeholder="Ex: Reunião com cliente"
        required
      />

      {/* Descrição */}
      <FormTextarea
        label="Descrição"
        value={formData.description}
        onChange={onDescriptionChange}
        placeholder="Detalhes do evento..."
      />

      {/* Data e Hora */}
      <div className="grid grid-cols-2 gap-4">
        <FormInput label="Data Início" type="date" value={formData.startDate} onChange={onStartDateChange} required />
        <FormInput label="Data Fim" type="date" value={formData.endDate} onChange={onEndDateChange} />
      </div>

      <div className="grid grid-cols-2 gap-4">
        <FormInput label="Hora Início" type="time" value={formData.startTime} onChange={onStartTimeChange} required />
        <FormInput label="Hora Fim" type="time" value={formData.endTime} onChange={onEndTimeChange} />
      </div>
    </>
  );
}
