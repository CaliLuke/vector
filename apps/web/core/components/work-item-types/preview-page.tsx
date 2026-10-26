import { type ComponentType, useState } from "react";
import { Button } from "@plane/propel/button";
import { PageHead } from "@/components/core/page-title";
import { SettingsContentWrapper } from "@/components/settings/content-wrapper";
import { TypeLibrary } from "./library";
import { WorkspaceTypeLibrary } from "./workspace-library";
import { WorkspaceHierarchy } from "./workspace-hierarchy";
import { HierarchyPreview } from "./hierarchy";
import { ProjectTypesPreview } from "./project-preview";
import { WorkItemsPreview } from "./work-item-preview";
import { EmptyState } from "./controls";
import { useTypePreview } from "./preview-provider";

const savedScreens: Record<string, ComponentType | undefined> = {
  "Workspace library": WorkspaceTypeLibrary,
  Hierarchy: WorkspaceHierarchy,
};

export function WorkItemTypesPreviewPage({
  hierarchy = false,
  screen = hierarchy ? "Hierarchy" : "Workspace library",
  initialState = "Populated",
}: {
  hierarchy?: boolean;
  screen?: string;
  initialState?: string;
}) {
  const { notice, content, setContent } = useTypePreview();
  const [scenario, setScenario] = useState(initialState);
  const SavedScreen = initialState === "Populated" ? savedScreens[screen] : undefined;
  const readOnly = scenario === "Read-only";
  const title =
    (
      { Hierarchy: "Hierarchy", "Project settings": "Project work item Types", "Work item": "Work items" } as Record<
        string,
        string
      >
    )[screen] ?? "Work item Types";
  const descriptions: Record<string, string> = {
    "Workspace library": "Create consistent work item types and properties across your workspace.",
    Hierarchy: "Define how work breaks down across your teams.",
    "Project settings": "Bring the right workspace types into each project.",
    "Work item": "Manage work item types and custom properties.",
  };
  return (
    <SettingsContentWrapper hugging={screen === "Work item"}>
      <PageHead title={`${title} · Vector`} />
      <div className="mx-auto max-w-6xl">
        <div className="mb-7">
          <h1 className="text-20 font-semibold text-primary">{title}</h1>
          <p className="mt-2 text-13 text-tertiary">{descriptions[screen]}</p>
        </div>
        {notice && <output className="mb-5 rounded-md bg-layer-1 px-3 py-2 text-12 text-secondary">{notice}</output>}
        {SavedScreen ? (
          <SavedScreen />
        ) : scenario === "Loading" ? (
          <div aria-busy="true" aria-label="Loading settings" className="space-y-4">
            {[0, 1, 2, 3].map((index) => (
              <div key={index} className="h-20 animate-pulse rounded-lg bg-layer-1" />
            ))}
          </div>
        ) : scenario === "Error" ? (
          <EmptyState
            title="Couldn't load these settings"
            description="Your changes have not been lost. Try loading the settings again."
          >
            <Button variant="secondary" onClick={() => setScenario("Populated")}>
              Try again
            </Button>
          </EmptyState>
        ) : scenario === "Empty" ? (
          <EmptyState
            title="Nothing here yet"
            description="Create a type, attach properties, and import it into a project to start shaping your work."
          >
            <Button onClick={() => setScenario("Populated")}>Explore sample content</Button>
          </EmptyState>
        ) : scenario === "Not enabled" || !content.enabled ? (
          <EmptyState
            title="Give every kind of work its own type"
            description="Enable work item types to create shared definitions and custom properties. Once enabled, this feature cannot be disabled."
          >
            <Button
              onClick={() => {
                setContent((current) => ({ ...current, enabled: true }));
                setScenario("Populated");
              }}
            >
              Turn on Work item types
            </Button>
          </EmptyState>
        ) : (
          <div key={`${screen}-${readOnly}`}>
            <SettingsScreen screen={screen} readOnly={readOnly} />
          </div>
        )}
      </div>
    </SettingsContentWrapper>
  );
}

function SettingsScreen({ screen, readOnly }: { screen: string; readOnly: boolean }) {
  switch (screen) {
    case "Workspace library":
      return <TypeLibrary readOnly={readOnly} />;
    case "Hierarchy":
      return <HierarchyPreview readOnly={readOnly} />;
    case "Project settings":
      return <ProjectTypesPreview readOnly={readOnly} />;
    default:
      return <WorkItemsPreview readOnly={readOnly} />;
  }
}
