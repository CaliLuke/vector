/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

export const GANTT_TIMELINE_TYPE = {
  ISSUE: "ISSUE",
  MODULE: "MODULE",
  PROJECT: "PROJECT",
  GROUPED: "GROUPED",
} as const;

export type TTimelineTypeCore = (typeof GANTT_TIMELINE_TYPE)[keyof typeof GANTT_TIMELINE_TYPE];
export type TTimelineType = TTimelineTypeCore;
