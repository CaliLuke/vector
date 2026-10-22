/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { action, makeObservable, runInAction } from "mobx";
// base class
import type {
  TIssue,
  TLoader,
  ViewFlags,
  IssuePaginationOptions,
  TIssuesResponse,
  TBulkOperationsPayload,
} from "@plane/types";
// helpers
import { getDistributionPathsPostUpdate } from "@plane/utils";
import type { IBaseIssuesStore } from "../helpers/base-issues.store";
import { BaseIssuesStore } from "../helpers/base-issues.store";
//
import type { IIssueRootStore } from "../root.store";
import type { IModuleIssuesFilter } from "./filter.store";

export interface IModuleIssues extends IBaseIssuesStore {
  viewFlags: ViewFlags;
  // actions
  getIssueIds: (groupId?: string, subGroupId?: string) => string[] | undefined;
  fetchIssues: (
    workspaceSlug: string,
    projectId: string,
    loadType: TLoader,
    options: IssuePaginationOptions,
    moduleId: string
  ) => Promise<TIssuesResponse | undefined>;
  fetchIssuesWithExistingPagination: (
    workspaceSlug: string,
    projectId: string,
    loadType: TLoader,
    moduleId: string
  ) => Promise<TIssuesResponse | undefined>;
  fetchNextIssues: (
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    groupId?: string,
    subGroupId?: string
  ) => Promise<TIssuesResponse | undefined>;

  createIssue: (workspaceSlug: string, projectId: string, data: Partial<TIssue>, moduleId: string) => Promise<TIssue>;
  updateIssue: (workspaceSlug: string, projectId: string, issueId: string, data: Partial<TIssue>) => Promise<void>;
  archiveIssue: (workspaceSlug: string, projectId: string, issueId: string) => Promise<void>;
  quickAddIssue: (
    workspaceSlug: string,
    projectId: string,
    data: TIssue,
    moduleId: string
  ) => Promise<TIssue | undefined>;
  removeBulkIssues: (workspaceSlug: string, projectId: string, issueIds: string[]) => Promise<void>;
  archiveBulkIssues: (workspaceSlug: string, projectId: string, issueIds: string[]) => Promise<void>;
  bulkUpdateProperties: (workspaceSlug: string, projectId: string, data: TBulkOperationsPayload) => Promise<void>;
}

export class ModuleIssues extends BaseIssuesStore implements IModuleIssues {
  private fetchRequestId = 0;

  viewFlags = {
    enableQuickAdd: true,
    enableIssueCreation: true,
    enableInlineEditing: true,
  };
  // filter store
  issueFilterStore: IModuleIssuesFilter;

  constructor(_rootStore: IIssueRootStore, issueFilterStore: IModuleIssuesFilter) {
    super(_rootStore, issueFilterStore);

    makeObservable(this, {
      // action
      fetchIssues: action,
      fetchNextIssues: action,
      fetchIssuesWithExistingPagination: action,

      quickAddIssue: action,
    });
    // filter store
    this.issueFilterStore = issueFilterStore;
  }

  /**
   * Fetches the module details
   * @param workspaceSlug
   * @param projectId
   * @param id is the module Id
   */
  fetchParentStats = (workspaceSlug: string, projectId?: string, id?: string) => {
    const moduleId = id ?? this.moduleId;
    if (projectId && moduleId) {
      this.rootIssueStore.rootStore.module.fetchModuleDetails(workspaceSlug, projectId, moduleId);
    }
  };

  /**
   * Update Parent stats before fetching from server
   * @param prevIssueState
   * @param nextIssueState
   * @param id
   */
  updateParentStats = (prevIssueState?: TIssue, nextIssueState?: TIssue, id?: string) => {
    try {
      // get distribution updates
      const distributionUpdates = getDistributionPathsPostUpdate(
        prevIssueState,
        nextIssueState,
        this.rootIssueStore.rootStore.state.stateMap,
        this.rootIssueStore.rootStore.projectEstimate?.currentActiveEstimate?.estimatePointById
      );

      const moduleId = id ?? this.moduleId;

      if (moduleId) {
        this.rootIssueStore.rootStore.module.updateModuleDistribution(distributionUpdates, moduleId);
      }
    } catch (_e) {
      console.warn("could not update module statistics");
    }
  };

