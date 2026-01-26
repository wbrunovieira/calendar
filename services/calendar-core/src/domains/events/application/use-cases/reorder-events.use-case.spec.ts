import { describe, it, expect, beforeEach, vi } from "vitest";
import { ReorderEventsUseCase } from "./reorder-events.use-case";

const mockEventRepository = {
  findAll: vi.fn(),
  findById: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  delete: vi.fn(),
  updateDisplayOrder: vi.fn(),
};

describe("ReorderEventsUseCase", () => {
  let useCase: ReorderEventsUseCase;

  beforeEach(() => {
    vi.clearAllMocks();
    useCase = new ReorderEventsUseCase(mockEventRepository as any);
  });

  it("should update display order for multiple events", async () => {
    const orderedIds = ["event-1", "event-2", "event-3"];
    mockEventRepository.updateDisplayOrder.mockResolvedValue(undefined);

    await useCase.execute({ orderedIds });

    expect(mockEventRepository.updateDisplayOrder).toHaveBeenCalledWith([
      { id: "event-1", displayOrder: 0 },
      { id: "event-2", displayOrder: 1 },
      { id: "event-3", displayOrder: 2 },
    ]);
  });

  it("should handle single event", async () => {
    const orderedIds = ["event-1"];
    mockEventRepository.updateDisplayOrder.mockResolvedValue(undefined);

    await useCase.execute({ orderedIds });

    expect(mockEventRepository.updateDisplayOrder).toHaveBeenCalledWith([
      { id: "event-1", displayOrder: 0 },
    ]);
  });

  it("should handle empty array", async () => {
    const orderedIds: string[] = [];
    mockEventRepository.updateDisplayOrder.mockResolvedValue(undefined);

    await useCase.execute({ orderedIds });

    expect(mockEventRepository.updateDisplayOrder).toHaveBeenCalledWith([]);
  });

  it("should propagate repository errors", async () => {
    const orderedIds = ["event-1", "event-2"];
    const error = new Error("Database error");
    mockEventRepository.updateDisplayOrder.mockRejectedValue(error);

    await expect(useCase.execute({ orderedIds })).rejects.toThrow("Database error");
  });

  it("should preserve order from input array", async () => {
    const orderedIds = ["event-3", "event-1", "event-2"];
    mockEventRepository.updateDisplayOrder.mockResolvedValue(undefined);

    await useCase.execute({ orderedIds });

    expect(mockEventRepository.updateDisplayOrder).toHaveBeenCalledWith([
      { id: "event-3", displayOrder: 0 },
      { id: "event-1", displayOrder: 1 },
      { id: "event-2", displayOrder: 2 },
    ]);
  });

  it("should handle large number of events", async () => {
    const orderedIds = Array.from({ length: 100 }, (_, i) => `event-${i}`);
    mockEventRepository.updateDisplayOrder.mockResolvedValue(undefined);

    await useCase.execute({ orderedIds });

    const expectedCalls = orderedIds.map((id, index) => ({
      id,
      displayOrder: index,
    }));
    expect(mockEventRepository.updateDisplayOrder).toHaveBeenCalledWith(expectedCalls);
  });
});
