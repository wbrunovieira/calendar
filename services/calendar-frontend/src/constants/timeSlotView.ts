/**
 * Constants for TimeSlotView component
 */

export const PIXELS_PER_HOUR = 96;
export const START_HOUR = 6; // 6am
export const END_HOUR = 22; // 10pm
export const TOTAL_HOURS = END_HOUR - START_HOUR + 1; // 17 hours (6am-10pm)
export const MIN_EVENT_HEIGHT = 48; // 30 minutes minimum

export const HOURS_ARRAY = Array.from({ length: TOTAL_HOURS }, (_, i) => i + START_HOUR);
