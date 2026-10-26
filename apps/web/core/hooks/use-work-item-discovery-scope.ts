import { useParams } from "@tanstack/react-router";
import { EIssuesStoreType } from "@plane/types";
import { useIssueListType } from "./use-issue-list-type";

type DiscoveryLocation = { workspaceSlug: string | undefined; projectId: string | undefined };

/** Scope shared by authorized list grouping and custom-property columns. */
export function useWorkItemDiscoveryScope(enabled: boolean, location?: DiscoveryLocation) {
  const route = useParams({ strict: false });
  const storeType = useIssueListType();
  const supported = [
    EIssuesStoreType.PROJECT,
    EIssuesStoreType.PROJECT_VIEW,
    EIssuesStoreType.MODULE,
    EIssuesStoreType.CYCLE,
    EIssuesStoreType.GLOBAL,
  ].includes(storeType);
  const workspaceSlug = location ? location.workspaceSlug : route.workspaceSlug;
  const projectId = storeType === EIssuesStoreType.GLOBAL ? undefined : location ? location.projectId : route.projectId;
  const canLoad = enabled && supported && !!workspaceSlug && (storeType === EIssuesStoreType.GLOBAL || !!projectId);
  return { workspaceSlug: workspaceSlug ?? "", projectId, canLoad, supported };
}
