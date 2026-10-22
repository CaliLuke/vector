/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { isEmpty } from "lodash-es";
import { autorun, makeObservable, observable, runInAction } from "mobx";
// types
import type { ICycle, IIssueLabel, IModule, IProject, IState, IUserLite, TIssueServiceType } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
// plane web store
import type { IIssueDetail } from "@/store/issue/issue-details/root.store";
import { IssueDetail } from "@/store/issue/issue-details/root.store";
import type { RootStore } from "@/store/root.store";
import type { IWorkspaceMembership } from "@/store/member/workspace/workspace-member.store";
// issues data store
import type { IArchivedIssuesFilter, IArchivedIssues } from "./archived";
import { ArchivedIssuesFilter, ArchivedIssues } from "./archived";
import type { ICycleIssuesFilter, ICycleIssues } from "./cycle";
import { CycleIssuesFilter, CycleIssues } from "./cycle";
import type { IIssueStore } from "./issue.store";
import { IssueStore } from "./issue.store";
import type { ICalendarStore } from "./issue_calendar_view.store";
import { CalendarStore } from "./issue_calendar_view.store";
import type { IIssueKanBanViewStore } from "./issue_kanban_view.store";
import { IssueKanBanViewStore } from "./issue_kanban_view.store";
import type { IModuleIssuesFilter, IModuleIssues } from "./module";
import { ModuleIssuesFilter, ModuleIssues } from "./module";
import type { IProfileIssuesFilter, IProfileIssues } from "./profile";
import { ProfileIssuesFilter, ProfileIssues } from "./profile";
import type { IProjectIssuesFilter, IProjectIssues } from "./project";
import { ProjectIssuesFilter, ProjectIssues } from "./project";
import type { IProjectViewIssuesFilter, IProjectViewIssues } from "./project-views";
import { ProjectViewIssuesFilter, ProjectViewIssues } from "./project-views";
import type { IWorkspaceIssuesFilter } from "./workspace";
import { WorkspaceIssuesFilter } from "./workspace";
import type { IWorkspaceIssues } from "./workspace/issue.store";
import { WorkspaceIssues } from "./workspace/issue.store";
import type { IWorkspaceDraftIssues, IWorkspaceDraftIssuesFilter } from "./workspace-draft";
import { WorkspaceDraftIssues, WorkspaceDraftIssuesFilter } from "./workspace-draft";

export interface IIssueRootStore {
  currentUserId: string | undefined;
  workspaceSlug: string | undefined;
  teamspaceId: string | undefined;
  projectId: string | undefined;
  cycleId: string | undefined;
  moduleId: string | undefined;
  viewId: string | undefined;
  globalViewId: string | undefined; // all issues view id
  userId: string | undefined; // user profile detail Id
  stateMap: Record<string, IState> | undefined;
  stateDetails: IState[] | undefined;
  workspaceStateDetails: IState[] | undefined;
  labelMap: Record<string, IIssueLabel> | undefined;
  workSpaceMemberRolesMap: Record<string, IWorkspaceMembership> | undefined;
  memberMap: Record<string, IUserLite> | undefined;
  projectMap: Record<string, IProject> | undefined;
  moduleMap: Record<string, IModule> | undefined;
  cycleMap: Record<string, ICycle> | undefined;

  rootStore: RootStore;
  serviceType: TIssueServiceType;

  issues: IIssueStore;

  issueDetail: IIssueDetail;
  epicDetail: IIssueDetail;

  workspaceIssuesFilter: IWorkspaceIssuesFilter;
  workspaceIssues: IWorkspaceIssues;

  workspaceDraftIssuesFilter: IWorkspaceDraftIssuesFilter;
  workspaceDraftIssues: IWorkspaceDraftIssues;

  profileIssuesFilter: IProfileIssuesFilter;
  profileIssues: IProfileIssues;

  teamIssuesFilter: IProjectIssuesFilter;
  teamIssues: IProjectIssues;

  projectIssuesFilter: IProjectIssuesFilter;
  projectIssues: IProjectIssues;

  cycleIssuesFilter: ICycleIssuesFilter;
  cycleIssues: ICycleIssues;

  moduleIssuesFilter: IModuleIssuesFilter;
  moduleIssues: IModuleIssues;

  teamViewIssuesFilter: IProjectViewIssuesFilter;
  teamViewIssues: IProjectViewIssues;

  teamProjectWorkItemsFilter: IProjectIssuesFilter;
  teamProjectWorkItems: IProjectIssues;

