'use client';

import { useEffect, useState } from 'react';
import AppLayout from '@/components/navigation/AppLayout';
import CreateCategoryModal from '@/components/modals/CreateCategoryModal';
import { calendars } from '@/data/calendars';
import { Category } from '@/types/calendar';
import { api } from '@/lib/api';

export default function CategoriesPage() {
  const [categoriesByCalendar, setCategoriesByCalendar] = useState<Record<string, Category[]>>({});
  const [loading, setLoading] = useState(true);
  const [isModalOpen, setIsModalOpen] = useState(false);

  useEffect(() => {
    const fetchCategories = async () => {
      try {
        const categoriesData: Record<string, Category[]> = {};

        // Fetch categories for each calendar
        for (const calendar of calendars) {
          const categories = await api.categories.list(calendar.id);
          categoriesData[calendar.id] = categories;
        }

        setCategoriesByCalendar(categoriesData);
      } catch (error) {
        console.error('Error fetching categories:', error);
      } finally {
        setLoading(false);
      }
    };

    fetchCategories();
  }, []);

  const handleCreateCategory = async (categoryData: {
    calendarId: string;
    name: string;
    icon: string;
    color: string;
    type: string;
  }) => {
    // Create category via API
    const newCategory = await api.categories.create(categoryData);

    // Update local state
    setCategoriesByCalendar((prev) => {
      const calendarCategories = prev[categoryData.calendarId] || [];
      return {
        ...prev,
        [categoryData.calendarId]: [...calendarCategories, newCategory],
      };
    });
  };

  const handleDeleteCategory = async (categoryId: string, calendarId: string) => {
    if (!confirm('Tem certeza que deseja deletar esta categoria?')) {
      return;
    }

    try {
      // Delete category via API
      await api.categories.delete(categoryId);

      // Update local state
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

  return (
    <AppLayout>
      <div className="flex-1 w-full py-8 relative">
        {/* Header */}
        <div className="mb-8 flex items-start justify-between">
          <div>
            <h1 className="text-4xl font-extrabold text-white drop-shadow-lg mb-2">
              Categorias
            </h1>
            <p className="text-white/70 text-lg">
              Gerencie as categorias dos seus calendários
            </p>
          </div>

          {/* Add Category Button */}
          <button
            onClick={() => setIsModalOpen(true)}
            className="flex items-center gap-2 px-6 py-3 bg-white/20 hover:bg-white/30 text-white rounded-xl font-semibold transition-all duration-300 shadow-lg hover:shadow-xl hover:scale-105 border border-white/20"
          >
            <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
            <span>Nova Categoria</span>
          </button>
        </div>

        {/* Loading State */}
        {loading && (
          <div className="flex items-center justify-center py-20">
            <div className="animate-spin rounded-full h-12 w-12 border-4 border-white/20 border-t-white"></div>
          </div>
        )}

        {/* Categories by Calendar */}
        {!loading && (
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
                          {/* Delete Button */}
                          <button
                            onClick={() => handleDeleteCategory(category.id, calendar.id)}
                            className="absolute top-2 right-2 p-1.5 bg-red-500/80 hover:bg-red-600 text-white rounded-lg opacity-0 group-hover:opacity-100 transition-all duration-300 hover:scale-110"
                            title="Deletar categoria"
                          >
                            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                            </svg>
                          </button>

                          <div className="flex items-center gap-3">
                            {/* Category Icon */}
                            {category.icon && (
                              <div className="text-2xl flex-shrink-0 transition-transform duration-300 group-hover:scale-110">
                                {category.icon}
                              </div>
                            )}

                            <div className="flex-1 overflow-hidden">
                              {/* Category Name */}
                              <h3 className="text-white font-semibold text-lg truncate">
                                {category.name}
                              </h3>

                              {/* Category Type */}
                              {category.type && (
                                <p className="text-white/60 text-sm capitalize">
                                  {category.type}
                                </p>
                              )}
                            </div>

                            {/* Category Color */}
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
        )}

        {/* Create Category Modal */}
        <CreateCategoryModal
          isOpen={isModalOpen}
          onClose={() => setIsModalOpen(false)}
          onSave={handleCreateCategory}
        />
      </div>
    </AppLayout>
  );
}
