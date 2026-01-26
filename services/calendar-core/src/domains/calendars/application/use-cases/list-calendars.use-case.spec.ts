import { describe, it, expect, beforeEach, vi } from "vitest";
import { ListCalendarsUseCase } from "./list-calendars.use-case";
import { Calendar } from "../../domain/entities/calendar.entity";

const mockRepository = {
  findAll: vi.fn(),
  findById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
};

describe("ListCalendarsUseCase", () => {
  let useCase: ListCalendarsUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new ListCalendarsUseCase(mockRepository as any);
  });

  it("should return all calendars when no userId is provided", async () => {
    const calendars = [
      Calendar.create({
        id: "cal-1",
        userId: "user-1",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      }),
      Calendar.create({
        id: "cal-2",
        userId: "user-2",
        name: "Work",
        color: "#0077B5",
        type: "professional",
      }),
    ];

    mockRepository.findAll.mockResolvedValue(calendars);

    const result = await useCase.execute();

    expect(result).toEqual(calendars);
    expect(mockRepository.findAll).toHaveBeenCalledWith(undefined);
  });

  it("should return calendars filtered by userId", async () => {
    const calendars = [
      Calendar.create({
        id: "cal-1",
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      }),
    ];

    mockRepository.findAll.mockResolvedValue(calendars);

    const result = await useCase.execute("user-123");

    expect(result).toEqual(calendars);
    expect(mockRepository.findAll).toHaveBeenCalledWith("user-123");
  });

  it("should return empty array when no calendars exist", async () => {
    mockRepository.findAll.mockResolvedValue([]);

    const result = await useCase.execute();

    expect(result).toEqual([]);
    expect(mockRepository.findAll).toHaveBeenCalled();
  });

  it("should return multiple calendars for same user", async () => {
    const calendars = [
      Calendar.create({
        id: "cal-1",
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      }),
      Calendar.create({
        id: "cal-2",
        userId: "user-123",
        name: "WB Digital Solutions",
        color: "#0077B5",
        type: "professional",
      }),
    ];

    mockRepository.findAll.mockResolvedValue(calendars);

    const result = await useCase.execute("user-123");

    expect(result).toHaveLength(2);
    expect(result[0].name).toBe("Personal");
    expect(result[1].name).toBe("WB Digital Solutions");
  });

  it("should propagate repository errors", async () => {
    const error = new Error("Database connection failed");
    mockRepository.findAll.mockRejectedValue(error);

    await expect(useCase.execute()).rejects.toThrow("Database connection failed");
  });

  it("should return calendars with all properties populated", async () => {
    const calendar = Calendar.create({
      id: "cal-1",
      userId: "user-123",
      name: "Work",
      email: "bruno@wbdigitalsolutions.com",
      color: "#0077B5",
      type: "professional",
      googleCalendarId: "google-123",
      financeProfileId: "finance-456",
    });

    mockRepository.findAll.mockResolvedValue([calendar]);

    const result = await useCase.execute("user-123");

    expect(result[0].email).toBe("bruno@wbdigitalsolutions.com");
    expect(result[0].googleCalendarId).toBe("google-123");
    expect(result[0].financeProfileId).toBe("finance-456");
  });
});