  projectViewIssuesFilter: IProjectViewIssuesFilter;
  projectViewIssues: IProjectViewIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;

  projectEpicsFilter: IProjectIssuesFilter;
  projectEpics: IProjectIssues;
}

export class IssueRootStore implements IIssueRootStore {
  currentUserId: string | undefined = undefined;
  workspaceSlug: string | undefined = undefined;
  teamspaceId: string | undefined = undefined;
  projectId: string | undefined = undefined;
  cycleId: string | undefined = undefined;
  moduleId: string | undefined = undefined;
  viewId: string | undefined = undefined;
  globalViewId: string | undefined = undefined;
  userId: string | undefined = undefined;
  stateMap: Record<string, IState> | undefined = undefined;
  stateDetails: IState[] | undefined = undefined;
  workspaceStateDetails: IState[] | undefined = undefined;
  labelMap: Record<string, IIssueLabel> | undefined = undefined;
  workSpaceMemberRolesMap: Record<string, IWorkspaceMembership> | undefined = undefined;
  memberMap: Record<string, IUserLite> | undefined = undefined;
  projectMap: Record<string, IProject> | undefined = undefined;
  moduleMap: Record<string, IModule> | undefined = undefined;
  cycleMap: Record<string, ICycle> | undefined = undefined;

  rootStore: RootStore;
  serviceType: TIssueServiceType;

  issues: IIssueStore;

  issueDetail: IIssueDetail;
  epicDetail: IIssueDetail;

  workspaceIssuesFilter: IWorkspaceIssuesFilter;
  workspaceIssues: IWorkspaceIssues;

  workspaceDraftIssuesFilter: IWorkspaceDraftIssuesFilter;
  workspaceDraftIssues: IWorkspaceDraftIssues;

  profileIssuesFilter: IProfileIssuesFilter;
  profileIssues: IProfileIssues;

  teamIssuesFilter: IProjectIssuesFilter;
  teamIssues: IProjectIssues;

  projectIssuesFilter: IProjectIssuesFilter;
  projectIssues: IProjectIssues;

  cycleIssuesFilter: ICycleIssuesFilter;
  cycleIssues: ICycleIssues;

  moduleIssuesFilter: IModuleIssuesFilter;
  moduleIssues: IModuleIssues;

  teamViewIssuesFilter: IProjectViewIssuesFilter;
  teamViewIssues: IProjectViewIssues;

  projectViewIssuesFilter: IProjectViewIssuesFilter;
  projectViewIssues: IProjectViewIssues;

  teamProjectWorkItemsFilter: IProjectIssuesFilter;
  teamProjectWorkItems: IProjectIssues;

  archivedIssuesFilter: IArchivedIssuesFilter;
  archivedIssues: IArchivedIssues;

  issueKanBanView: IIssueKanBanViewStore;
  issueCalendarView: ICalendarStore;

  projectEpicsFilter: IProjectIssuesFilter;
  projectEpics: IProjectIssues;

