import { createFileRoute } from "@tanstack/react-router";
import { WorkItemTypesPreviewPage } from "@/components/work-item-types/preview-page";

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/hierarchy")({
  component: () => <WorkItemTypesPreviewPage hierarchy />,
});
