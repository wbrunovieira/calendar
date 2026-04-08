'use client';

import { useState, useEffect } from 'react';
import type { LegalEntityType, TaxRegime } from '@/types/finances';

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
  legalEntityType?: LegalEntityType;
  companyName?: string;
  cnpj?: string;
  simplesNacional?: boolean;
  taxRegime?: TaxRegime;
  dasAliquota?: number;
  openingDate?: string;
}

interface ProfileModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSave: (profile: Profile) => void;
  profile?: Profile | null;
  calendars: Calendar[];
}

const LEGAL_ENTITY_LABELS: Record<LegalEntityType, string> = {
  MEI: 'MEI — Microempreendedor Individual',
  ME: 'ME — Microempresa',
  EPP: 'EPP — Empresa de Pequeno Porte',
  LTDA: 'LTDA — Sociedade Limitada',
  SA: 'S.A. — Sociedade Anônima',
};

const TAX_REGIME_LABELS: Record<TaxRegime, string> = {
  SIMPLES: 'Simples Nacional',
  LUCRO_PRESUMIDO: 'Lucro Presumido',
  LUCRO_REAL: 'Lucro Real',
};

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
      setFormData({ calendarId: '', name: '', type: 'PERSONAL' });
    }
  }, [profile, isOpen]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const payload: Profile = { ...formData };
    // Strip PJ fields when type is PERSONAL
    if (payload.type === 'PERSONAL') {
      delete payload.legalEntityType;
      delete payload.companyName;
      delete payload.cnpj;
      delete payload.simplesNacional;
      delete payload.taxRegime;
      delete payload.dasAliquota;
      delete payload.openingDate;
    }
    onSave(payload);
    onClose();
  };

  const isBusiness = formData.type === 'BUSINESS';

  if (!isOpen) return null;

  const inputClass =
    'w-full px-4 py-3 bg-white/10 border border-white/20 rounded-xl text-white placeholder-white/50 focus:outline-none focus:ring-2 focus:ring-white/30';
  const labelClass = 'block text-white font-semibold mb-2 text-sm';

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50 backdrop-blur-sm" onClick={onClose} />

      <div className="relative bg-gradient-to-br from-purple-900/90 to-blue-900/90 backdrop-blur-lg rounded-2xl p-8 w-full max-w-lg border border-white/10 shadow-2xl max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between mb-6">
          <h2 className="text-2xl font-bold text-white">
            {profile ? 'Editar Perfil' : 'Novo Perfil'}
          </h2>
          <button onClick={onClose} className="text-white/70 hover:text-white transition-colors">
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        <form onSubmit={handleSubmit} className="space-y-5">
          {/* Nome */}
          <div>
            <label className={labelClass}>Nome do Perfil</label>
            <input
              type="text"
              value={formData.name}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              className={inputClass}
              placeholder="Ex: WB Digital Solutions"
              required
            />
          </div>

          {/* Tipo */}
          <div>
            <label className={labelClass}>Tipo</label>
            <select
              value={formData.type}
              onChange={(e) =>
                setFormData({ ...formData, type: e.target.value as 'PERSONAL' | 'BUSINESS' })
              }
              className={inputClass}
              required
            >
              <option value="PERSONAL" className="bg-gray-800">Pessoal</option>
              <option value="BUSINESS" className="bg-gray-800">Empresarial</option>
            </select>
          </div>

          {/* PJ fields — shown only when BUSINESS */}
          {isBusiness && (
            <div className="border border-amber-500/30 rounded-xl p-4 space-y-4 bg-amber-500/5">
              <p className="text-amber-400 text-xs font-semibold uppercase tracking-wider">
                Dados da Empresa (opcional)
              </p>

              {/* Natureza jurídica */}
              <div>
                <label className={labelClass}>Natureza Jurídica</label>
                <select
                  value={formData.legalEntityType ?? ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      legalEntityType: e.target.value ? (e.target.value as LegalEntityType) : undefined,
                    })
                  }
                  className={inputClass}
                >
                  <option value="" className="bg-gray-800">Selecione (opcional)</option>
                  {(Object.keys(LEGAL_ENTITY_LABELS) as LegalEntityType[]).map((k) => (
                    <option key={k} value={k} className="bg-gray-800">{LEGAL_ENTITY_LABELS[k]}</option>
                  ))}
                </select>
              </div>

              {/* Razão Social */}
              <div>
                <label className={labelClass}>Razão Social</label>
                <input
                  type="text"
                  value={formData.companyName ?? ''}
                  onChange={(e) => setFormData({ ...formData, companyName: e.target.value || undefined })}
                  className={inputClass}
                  placeholder="Ex: WB Digital Solutions LTDA"
                />
              </div>

              {/* CNPJ */}
              <div>
                <label className={labelClass}>CNPJ</label>
                <input
                  type="text"
                  value={formData.cnpj ?? ''}
                  onChange={(e) => setFormData({ ...formData, cnpj: e.target.value || undefined })}
                  className={inputClass}
                  placeholder="00.000.000/0001-00"
                />
              </div>

              {/* Data de abertura */}
              <div>
                <label className={labelClass}>Data de Abertura</label>
                <input
                  type="date"
                  value={formData.openingDate ? formData.openingDate.slice(0, 10) : ''}
                  onChange={(e) => setFormData({ ...formData, openingDate: e.target.value || undefined })}
                  className={inputClass}
                />
              </div>

              {/* Regime tributário */}
              <div>
                <label className={labelClass}>Regime Tributário</label>
                <select
                  value={formData.taxRegime ?? ''}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      taxRegime: e.target.value ? (e.target.value as TaxRegime) : undefined,
                    })
                  }
                  className={inputClass}
                >
                  <option value="" className="bg-gray-800">Selecione (opcional)</option>
                  {(Object.keys(TAX_REGIME_LABELS) as TaxRegime[]).map((k) => (
                    <option key={k} value={k} className="bg-gray-800">{TAX_REGIME_LABELS[k]}</option>
                  ))}
                </select>
              </div>

              {/* Simples Nacional + Alíquota */}
              <div className="flex gap-4">
                <div className="flex items-center gap-2">
                  <input
                    type="checkbox"
                    id="simplesNacional"
                    checked={formData.simplesNacional ?? false}
                    onChange={(e) => setFormData({ ...formData, simplesNacional: e.target.checked })}
                    className="w-4 h-4 accent-amber-400"
                  />
                  <label htmlFor="simplesNacional" className="text-white text-sm">Simples Nacional</label>
                </div>
                {formData.simplesNacional && (
                  <div className="flex-1">
                    <input
                      type="number"
                      step="0.01"
                      min="0"
                      max="100"
                      value={formData.dasAliquota ?? ''}
                      onChange={(e) =>
                        setFormData({
                          ...formData,
                          dasAliquota: e.target.value ? parseFloat(e.target.value) : undefined,
                        })
                      }
                      className={inputClass}
                      placeholder="Alíquota DAS (%)"
                    />
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Calendário */}
          <div>
            <label className={labelClass}>Calendário Vinculado</label>
            <select
              value={formData.calendarId}
              onChange={(e) => setFormData({ ...formData, calendarId: e.target.value })}
              className={inputClass}
              required
              disabled={!!profile}
            >
              <option value="" className="bg-gray-800">Selecione um calendário</option>
              {calendars.map((calendar) => (
                <option key={calendar.id} value={calendar.id} className="bg-gray-800">
                  {calendar.name} {calendar.email ? `(${calendar.email})` : ''}
                </option>
              ))}
            </select>
            {profile && (
              <p className="text-white/60 text-xs mt-1">O calendário não pode ser alterado após a criação</p>
            )}
          </div>

          {/* Buttons */}
          <div className="flex gap-3 pt-2">
            <button
              type="button"
              onClick={onClose}
              className="flex-1 px-6 py-3 bg-white/10 hover:bg-white/20 text-white rounded-xl font-semibold transition-all border border-white/20"
            >
              Cancelar
            </button>
            <button
              type="submit"
              className="flex-1 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all shadow-lg border border-white/20"
            >
              Salvar
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