  constructor(rootStore: RootStore, serviceType: TIssueServiceType = EIssueServiceType.ISSUES) {
    makeObservable(this, {
      workspaceSlug: observable.ref,
      teamspaceId: observable.ref,
      projectId: observable.ref,
      cycleId: observable.ref,
      moduleId: observable.ref,
      viewId: observable.ref,
      userId: observable.ref,
      globalViewId: observable.ref,
      stateMap: observable,
      stateDetails: observable,
      workspaceStateDetails: observable,
      labelMap: observable,
      memberMap: observable,
      workSpaceMemberRolesMap: observable,
      projectMap: observable,
      moduleMap: observable,
      cycleMap: observable,
    });

    this.serviceType = serviceType;
    this.rootStore = rootStore;

    autorun(() => {
      const currentUserId = rootStore?.user?.data?.id;
      const workspaceSlug = rootStore.router.workspaceSlug;
      const teamspaceId = rootStore.router.teamspaceId;
      const projectId = rootStore.router.projectId;
      const cycleId = rootStore.router.cycleId;
      const moduleId = rootStore.router.moduleId;
      const viewId = rootStore.router.viewId;
      const globalViewId = rootStore.router.globalViewId;
      const userId = rootStore.router.userId;
      const stateMap = rootStore?.state?.stateMap;
      const stateDetails = rootStore?.state?.projectStates;
      const workspaceStateDetails = rootStore?.state?.workspaceStates;
      const labelMap = rootStore?.label?.labelMap;
      const workspaceMemberMap = rootStore?.memberRoot?.workspace?.workspaceMemberMap;
      const workspaceMemberRolesMap = rootStore?.memberRoot?.workspace?.memberMap || undefined;
      const memberMap = rootStore?.memberRoot?.memberMap || undefined;
      const projectMap = rootStore?.projectRoot?.project?.projectMap;
      const moduleMap = rootStore?.module?.moduleMap;
      const cycleMap = rootStore?.cycle?.cycleMap;

      const hasStateMap = !isEmpty(stateMap);
      const hasStateDetails = !isEmpty(stateDetails);
      const hasWorkspaceStateDetails = !isEmpty(workspaceStateDetails);
      const hasLabelMap = !isEmpty(labelMap);
      const hasWorkspaceMemberMap = !isEmpty(workspaceMemberMap);
      const hasMemberMap = !isEmpty(memberMap);
      const hasProjectMap = !isEmpty(projectMap);
      const hasModuleMap = !isEmpty(moduleMap);
      const hasCycleMap = !isEmpty(cycleMap);

      runInAction(() => {
        if (currentUserId) this.currentUserId = currentUserId;
        this.workspaceSlug = workspaceSlug;
        this.teamspaceId = teamspaceId;
        this.projectId = projectId;
        this.cycleId = cycleId;
        this.moduleId = moduleId;
        this.viewId = viewId;
        this.globalViewId = globalViewId;
        this.userId = userId;
        if (hasStateMap) this.stateMap = stateMap;
        if (hasStateDetails) this.stateDetails = stateDetails;
        if (hasWorkspaceStateDetails) this.workspaceStateDetails = workspaceStateDetails;
        if (hasLabelMap) this.labelMap = labelMap;
        if (hasWorkspaceMemberMap) this.workSpaceMemberRolesMap = workspaceMemberRolesMap;
        if (hasMemberMap) this.memberMap = memberMap;
        if (hasProjectMap) this.projectMap = projectMap;
        if (hasModuleMap) this.moduleMap = moduleMap;
        if (hasCycleMap) this.cycleMap = cycleMap;
      });
    });

    this.issues = new IssueStore(rootStore);

    this.issueDetail = new IssueDetail(this, EIssueServiceType.ISSUES);
    this.epicDetail = new IssueDetail(this, EIssueServiceType.EPICS);

    this.workspaceIssuesFilter = new WorkspaceIssuesFilter(this);
    this.workspaceIssues = new WorkspaceIssues(this, this.workspaceIssuesFilter);

    this.profileIssuesFilter = new ProfileIssuesFilter(this);
    this.profileIssues = new ProfileIssues(this, this.profileIssuesFilter);

    this.workspaceDraftIssuesFilter = new WorkspaceDraftIssuesFilter(this);
    this.workspaceDraftIssues = new WorkspaceDraftIssues(this);

    this.projectIssuesFilter = new ProjectIssuesFilter(this);
    this.projectIssues = new ProjectIssues(this, this.projectIssuesFilter);

    this.teamIssuesFilter = new ProjectIssuesFilter(this);
    this.teamIssues = new ProjectIssues(this, this.teamIssuesFilter);

    this.cycleIssuesFilter = new CycleIssuesFilter(this);
    this.cycleIssues = new CycleIssues(this, this.cycleIssuesFilter);

    this.moduleIssuesFilter = new ModuleIssuesFilter(this);
    this.moduleIssues = new ModuleIssues(this, this.moduleIssuesFilter);

    this.teamViewIssuesFilter = new ProjectViewIssuesFilter(this);
    this.teamViewIssues = new ProjectViewIssues(this, this.teamViewIssuesFilter);

    this.projectViewIssuesFilter = new ProjectViewIssuesFilter(this);
    this.projectViewIssues = new ProjectViewIssues(this, this.projectViewIssuesFilter);

    this.teamProjectWorkItemsFilter = new ProjectIssuesFilter(this);
    this.teamProjectWorkItems = new ProjectIssues(this, this.teamProjectWorkItemsFilter);

    this.archivedIssuesFilter = new ArchivedIssuesFilter(this);
    this.archivedIssues = new ArchivedIssues(this, this.archivedIssuesFilter);

    this.issueKanBanView = new IssueKanBanViewStore(this);
    this.issueCalendarView = new CalendarStore(this);

    this.projectEpicsFilter = new ProjectIssuesFilter(this);
    this.projectEpics = new ProjectIssues(this, this.projectEpicsFilter);
  }
}
