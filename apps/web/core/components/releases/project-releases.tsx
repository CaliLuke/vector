import { useQueryClient } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { useCurrentUserId } from "@/hooks/use-current-user";
import { useIdentityGeneration } from "@/hooks/use-identity-generation";
import { useProjectPermissions } from "@/hooks/use-permissions";
import { EmptyState } from "@/components/work-item-types/controls";
import { typeErrorMessage } from "@/services/work-item-types.service";
import type { ReleaseScope } from "@/queries/project-releases";
import { ReleaseCatalog } from "./release-catalog";
import { useReleaseCatalog } from "./use-release-catalog";

export function ProjectReleases({ workspaceSlug, projectId }: { workspaceSlug: string; projectId: string }) {
  const accountId = useCurrentUserId(),
    generation = useIdentityGeneration(useQueryClient());
  const permissions = useProjectPermissions(workspaceSlug, projectId);
  if (permissions.isPending) return <output>Loading project access…</output>;
  if (permissions.isError)
    return (
      <EmptyState
        title="Couldn't verify project access"
        description="Try loading the current project permissions again."
      >
        <Button
          onClick={() => {
            void permissions.membershipQuery.refetch();
            void permissions.rolesQuery.refetch();
          }}
        >
          Retry
        </Button>
      </EmptyState>
    );
  if (permissions.projectRole === undefined) return null;
  const scope = {
    workspaceSlug,
    projectId,
    accountId,
    generation,
    projectRole: permissions.projectRole,
    membershipRole: permissions.workspaceRole,
    membershipRevision: permissions.membershipQuery.data?.updated_at,
  };
  return <ReleaseSettings key={JSON.stringify(scope)} scope={scope} />;
}
function ReleaseSettings({ scope }: { scope: ReleaseScope }) {
  const catalog = useReleaseCatalog(scope);
  if (catalog.denied || (catalog.query.isError && !catalog.query.data))
    return (
      <EmptyState title="Couldn't load releases" description={typeErrorMessage(catalog.query.error)}>
        <Button onClick={() => void catalog.retry()}>Retry</Button>
      </EmptyState>
    );
  if (!catalog.query.data)
    return <output aria-label="Loading releases" className="block h-24 animate-pulse rounded bg-layer-1" />;
  return (
    <div>
      {catalog.query.isError && (
        <p role="alert" className="mb-4 text-13 text-danger-primary">
          {typeErrorMessage(catalog.query.error)}{" "}
          <Button variant="secondary" onClick={() => void catalog.retry()}>
            Retry loading releases
          </Button>
        </p>
      )}
      <ReleaseCatalog
        catalog={catalog.query.data}
        pending={catalog.pending}
        onSave={catalog.save}
        onReload={catalog.reload}
      />
    </div>
  );
}
