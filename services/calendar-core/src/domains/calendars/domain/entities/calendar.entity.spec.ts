import { describe, it, expect, beforeEach, vi } from "vitest";
import { Calendar } from "./calendar.entity";

describe("Calendar Entity", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2024-01-15T10:00:00Z"));
  });

  describe("create", () => {
    it("should create a calendar with required fields", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.userId).toBe("user-123");
      expect(calendar.name).toBe("Personal");
      expect(calendar.color).toBe("#FF5733");
      expect(calendar.type).toBe("personal");
      expect(calendar.isActive).toBe(true);
    });

    it("should generate id when not provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Work",
        color: "#0077B5",
        type: "professional",
      });

      expect(calendar.id).toBeDefined();
      expect(typeof calendar.id).toBe("string");
      expect(calendar.id.length).toBeGreaterThan(0);
    });

    it("should use provided id when given", () => {
      const calendar = Calendar.create({
        id: "cal-custom-123",
        userId: "user-123",
        name: "Work",
        color: "#0077B5",
        type: "professional",
      });

      expect(calendar.id).toBe("cal-custom-123");
    });

    it("should set email to null when not provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.email).toBeNull();
    });

    it("should set email when provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Work",
        color: "#0077B5",
        type: "professional",
        email: "bruno@wbdigitalsolutions.com",
      });

      expect(calendar.email).toBe("bruno@wbdigitalsolutions.com");
    });

    it("should set googleCalendarId to null when not provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.googleCalendarId).toBeNull();
    });

    it("should set googleCalendarId when provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Work",
        color: "#0077B5",
        type: "professional",
        googleCalendarId: "google-cal-123",
      });

      expect(calendar.googleCalendarId).toBe("google-cal-123");
    });

    it("should set financeProfileId to null when not provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.financeProfileId).toBeNull();
    });

    it("should set financeProfileId when provided", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Work",
        color: "#0077B5",
        type: "professional",
        financeProfileId: "finance-123",
      });

      expect(calendar.financeProfileId).toBe("finance-123");
    });

    it("should set isActive to true by default", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.isActive).toBe(true);
    });

    it("should allow setting isActive to false", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Archived",
        color: "#808080",
        type: "personal",
        isActive: false,
      });

      expect(calendar.isActive).toBe(false);
    });

    it("should set createdAt and updatedAt dates", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "Personal",
        color: "#FF5733",
        type: "personal",
      });

      expect(calendar.createdAt).toEqual(new Date("2024-01-15T10:00:00Z"));
      expect(calendar.updatedAt).toEqual(new Date("2024-01-15T10:00:00Z"));
    });

    it("should use provided dates when given", () => {
      const customCreated = new Date("2023-06-01T00:00:00Z");
      const customUpdated = new Date("2023-12-01T00:00:00Z");

      const calendar = Calendar.create({
        userId: "user-123",
        name: "Old Calendar",
        color: "#FF5733",
        type: "personal",
        createdAt: customCreated,
        updatedAt: customUpdated,
      });

      expect(calendar.createdAt).toEqual(customCreated);
      expect(calendar.updatedAt).toEqual(customUpdated);
    });

    it("should handle professional calendar type", () => {
      const calendar = Calendar.create({
        userId: "user-123",
        name: "WB Digital Solutions",
        color: "#0077B5",
        type: "professional",
        email: "bruno@wbdigitalsolutions.com",
      });

      expect(calendar.type).toBe("professional");
      expect(calendar.name).toBe("WB Digital Solutions");
    });
  });
});
