import { describe, it, expect, beforeEach, vi } from "vitest";
import { CreateCategoryUseCase } from "./create-category.use-case";
import { Category } from "../../domain/entities/category.entity";

const mockRepository = {
  findByCalendarId: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};

describe("CreateCategoryUseCase", () => {
  let useCase: CreateCategoryUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-15T10:00:00Z"));
    useCase = new CreateCategoryUseCase(mockRepository as any);
  });

  it("should create a category with required fields", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "LinkedIn",
      color: "#0077B5",
    };

    const createdCategory = Category.create(dto);
    mockRepository.create.mockResolvedValue(createdCategory);

    const result = await useCase.execute(dto);

    expect(mockRepository.create).toHaveBeenCalled();
    expect(result.name).toBe("LinkedIn");
    expect(result.color).toBe("#0077B5");
  });

  it("should create a category with icon", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "Health",
      color: "#22C55E",
      icon: "💪",
    };

    const createdCategory = Category.create(dto);
    mockRepository.create.mockResolvedValue(createdCategory);

    const result = await useCase.execute(dto);

    expect(result.icon).toBe("💪");
  });

  it("should create a category with type for backward compatibility", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "Work Tasks",
      color: "#0077B5",
      type: "work",
    };

    const createdCategory = Category.create(dto);
    mockRepository.create.mockResolvedValue(createdCategory);

    const result = await useCase.execute(dto);

    expect(result.type).toBe("work");
  });

  it("should pass typeIds to repository when provided", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "Multi-type Category",
      color: "#8B5CF6",
      typeIds: ["type-1", "type-2"],
    };

    const createdCategory = Category.create({
      calendarId: dto.calendarId,
      name: dto.name,
      color: dto.color,
    });
    mockRepository.create.mockResolvedValue(createdCategory);

    await useCase.execute(dto);

    expect(mockRepository.create).toHaveBeenCalledWith(
      expect.any(Category),
      ["type-1", "type-2"]
    );
  });

  it("should pass undefined typeIds when not provided", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "Simple Category",
      color: "#FF5733",
    };

    const createdCategory = Category.create(dto);
    mockRepository.create.mockResolvedValue(createdCategory);

    await useCase.execute(dto);

    expect(mockRepository.create).toHaveBeenCalledWith(
      expect.any(Category),
      undefined
    );
  });

  it("should propagate repository errors", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "Test",
      color: "#FF5733",
    };

    const error = new Error("Database constraint violation");
    mockRepository.create.mockRejectedValue(error);

    await expect(useCase.execute(dto)).rejects.toThrow("Database constraint violation");
  });

  it("should create category with all fields", async () => {
    const dto = {
      calendarId: "wb-digital",
      name: "Client Projects",
      color: "#0077B5",
      icon: "🎯",
      type: "work",
      typeIds: ["professional", "projects"],
    };

    const createdCategory = Category.create({
      calendarId: dto.calendarId,
      name: dto.name,
      color: dto.color,
      icon: dto.icon,
      type: dto.type,
    });
    mockRepository.create.mockResolvedValue(createdCategory);

    const result = await useCase.execute(dto);

    expect(result.calendarId).toBe("wb-digital");
    expect(result.name).toBe("Client Projects");
    expect(result.icon).toBe("🎯");
  });

  it("should set isActive to true for new categories", async () => {
    const dto = {
      calendarId: "cal-123",
      name: "New Category",
      color: "#FF5733",
    };

    const createdCategory = Category.create(dto);
    mockRepository.create.mockResolvedValue(createdCategory);

    const result = await useCase.execute(dto);

    expect(result.isActive).toBe(true);
  });
});
