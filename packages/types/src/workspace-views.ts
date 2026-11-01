/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import type { IWorkspaceViewProps } from "./view-props";
import type { IProjectView } from "./views";

export interface IWorkspaceView extends Omit<IProjectView, "logo_props" | "anchor" | "query" | "query_data"> {
  query: unknown;
  query_data: IWorkspaceViewProps;
  workspace_detail?: {
    id: string;
    name: string;
    slug: string;
  };
}

export const STATIC_VIEW_TYPES = ["all-issues", "assigned", "created", "subscribed"];

export type TStaticViewTypes = (typeof STATIC_VIEW_TYPES)[number];
