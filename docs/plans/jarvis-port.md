# JARVIS port — personal "life OS" on top of calendar

Source: a single-file prototype ("JARVIS", ~1,800 lines of HTML/JS, data in
localStorage). This plan ports its features into the existing services instead of
copying the file: nothing it does is new data the project can't own, and most of it
already has a natural home.

Single-user, like the rest of the project.

## Feature inventory → where it lives

| JARVIS screen | What it does | Home | Already exists |
|---|---|---|---|
| Today | Progress rings (nutrition, habits, tasks, sleep, training), next meal, water, supplements, mood + daily question | calendar-frontend `/today` (aggregates core + health) | Day view sections, `/dashboard` |
| Habits | Build habits (set days / N× per week / daily amount), quit habits (clean days, slips + triggers), streaks, best streak, 30-day rate, 16-week heatmap, pause, time-of-day groups, ideas | calendar-core `events` + calendar-frontend `/habits` | `HABIT` events, `EventCompletion`, flexible weekly habits, streak stats |
| Planner | Mon–Sun tasks, a reward per day | calendar-core (`TODO` + `daily_plans`) | `TODO` events with `dueDate` |
| Training | Templates + rotation (PPL/UL), set logging, rest timer, PRs / estimated 1RM, history, consistency | calendar-health + health-frontend | `workouts`, `exercise_sets` (not wired), `personal_records` (no handlers) |
| Nutrition | Food DB with macros, meal plan with alternative options, "mark eaten", water, shopping list, adherence | calendar-health (new `nutrition` tables) | — |
| Supplements | Daily checklist + streak | calendar-health | — |
| Body | Weight with 7-day average, measurements, trend | calendar-health `body_measurements` | weight + body fat only |
| Sleep & energy | Bed/wake, quality, wake-ups, factors, morning energy/mood | calendar-health (new `sleep_entries`) | — |
| Goals | Dreams (no deadline), quarterly goals per year, motto — personal / professional | calendar-core (new `goals` domain) | unused `MonthlyGoal` |
| Wheel of Life | 0–10 score per life area, monthly, radar vs last month | calendar-core (`goals` domain) | — |
| Weekly review | Week stats + LLM coaching note with 3 focus points | agents (new graph) + calendar-frontend | unused `Report` model |
| Trophies | Tiered achievements computed from logged data; only unlock dates stored | calendar-core (reads health via client) | — |

### Cross-service rule

Habits can tick themselves off from other data ("auto-check" links: training
logged, water goal, sleep goal, calories within 5%, protein ≥ 95%, all supplements).
calendar-core resolves those by calling calendar-health over the Docker network
(`http://calendar-health:3336`), never by reading the `health` schema directly.

## Phases

Each phase is shippable on its own and tracked as one issue in WB Calendar.

1. **Habits v2** — quit habits with slips/triggers, amount habits (value per
   check-in), time of day, pause, best streak, 30-day rate, 16-week heatmap, idea
   chips.
2. **Today + weekly planner + journal** — `/today` page with rings, daily reward,
   mood / daily question / one word.
3. **Training v2** — wire `exercise_sets`, templates + rotation, rest timer, PRs and
   estimated 1RM.
4. **Nutrition + water + supplements** — food DB, meal plan with alternatives,
   eaten log, water, shopping list, supplements checklist.
5. **Sleep & body** — sleep log + trends, weight 7-day average and measurements.
6. **Habit auto-check** — calendar-core ↔ calendar-health client, linked habits.
7. **Goals + Wheel of Life.**
8. **Weekly review with AI** — agents graph reading week stats, stored per week.
9. **Trophies.**

Phase 1 has no cross-service dependency and starts first; phases 3–5 are
independent of each other; 6 needs 1 + 3–5; 8 and 9 need everything they summarize.
