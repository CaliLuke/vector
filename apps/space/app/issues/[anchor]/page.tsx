/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { publishSettingsOptions } from "@/queries/publishing";

import { useParams, useSearchParams } from "react-router";
// components
import { IssuesLayoutsRoot } from "@/components/issues/issue-layouts";
// hooks

const IssuesPage = function IssuesPage() {
  // params
  const params = useParams<{ anchor: string }>();
  const { anchor } = params;
  const [searchParams] = useSearchParams();
  const peekId = searchParams.get("peekId") || undefined;
  const { data: publishSettings } = useQuery(publishSettingsOptions(anchor));

  if (!publishSettings || !anchor) return null;

  return <IssuesLayoutsRoot peekId={peekId} anchor={anchor} />;
};

export default IssuesPage;
