export async function seedWorkspaceQuery(client, workspace = { id: "workspace", slug: "workspace" }) {
  const identity = await import("../../../apps/web/core/queries/identity.ts");
  const workspaces = await import("../../../apps/web/core/queries/workspaces.ts");
  const userId = "workspace-query-test-user";
  client.setQueryData(identity.identityKeys.currentUser(), { id: userId });
  client.setQueryData(workspaces.workspaceKeys.list(userId), [workspace]);
}
