/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useParams } from "@tanstack/react-router";
import { ProjectIssueLayout } from "../../project-issue-layout";
// local imports
import { BaseKanBanRoot } from "../base-kanban-root";

export function ProfileIssuesKanBanLayout() {
  const { profileViewId } = useParams({ strict: false });
  return <ProjectIssueLayout Layout={BaseKanBanRoot} viewId={profileViewId?.toString()} />;
}