  /**
   * This method is called to fetch the first issues of pagination
   * @param workspaceSlug
   * @param projectId
   * @param loadType
   * @param options
   * @param moduleId
   * @returns
   */
  fetchIssues = async (
    workspaceSlug: string,
    projectId: string,
    loadType: TLoader,
    options: IssuePaginationOptions,
    moduleId: string,
    isExistingPaginationOptions: boolean = false
  ) => {
    const requestId = ++this.fetchRequestId;
    let controller: AbortController | undefined;
    let response: TIssuesResponse;

    try {
      // set loader and clear store
      runInAction(() => {
        this.setLoader(loadType);
        this.clear(!isExistingPaginationOptions); // clear while fetching from server.
      });
      controller = this.controller;

      // get params from pagination options
      const params = this.issueFilterStore?.getFilterParams(options, moduleId, undefined, undefined, undefined);
      // call the fetch issues API with the params
      response = await this.issueService.getIssues(workspaceSlug, projectId, params, {
        signal: controller.signal,
      });
    } catch (error) {
      // A request replaced by a newer fetch or clear() no longer owns the loader.
      if (requestId !== this.fetchRequestId || (controller && controller !== this.controller)) return undefined;

      this.setLoader(undefined);
      throw error;
    }

    // Ignore responses that complete after a newer fetch or clear() took ownership.
    if (requestId !== this.fetchRequestId || controller !== this.controller) return undefined;

    try {
      // Process the response only after confirming this request still owns the store.
      // onfetchIssues may clear and replace the controller as part of processing.
      this.onfetchIssues(response, options, workspaceSlug, projectId, moduleId, !isExistingPaginationOptions);
      return response;
    } catch (error) {
      if (requestId === this.fetchRequestId) this.setLoader(undefined);
      throw error;
    }
  };

  /**
   * This method is called subsequent pages of pagination
   * if groupId/subgroupId is provided, only that specific group's next page is fetched
   * else all the groups' next page is fetched
   * @param workspaceSlug
   * @param projectId
   * @param moduleId
   * @param groupId
   * @param subGroupId
   * @returns
   */
  fetchNextIssues = async (
    workspaceSlug: string,
    projectId: string,
    moduleId: string,
    groupId?: string,
    subGroupId?: string
  ) => {
    const cursorObject = this.getPaginationData(groupId, subGroupId);
    // if there are no pagination options and the next page results do not exist the return
    if (!this.paginationOptions || (cursorObject && !cursorObject?.nextPageResults)) return;
    try {
      // set Loader
      this.setLoader("pagination", groupId, subGroupId);

      // get params from stored pagination options
      const params = this.issueFilterStore?.getFilterParams(
        this.paginationOptions,
        moduleId,
        this.getNextCursor(groupId, subGroupId),
        groupId,
        subGroupId
      );
      // call the fetch issues API with the params for next page in issues
      const response = await this.issueService.getIssues(workspaceSlug, projectId, params);

      // after the next page of issues are fetched, call the base method to process the response
      this.onfetchNexIssues(response, groupId, subGroupId);
      return response;
    } catch (error) {
      // set Loader as undefined if errored out
      this.setLoader(undefined, groupId, subGroupId);
      throw error;
    }
  };

  /**
   * This Method exists to fetch the first page of the issues with the existing stored pagination
   * This is useful for refetching when filters, groupBy, orderBy etc changes
   * @param workspaceSlug
   * @param projectId
   * @param loadType
   * @param moduleId
   * @returns
   */
  fetchIssuesWithExistingPagination = async (
    workspaceSlug: string,
    projectId: string,
    loadType: TLoader,
    moduleId: string
  ) => {
    if (!this.paginationOptions) return;
    return await this.fetchIssues(workspaceSlug, projectId, loadType, this.paginationOptions, moduleId, true);
  };

  /**
   * Override inherited create issue, to also add issue to module
   * @param workspaceSlug
   * @param projectId
   * @param data
   * @param moduleId
   * @returns
   */
  override createIssue = async (workspaceSlug: string, projectId: string, data: Partial<TIssue>, moduleId: string) => {
    const response = await super.createIssue(workspaceSlug, projectId, data, moduleId, false);
    const moduleIds = data.module_ids && data.module_ids.length > 1 ? data.module_ids : [moduleId];
    await this.addModulesToIssue(workspaceSlug, projectId, response.id, moduleIds);

    return response;
  };

  /**
   * This Method overrides the base quickAdd issue
   * @param workspaceSlug
   * @param projectId
   * @param data
   * @param moduleId
   * @returns
   */
  quickAddIssue = async (workspaceSlug: string, projectId: string, data: TIssue, moduleId: string) => {
    // add temporary issue to store list
    this.addIssue(data);

    // call overridden create issue
    const response = await this.createIssue(workspaceSlug, projectId, data, moduleId);

    // remove temp Issue from store list
    runInAction(() => {
      this.removeIssueFromList(data.id);
      this.rootIssueStore.issues.removeIssue(data.id);
    });

    const currentCycleId = data.cycle_id !== "" && data.cycle_id === "None" ? undefined : data.cycle_id;

    if (currentCycleId) {
      await this.addCycleToIssue(workspaceSlug, projectId, currentCycleId, response.id);
    }

    return response;
  };

  // Using aliased names as they cannot be overridden in other stores
  archiveBulkIssues = this.bulkArchiveIssues;
  updateIssue = this.issueUpdate;
  archiveIssue = this.issueArchive;
}
