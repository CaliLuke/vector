import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { NOTIFICATION_TABS, NOTIFICATION_SNOOZE_OPTIONS } = await import("../../packages/constants/src/notification.ts");

void test("notification tabs use independent unread counters and default missing counts to zero", () => {
  const [all, mentions] = NOTIFICATION_TABS;
  const unread = { total_unread_notifications_count: 7, mention_unread_notifications_count: 2 };
  assert.equal(all.count(unread), 7);
  assert.equal(mentions.count(unread), 2);
  assert.equal(all.count(undefined), 0);
  assert.equal(mentions.count({}), 0);
  assert.equal(all.count({ total_unread_notifications_count: 0 }), 0);
  assert.equal(mentions.count({ mention_unread_notifications_count: 0 }), 0);
});

void test("snooze presets use elapsed days across a daylight-saving boundary and leave custom dates unset", (t) => {
  t.mock.timers.enable({ apis: ["Date"], now: Date.parse("2026-10-31T20:00:00Z") });
  const [day, threeDays, fiveDays, week, twoWeeks, custom] = NOTIFICATION_SNOOZE_OPTIONS;
  assert.equal(day.value().toISOString(), "2026-11-01T20:00:00.000Z");
  assert.equal(threeDays.value().toISOString(), "2026-11-03T20:00:00.000Z");
  assert.equal(fiveDays.value().toISOString(), "2026-11-05T20:00:00.000Z");
  assert.equal(week.value().toISOString(), "2026-11-07T20:00:00.000Z");
  assert.equal(twoWeeks.value().toISOString(), "2026-11-14T20:00:00.000Z");
  assert.equal(custom.value, undefined);
});
