import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import type { ISearchIssueResponse, TIssueRelationTypes } from "@plane/types";

export async function submitRelationSelection(
  selected: ISearchIssueResponse[],
  relation: TIssueRelationTypes,
  create: (relation: TIssueRelationTypes, issueIds: string[]) => Promise<unknown>
) {
  if (selected.length === 0) {
    setToast({
      type: TOAST_TYPE.ERROR,
      title: "Error!",
      message: "Please select at least one work item.",
    });
    return false;
  }
  await create(
    relation,
    selected.map(({ id }) => id)
  );
  return true;
}
