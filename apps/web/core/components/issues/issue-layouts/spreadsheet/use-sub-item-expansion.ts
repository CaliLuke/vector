import type { Dispatch, MouseEvent, SetStateAction } from "react";
import type { TIssue } from "@plane/types";
import { EIssueServiceType } from "@plane/types";
import { useWorkItemSubItems } from "@/hooks/use-work-item-detail";

export function useSubItemExpansion({
  workspaceSlug,
  issue,
  nestingLevel,
  expanded,
  setExpanded,
  onOpen,
}: {
  workspaceSlug: string | undefined;
  issue: TIssue | undefined;
  nestingLevel: number;
  expanded: boolean;
  setExpanded: Dispatch<SetStateAction<boolean>>;
  onOpen: (issue: TIssue) => void;
}) {
  const childQuery = useWorkItemSubItems(
    workspaceSlug,
    issue?.project_id ?? undefined,
    issue?.id,
    EIssueServiceType.ISSUES,
    expanded
  );
  const error = childQuery.isError;
  return {
    label: error ? "Could not load sub-items. Retry" : expanded ? "Collapse sub-items" : "Expand sub-items",
    title: error ? "Could not load sub-items. Click to retry." : undefined,
    onClick: (event: MouseEvent<HTMLButtonElement>) => {
      event.stopPropagation();
      event.preventDefault();
      if (!issue) return;
      if (nestingLevel >= 3) {
        onOpen(issue);
        return;
      }
      setExpanded(!expanded);
    },
  };
}
