import type { Calendar } from '@/types/calendar';

/**
 * Calendários hardcoded (serão substituídos por API futuramente)
 */
export const calendars: Calendar[] = [
  {
    id: 'wb-digital-calendar',
    name: 'WB Digital Solutions',
    email: 'bruno@wbdigitalsolutions.com',
    color: '#350545',
    type: 'professional',
    isActive: true,
  },
  {
    id: 'bruno-personal-calendar',
    name: 'Bruno - Pessoal',
    email: 'wbrunovieira77@gmail.com',
    color: '#792990',
    type: 'personal',
    isActive: true,
  },
];
