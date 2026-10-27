import { Flag } from "lucide-react";
import type { TIssueActivity } from "@plane/types";
import { MilestoneActivityMessage } from "@/components/core/milestone-activity-message";
import { IssueActivityBlockComponent } from "./helpers/activity-block";

export function IssueMilestoneActivity({
  activity,
  ends,
}: {
  activity: TIssueActivity;
  ends: "top" | "bottom" | undefined;
}) {
  return (
    <IssueActivityBlockComponent
      activity={activity}
      ends={ends}
      wrap
      icon={<Flag size={14} className="text-secondary" aria-hidden="true" />}
    >
      <MilestoneActivityMessage activity={activity} />
    </IssueActivityBlockComponent>
  );
}
