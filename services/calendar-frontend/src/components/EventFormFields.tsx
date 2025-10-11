/**
 * Event Form Fields Component
 * All basic fields for event creation/editing
 */

import FormSelect from './FormSelect';
import FormInput from './FormInput';
import FormTextarea from './FormTextarea';

interface EventFormFieldsProps {
  formData: {
    calendarId: string;
    categoryId: string;
    title: string;
    description: string;
    startDate: string;
    endDate: string;
    startTime: string;
    endTime: string;
  };
  calendarOptions: Array<{ value: string; label: string }>;
  categoryOptions: Array<{ value: string; label: string }>;
  onCalendarChange: (value: string) => void;
  onCategoryChange: (value: string) => void;
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
  onCalendarChange,
  onCategoryChange,
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

      {/* Categoria */}
      <FormSelect
        label="Categoria"
        value={formData.categoryId}
        onChange={onCategoryChange}
        options={categoryOptions}
        placeholder="Sem categoria"
        disabled={!formData.calendarId}
      />

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
