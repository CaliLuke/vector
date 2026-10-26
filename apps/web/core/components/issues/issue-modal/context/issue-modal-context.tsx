/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { createContext } from "react";
import type { Dispatch, SetStateAction } from "react";
import type { ISearchIssueResponse } from "@plane/types";
export type TIssueModalContext = {
  allowedProjectIds: string[];
  defaultProjectId: string | undefined;
  selectedParentIssue: ISearchIssueResponse | null;
  setSelectedParentIssue: Dispatch<SetStateAction<ISearchIssueResponse | null>>;
};
export const IssueModalContext = createContext<TIssueModalContext | undefined>(undefined);
