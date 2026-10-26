/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { createContext, useContext } from "react";
import type {
  IIssueDisplayFilterOptions,
  IIssueDisplayProperties,
  IProjectUserPropertiesResponse,
  TWorkItemFilterExpression,
} from "@plane/types";
import type { ProjectUserPropertiesPatch } from "@/queries/project-user-properties";
import type { TProjectIssueRequestInput } from "@/helpers/issue-list/request";

export interface IProjectIssuePreferencesContext {
  workspaceSlug: string;
  projectId: string;
  properties: IProjectUserPropertiesResponse | undefined;
  issueFilters:
    | {
        richFilters: TWorkItemFilterExpression;
        displayFilters: IIssueDisplayFilterOptions;
        displayProperties: IIssueDisplayProperties;
      }
    | undefined;
  requestInput: TProjectIssueRequestInput | undefined;
  isPending: boolean;
  isError: boolean;
  refetch: () => Promise<unknown>;
  save: (patch: ProjectUserPropertiesPatch) => Promise<IProjectUserPropertiesResponse>;
}

export const ProjectIssuePreferencesContext = createContext<IProjectIssuePreferencesContext | undefined>(undefined);

export const useProjectIssuePreferencesContext = () => useContext(ProjectIssuePreferencesContext);
