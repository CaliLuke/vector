/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { ProjectIssueLayout } from "../../project-issue-layout";
// local imports
import { BaseCalendarRoot } from "../base-calendar-root";

export function CalendarLayout() {
  return <ProjectIssueLayout Layout={BaseCalendarRoot} />;
}
