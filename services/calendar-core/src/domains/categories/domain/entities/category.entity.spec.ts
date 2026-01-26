import { describe, it, expect, beforeEach, vi } from "vitest";
import { Category } from "./category.entity";

describe("Category Entity", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-15T10:00:00Z"));
  });

  describe("create", () => {
    it("should create a category with required fields", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "LinkedIn",
        color: "#0077B5",
      });

      expect(category.calendarId).toBe("cal-123");
      expect(category.name).toBe("LinkedIn");
      expect(category.color).toBe("#0077B5");
      expect(category.isActive).toBe(true);
    });

    it("should set icon when provided", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Health",
        color: "#22C55E",
        icon: "💪",
      });

      expect(category.icon).toBe("💪");
    });

    it("should set type when provided", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Work Tasks",
        color: "#0077B5",
        type: "work",
      });

      expect(category.type).toBe("work");
    });

    it("should set createdAt and updatedAt dates", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Personal",
        color: "#FF5733",
      });

      expect(category.createdAt).toEqual(new Date("2024-01-15T10:00:00Z"));
      expect(category.updatedAt).toEqual(new Date("2024-01-15T10:00:00Z"));
    });

    it("should be active by default", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Finance",
        color: "#10B981",
      });

      expect(category.isActive).toBe(true);
    });
  });

  describe("update", () => {
    it("should update name", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Old Name",
        color: "#FF5733",
      });

      vi.setSystemTime(new Date("2024-01-16T10:00:00Z"));
      category.update({ name: "New Name" });

      expect(category.name).toBe("New Name");
      expect(category.updatedAt).toEqual(new Date("2024-01-16T10:00:00Z"));
    });

    it("should update color", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.update({ color: "#0077B5" });

      expect(category.color).toBe("#0077B5");
    });

    it("should update icon", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.update({ icon: "🎯" });

      expect(category.icon).toBe("🎯");
    });

    it("should update type", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.update({ type: "leisure" });

      expect(category.type).toBe("leisure");
    });

    it("should update multiple fields at once", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Old",
        color: "#FF5733",
      });

      category.update({
        name: "New",
        color: "#0077B5",
        icon: "📱",
        type: "social",
      });

      expect(category.name).toBe("New");
      expect(category.color).toBe("#0077B5");
      expect(category.icon).toBe("📱");
      expect(category.type).toBe("social");
    });

    it("should update updatedAt timestamp", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      const originalUpdatedAt = category.updatedAt;
      vi.setSystemTime(new Date("2024-01-20T10:00:00Z"));

      category.update({ name: "Updated" });

      expect(category.updatedAt).not.toEqual(originalUpdatedAt);
      expect(category.updatedAt).toEqual(new Date("2024-01-20T10:00:00Z"));
    });
  });

  describe("deactivate", () => {
    it("should set isActive to false", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.deactivate();

      expect(category.isActive).toBe(false);
    });

    it("should update updatedAt timestamp", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      vi.setSystemTime(new Date("2024-01-20T10:00:00Z"));
      category.deactivate();

      expect(category.updatedAt).toEqual(new Date("2024-01-20T10:00:00Z"));
    });
  });

  describe("activate", () => {
    it("should set isActive to true", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.deactivate();
      expect(category.isActive).toBe(false);

      category.activate();
      expect(category.isActive).toBe(true);
    });

    it("should update updatedAt timestamp", () => {
      const category = Category.create({
        calendarId: "cal-123",
        name: "Test",
        color: "#FF5733",
      });

      category.deactivate();
      vi.setSystemTime(new Date("2024-01-25T10:00:00Z"));
      category.activate();

      expect(category.updatedAt).toEqual(new Date("2024-01-25T10:00:00Z"));
    });
  });
});
