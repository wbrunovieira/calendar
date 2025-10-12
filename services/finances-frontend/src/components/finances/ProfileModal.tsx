'use client';

import { useState, useEffect } from 'react';

interface Calendar {
  id: string;
  name: string;
  email: string | null;
}

interface Profile {
  id?: string;
  calendarId: string;
  name: string;
  type: 'PERSONAL' | 'BUSINESS';
}

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (profile: Profile) => void;
  profile?: Profile | null;
  calendars: Calendar[];
}

export default function ProfileModal({
  isOpen,
  onClose,
  onSave,
  profile,
  calendars,
}: ProfileModalProps) {
  const [formData, setFormData] = useState<Profile>({
    calendarId: '',
    name: '',
    type: 'PERSONAL',
  });

  useEffect(() => {
    if (profile) {
      setFormData(profile);
    } else {
      setFormData({
        calendarId: '',
        name: '',
        type: 'PERSONAL',
      });
    }
  }, [profile, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSave(formData);
    onClose();
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div
        className="absolute inset-0 bg-black/50 backdrop-blur-sm"
        onClick={onClose}
      ></div>

      {/* Modal */}
      <div className="relative bg-gradient-to-br from-purple-900/90 to-blue-900/90 backdrop-blur-lg rounded-2xl p-8 w-full max-w-md border border-white/10 shadow-2xl">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">
            {profile ? 'Editar Perfil' : 'Novo Perfil'}
          </h2>
          <button
            onClick={onClose}
            className="text-white/70 hover:text-white transition-colors"
          >
            <svg
              className="w-6 h-6"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Name */}
          <div>
            <label className="block text-white font-semibold mb-2">
              Nome do Perfil
            </label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) =>
                setFormData({ ...formData, name: e.target.value })
              }
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-xl text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-white/30"
              placeholder="Ex: Pessoal, Empresa"
              required
            />
          </div>

          {/* Type */}
          <div>
            <label className="block text-white font-semibold mb-2">Tipo</label>
            <select
              value={formData.type}
              onChange={(e) =>
                setFormData({
                  ...formData,
                  type: e.target.value as 'PERSONAL' | 'BUSINESS',
                })
              }
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-white/30"
              required
            >
              <option value="PERSONAL" className="bg-gray-800">
                Pessoal
              </option>
              <option value="BUSINESS" className="bg-gray-800">
                Empresarial
              </option>
            </select>
          </div>

          {/* Calendar */}
          <div>
            <label className="block text-white font-semibold mb-2">
              Calendário Vinculado
            </label>
            <select
              value={formData.calendarId}
              onChange={(e) =>
                setFormData({ ...formData, calendarId: e.target.value })
              }
              className="w-full px-4 py-3 bg-white/10 border border-white/20 rounded-xl text-white focus:outline-none focus:ring-2 focus:ring-white/30"
              required
              disabled={!!profile}
            >
              <option value="" className="bg-gray-800">
                Selecione um calendário
              </option>
              {calendars.map((calendar) => (
                <option
                  key={calendar.id}
                  value={calendar.id}
                  className="bg-gray-800"
                >
                  {calendar.name} {calendar.email ? `(${calendar.email})` : ''}
                </option>
              ))}
            </select>
            {profile && (
              <p className="text-white/60 text-sm mt-2">
                O calendário não pode ser alterado após a criação
              </p>
            )}
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-4">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 bg-white/10 hover:bg-white/20 text-white rounded-xl font-semibold transition-all duration-300 border border-white/20"
            >
              Cancelar
            </button>
            <button
              type="submit"
              className="flex-1 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl border border-white/20"
            >
              Salvar
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
