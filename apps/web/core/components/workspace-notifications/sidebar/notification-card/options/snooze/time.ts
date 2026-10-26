/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export type SnoozeTime = {
  date: Date | undefined;
  time: string | undefined;
  period: "AM" | "PM";
};

export function snoozeDate({ date, time, period }: SnoozeTime): Date | undefined {
  if (!date || !time || !/^(0?[1-9]|1[0-2]):[0-5][0-9]$/.test(time)) return undefined;
  const [hours, minutes] = time.split(":").map(Number);
  const result = new Date(date);
  result.setHours((hours % 12) + (period === "PM" ? 12 : 0), minutes, 0, 0);
  return Number.isNaN(result.getTime()) ? undefined : result;
}
