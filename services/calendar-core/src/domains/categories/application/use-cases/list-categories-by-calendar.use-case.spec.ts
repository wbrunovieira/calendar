import { describe, it, expect, beforeEach, vi } from "vitest";
import { ListCategoriesByCalendarUseCase } from "./list-categories-by-calendar.use-case";
import { Category } from "../../domain/entities/category.entity";

const mockRepository = {
  findByCalendarId: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};

describe("ListCategoriesByCalendarUseCase", () => {
  let useCase: ListCategoriesByCalendarUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new ListCategoriesByCalendarUseCase(mockRepository as any);
  });

  it("should return categories for a specific calendar", async () => {
    const categories = [
      Category.create({
        calendarId: "cal-123",
        name: "LinkedIn",
        color: "#0077B5",
        icon: "💼",
      }),
      Category.create({
        calendarId: "cal-123",
        name: "Twitter",
        color: "#1DA1F2",
        icon: "🐦",
      }),
    ];

    mockRepository.findByCalendarId.mockResolvedValue(categories);

    const result = await useCase.execute("cal-123");

    expect(result).toEqual(categories);
    expect(mockRepository.findByCalendarId).toHaveBeenCalledWith("cal-123");
  });

  it("should return empty array when calendar has no categories", async () => {
    mockRepository.findByCalendarId.mockResolvedValue([]);

    const result = await useCase.execute("cal-empty");

    expect(result).toEqual([]);
    expect(mockRepository.findByCalendarId).toHaveBeenCalledWith("cal-empty");
  });

  it("should return categories with all properties", async () => {
    const category = Category.create({
      calendarId: "cal-123",
      name: "Health",
      color: "#22C55E",
      icon: "💪",
      type: "health",
    });

    mockRepository.findByCalendarId.mockResolvedValue([category]);

    const result = await useCase.execute("cal-123");

    expect(result[0].name).toBe("Health");
    expect(result[0].color).toBe("#22C55E");
    expect(result[0].icon).toBe("💪");
    expect(result[0].type).toBe("health");
    expect(result[0].isActive).toBe(true);
  });

  it("should propagate repository errors", async () => {
    const error = new Error("Database error");
    mockRepository.findByCalendarId.mockRejectedValue(error);

    await expect(useCase.execute("cal-123")).rejects.toThrow("Database error");
  });

  it("should call repository with correct calendar id", async () => {
    mockRepository.findByCalendarId.mockResolvedValue([]);

    await useCase.execute("wb-digital-solutions");

    expect(mockRepository.findByCalendarId).toHaveBeenCalledWith("wb-digital-solutions");
    expect(mockRepository.findByCalendarId).toHaveBeenCalledTimes(1);
  });

  it("should return multiple categories with different types", async () => {
    const categories = [
      Category.create({
        calendarId: "cal-123",
        name: "LinkedIn",
        color: "#0077B5",
        icon: "💼",
        type: "social",
      }),
      Category.create({
        calendarId: "cal-123",
        name: "Meetings",
        color: "#EF4444",
        icon: "📅",
        type: "work",
      }),
      Category.create({
        calendarId: "cal-123",
        name: "Learning",
        color: "#8B5CF6",
        icon: "📚",
        type: "education",
      }),
    ];

    mockRepository.findByCalendarId.mockResolvedValue(categories);

    const result = await useCase.execute("cal-123");

    expect(result).toHaveLength(3);
    expect(result[0].type).toBe("social");
    expect(result[1].type).toBe("work");
    expect(result[2].type).toBe("education");
  });
});
