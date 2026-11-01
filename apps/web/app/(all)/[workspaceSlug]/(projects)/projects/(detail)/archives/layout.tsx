/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createFileRoute } from "@tanstack/react-router";
import { ProjectListLayout } from "@/components/projects/list-layout";

export const Route = createFileRoute("/_authenticated/$workspaceSlug/_projects/projects/archives")({
  component: ProjectListLayout,
});
