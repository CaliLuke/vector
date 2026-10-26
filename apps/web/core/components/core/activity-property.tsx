import { ListFilter } from "lucide-react";
import type { IIssueActivity } from "@plane/types";
import { IssueLink } from "./activity-links";
import { PropertyActivityMessage } from "./property-activity-message";

export const propertyActivityDetails = Object.fromEntries(
  ["type", "custom_property", "custom_properties"].map((field) => [
    field,
    {
      message: (activity: IIssueActivity, showIssue: boolean) => (
        <>
          <PropertyActivityMessage activity={activity} />
          {showIssue && (
            <>
              {" "}
              on <IssueLink activity={activity} />
            </>
          )}
        </>
      ),
      icon: <ListFilter size={12} className="text-secondary" aria-hidden="true" />,
    },
  ])
);
