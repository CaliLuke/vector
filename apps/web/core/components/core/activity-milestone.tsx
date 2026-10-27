import { Flag } from "lucide-react";
import type { IIssueActivity } from "@plane/types";
import { IssueLink } from "./activity-links";
import { MilestoneActivityMessage } from "./milestone-activity-message";

export const milestoneActivityDetails = {
  milestone: {
    message: (activity: IIssueActivity, showIssue: boolean) => (
      <>
        <MilestoneActivityMessage activity={activity} />
        {showIssue && (
          <>
            {" "}
            on <IssueLink activity={activity} />
          </>
        )}
      </>
    ),
    icon: <Flag size={12} className="text-secondary" aria-hidden="true" />,
  },
};
