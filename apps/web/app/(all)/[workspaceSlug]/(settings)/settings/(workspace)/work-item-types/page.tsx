import { createFileRoute } from "@tanstack/react-router";
import { WorkItemTypesPreviewPage } from "@/components/work-item-types/preview-page";

export const Route = createFileRoute("/_authenticated/$workspaceSlug/settings/_workspace/work-item-types")({
  validateSearch: (search: Record<string, unknown>) => ({
    screen: ["Workspace library", "Hierarchy", "Project settings", "Work item"].find(
      (value) => value === search.screen
    ),
    state: ["Populated", "Empty", "Loading", "Error", "Read-only", "Not enabled"].find(
      (value) => value === search.state
    ),
  }),
  component: WorkItemTypesPage,
});

function WorkItemTypesPage() {
  const { screen, state } = Route.useSearch();
  return <WorkItemTypesPreviewPage key={`${screen}-${state}`} screen={screen} initialState={state} />;
}
