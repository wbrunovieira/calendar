'use client';

import { useEffect, useState } from 'react';
import AppLayout from '@/components/navigation/AppLayout';
import CreateCategoryModal from '@/components/modals/CreateCategoryModal';
import CreateCategoryTypeModal from '@/components/modals/CreateCategoryTypeModal';
import { calendars } from '@/data/calendars';
import { Category, CategoryType } from '@/types/calendar';
import { api } from '@/lib/api';

type TabType = 'categories' | 'types';

export default function SettingsPage() {
  const [activeTab, setActiveTab] = useState<TabType>('categories');
  const [categoriesByCalendar, setCategoriesByCalendar] = useState<Record<string, Category[]>>({});
  const [categoryTypes, setCategoryTypes] = useState<CategoryType[]>([]);
  const [loading, setLoading] = useState(true);
  const [isCategoryModalOpen, setIsCategoryModalOpen] = useState(false);
  const [isTypeModalOpen, setIsTypeModalOpen] = useState(false);
  const [editingCategory, setEditingCategory] = useState<Category | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const categoriesData: Record<string, Category[]> = {};

        // Fetch categories for each calendar
        for (const calendar of calendars) {
          const categories = await api.categories.list(calendar.id);
          categoriesData[calendar.id] = categories;
        }

        // Fetch category types
        const types = await api.categoryTypes.list();

        setCategoriesByCalendar(categoriesData);
        setCategoryTypes(types);
      } catch (error) {
        console.error('Error fetching data:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, []);

  const handleCreateCategory = async (categoryData: {
    calendarId: string;
    name: string;
    icon: string;
    color: string;
    type: string;
  }) => {
    const newCategory = await api.categories.create(categoryData);

    setCategoriesByCalendar((prev) => {
      const calendarCategories = prev[categoryData.calendarId] || [];
      return {
        ...prev,
        [categoryData.calendarId]: [...calendarCategories, newCategory],
      };
    });
  };

  const handleEditCategory = async (categoryData: {
    calendarId: string;
    name: string;
    icon: string;
    color: string;
    type: string;
  }) => {
    if (!editingCategory) return;

    const updatedCategory = await api.categories.update(editingCategory.id, categoryData);

    setCategoriesByCalendar((prev) => {
      const calendarCategories = prev[categoryData.calendarId] || [];
      return {
        ...prev,
        [categoryData.calendarId]: calendarCategories.map((cat) =>
          cat.id === editingCategory.id ? updatedCategory : cat
        ),
      };
    });

    setEditingCategory(null);
  };

  const handleDeleteCategory = async (categoryId: string, calendarId: string) => {
    if (!confirm('Tem certeza que deseja deletar esta categoria?')) {
      return;
    }

    try {
      await api.categories.delete(categoryId);

      setCategoriesByCalendar((prev) => {
        const calendarCategories = prev[calendarId] || [];
        return {
          ...prev,
          [calendarId]: calendarCategories.filter((cat) => cat.id !== categoryId),
        };
      });
    } catch (error) {
      console.error('Error deleting category:', error);
      alert('Erro ao deletar categoria. Tente novamente.');
    }
  };

  const handleCreateType = async (typeData: {
    calendarId: string;
    name: string;
    value: string;
    color: string;
    icon?: string;
  }) => {
    const newType = await api.categoryTypes.create(typeData);
    setCategoryTypes((prev) => [...prev, newType]);
  };

  const handleDeleteType = async (typeId: string) => {
    if (!confirm('Tem certeza que deseja deletar este tipo?')) {
      return;
    }

    try {
      await api.categoryTypes.delete(typeId);
      setCategoryTypes((prev) => prev.filter((type) => type.id !== typeId));
    } catch (error) {
      console.error('Error deleting type:', error);
      alert('Erro ao deletar tipo. Tente novamente.');
    }
  };

  return (
    <AppLayout>
      <div className="flex-1 w-full py-8 relative">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-4xl font-extrabold text-white drop-shadow-lg mb-2">
            Configurações
          </h1>
          <p className="text-white/70 text-lg">
            Gerencie categorias e tipos do seu calendário
          </p>
        </div>

        {/* Tabs */}
        <div className="flex gap-2 mb-6">
          <button
            onClick={() => setActiveTab('categories')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'categories'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Categorias
          </button>
          <button
            onClick={() => setActiveTab('types')}
            className={`px-6 py-3 rounded-xl font-semibold transition-all duration-300 ${
              activeTab === 'types'
                ? 'bg-white/20 text-white shadow-lg'
                : 'bg-white/5 text-white/70 hover:bg-white/10'
            }`}
          >
            Tipos
          </button>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-4 border-white/20 border-t-white"></div>
          </div>
        )}

        {/* Categories Tab */}
        {!loading && activeTab === 'categories' && (
          <>
            <div className="mb-6 flex justify-end">
              <button
                onClick={() => setIsCategoryModalOpen(true)}
                className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                <span>Nova Categoria</span>
              </button>
            </div>

            <div className="space-y-8">
              {calendars.map((calendar) => {
                const categories = categoriesByCalendar[calendar.id] || [];

                return (
                  <div
                    key={calendar.id}
                    className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-2xl"
                  >
                    {/* Calendar Header */}
                    <div className="flex items-center gap-4 mb-6 pb-4 border-b border-white/10">
                      <div
                        className="w-4 h-4 rounded-full shadow-lg"
                        style={{ backgroundColor: calendar.color }}
                      />
                      <div>
                        <h2 className="text-2xl font-bold text-white">{calendar.name}</h2>
                        <p className="text-white/60 text-sm">{calendar.email}</p>
                      </div>
                    </div>

                    {/* Categories Grid */}
                    {categories.length === 0 ? (
                      <div className="text-center py-12">
                        <p className="text-white/50 text-lg">
                          Nenhuma categoria encontrada para este calendário
                        </p>
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {categories.map((category) => (
                          <div
                            key={category.id}
                            className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/10 hover:bg-white/15 transition-all duration-300 hover:scale-105 hover:shadow-lg group relative"
                          >
                            {/* Action Buttons */}
                            <div className="absolute top-2 right-2 flex gap-1 opacity-0 group-hover:opacity-100 transition-all duration-300">
                              {/* Edit Button */}
                              <button
                                onClick={() => setEditingCategory(category)}
                                className="p-1.5 bg-blue-500/80 hover:bg-blue-600 text-white rounded-lg transition-all hover:scale-110"
                                title="Editar categoria"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                                </svg>
                              </button>

                              {/* Delete Button */}
                              <button
                                onClick={() => handleDeleteCategory(category.id, calendar.id)}
                                className="p-1.5 bg-red-500/80 hover:bg-red-600 text-white rounded-lg transition-all hover:scale-110"
                                title="Deletar categoria"
                              >
                                <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                                </svg>
                              </button>
                            </div>

                            <div className="flex items-center gap-3">
                              {category.icon && (
                                <div className="text-2xl flex-shrink-0 transition-transform duration-300 group-hover:scale-110">
                                  {category.icon}
                                </div>
                              )}

                              <div className="flex-1 overflow-hidden">
                                <h3 className="text-white font-semibold text-lg truncate">
                                  {category.name}
                                </h3>
                                {category.type && (
                                  <p className="text-white/60 text-sm capitalize">
                                    {category.type}
                                  </p>
                                )}
                              </div>

                              <div
                                className="w-8 h-8 rounded-lg shadow-md flex-shrink-0 border-2 border-white/20"
                                style={{ backgroundColor: category.color }}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </>
        )}

        {/* Types Tab */}
        {!loading && activeTab === 'types' && (
          <>
            <div className="mb-6 flex justify-end">
              <button
                onClick={() => setIsTypeModalOpen(true)}
                className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20"
              >
                <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
                </svg>
                <span>Novo Tipo</span>
              </button>
            </div>

            <div className="space-y-8">
              {calendars.map((calendar) => {
                const types = categoryTypes.filter((type) => type.calendarId === calendar.id);

                return (
                  <div
                    key={calendar.id}
                    className="bg-white/5 backdrop-blur-sm rounded-2xl p-6 border border-white/10 shadow-2xl"
                  >
                    {/* Calendar Header */}
                    <div className="flex items-center gap-4 mb-6 pb-4 border-b border-white/10">
                      <div
                        className="w-4 h-4 rounded-full shadow-lg"
                        style={{ backgroundColor: calendar.color }}
                      />
                      <div>
                        <h2 className="text-2xl font-bold text-white">{calendar.name}</h2>
                        <p className="text-white/60 text-sm">{calendar.email}</p>
                      </div>
                    </div>

                    {/* Types Grid */}
                    {types.length === 0 ? (
                      <div className="text-center py-12">
                        <p className="text-white/50 text-lg">
                          Nenhum tipo encontrado para este calendário
                        </p>
                      </div>
                    ) : (
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
                        {types.map((type) => (
                          <div
                            key={type.id}
                            className="bg-white/10 backdrop-blur-sm rounded-xl p-4 border border-white/10 hover:bg-white/15 transition-all duration-300 hover:scale-105 hover:shadow-lg group relative"
                          >
                            {/* Delete Button */}
                            <button
                              onClick={() => handleDeleteType(type.id)}
                              className="absolute top-2 right-2 p-1.5 bg-red-500/80 hover:bg-red-600 text-white rounded-lg opacity-0 group-hover:opacity-100 transition-all duration-300 hover:scale-110"
                              title="Deletar tipo"
                            >
                              <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                              </svg>
                            </button>

                            <div className="flex items-center gap-3">
                              {type.icon && (
                                <div className="text-2xl flex-shrink-0 transition-transform duration-300 group-hover:scale-110">
                                  {type.icon}
                                </div>
                              )}

                              <div className="flex-1 overflow-hidden">
                                <h3 className="text-white font-semibold text-lg truncate">
                                  {type.name}
                                </h3>
                                <p className="text-white/60 text-sm font-mono">
                                  {type.value}
                                </p>
                              </div>

                              <div
                                className="w-8 h-8 rounded-lg shadow-md flex-shrink-0 border-2 border-white/20"
                                style={{ backgroundColor: type.color }}
                              />
                            </div>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </>
        )}

        {/* Modals */}
        <CreateCategoryModal
          isOpen={isCategoryModalOpen}
          onClose={() => setIsCategoryModalOpen(false)}
          onSave={handleCreateCategory}
        />

        <CreateCategoryModal
          isOpen={!!editingCategory}
          onClose={() => setEditingCategory(null)}
          onSave={handleEditCategory}
          initialData={editingCategory || undefined}
        />

        <CreateCategoryTypeModal
          isOpen={isTypeModalOpen}
          onClose={() => setIsTypeModalOpen(false)}
          onSave={handleCreateType}
          calendars={calendars}
        />
      </div>
    </AppLayout>
  );
}
