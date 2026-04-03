export const formatCurrency = (value: number, currency = 'BRL') =>
  new Intl.NumberFormat('pt-BR', { style: 'currency', currency }).format(value);

export const formatCurrencyCompact = (value: number) =>
  new Intl.NumberFormat('pt-BR', {
    style: 'currency',
    currency: 'BRL',
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);

export const parseLocalDate = (value: string) => {
  const [year, month, day] = value.slice(0, 10).split('-').map(Number);
  return new Date(year, month - 1, day);
};

export const getLocalDateString = (date?: Date) => {
  const d = date || new Date();
  const year = d.getFullYear();
  const month = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${year}-${month}-${day}`;
};

type DateFormat = 'short' | 'medium' | 'full' | 'text' | 'text-short';

const dateFormatOptions: Record<DateFormat, Intl.DateTimeFormatOptions> = {
  short: { day: '2-digit', month: '2-digit' },
  medium: { day: '2-digit', month: '2-digit', year: '2-digit' },
  full: { day: '2-digit', month: '2-digit', year: 'numeric' },
  text: { day: '2-digit', month: 'short', year: 'numeric' },
  'text-short': { day: '2-digit', month: 'short' },
};

export const formatDate = (dateStr: string, format: DateFormat = 'medium') => {
  const raw = dateStr.includes('T') ? dateStr.replace('Z', '') : dateStr + 'T12:00:00';
  const date = new Date(raw);
  return date.toLocaleDateString('pt-BR', dateFormatOptions[format]);
};
