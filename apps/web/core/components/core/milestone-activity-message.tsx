import type { IIssueActivity } from "@plane/types";

type MilestoneActivity = Partial<Pick<IIssueActivity, "old_value" | "new_value">>;

export function milestoneActivityContent(newValue?: string | null, oldValue?: string | null) {
  return newValue
    ? { action: "set the milestone to", value: newValue, showConnector: false }
    : { action: "removed the milestone", value: oldValue ?? "", showConnector: false };
}

/** Display the committed title snapshot even after the milestone is deleted. */
export function MilestoneActivityMessage({ activity }: { activity: MilestoneActivity }) {
  const { action, value } = milestoneActivityContent(activity.new_value, activity.old_value);
  return (
    <>
      {action} <span className="font-medium whitespace-pre-wrap">{value}</span>.
    </>
  );
}
