import { ListFilter } from "lucide-react";
import { PropertyActivityMessage } from "@/components/core/property-activity-message";
import type { TIssueActivity } from "@plane/types";
import { IssueActivityBlockComponent } from "./helpers/activity-block";

export const IssuePropertyActivity = function IssuePropertyActivity({
  activity,
  ends,
}: {
  activity: TIssueActivity;
  ends: "top" | "bottom" | undefined;
}) {
  if (!activity || !["type", "custom_property", "custom_properties"].includes(activity.field ?? "")) return null;
  return (
    <IssueActivityBlockComponent
      activity={activity}
      ends={ends}
      wrap
      icon={<ListFilter size={14} className="text-secondary" aria-hidden="true" />}
    >
      <PropertyActivityMessage activity={activity} />
    </IssueActivityBlockComponent>
  );
};
