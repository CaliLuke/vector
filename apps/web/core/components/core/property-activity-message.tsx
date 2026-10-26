import type { IIssueActivity } from "@plane/types";

type PropertyActivity = Partial<Pick<IIssueActivity, "field" | "verb" | "comment" | "old_value" | "new_value">>;

/** Activity stores historical labels and readable values; never interpret them as HTML. */
export function PropertyActivityMessage({ activity }: { activity: PropertyActivity }) {
  if (activity.field === "custom_properties")
    return (
      <>
        {activity.comment || "updated custom properties"}. {activity.new_value}
        {activity.old_value && <span className="whitespace-pre-wrap"> Previous values: {activity.old_value}</span>}
      </>
    );
  const label = activity.field === "type" ? "type" : activity.comment || "custom property";
  return (
    <>
      {activity.verb === "restored" ? "restored" : "changed"} <span className="font-medium">{label}</span> from{" "}
      <span className="whitespace-pre-wrap">{activity.old_value || "empty"}</span> to{" "}
      <span className="whitespace-pre-wrap">{activity.new_value || "empty"}</span>.
    </>
  );
}
