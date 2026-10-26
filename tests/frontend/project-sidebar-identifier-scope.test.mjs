import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { createQueryRouterFixture } from "./helpers/query-router.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { seedPermissionQueries } from "./helpers/permission-query-fixture.mjs";
import { waitForQueryState } from "./helpers/query-state.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.projectSidebarIdentifierReact = React;

const stubs = {
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/icons": `const Icon = () => null; export const CycleIcon=Icon,IntakeIcon=Icon,ModuleIcon=Icon,PageIcon=Icon,ViewsIcon=Icon,WorkItemsIcon=Icon;`,
  "@/providers/sidebar-state":
    "export const useSidebarActions = () => ({ toggleSidebar() {}, toggleExtendedProjectSidebar() {} }); export const useExtendedProjectSidebarOpen = () => false;",
  "@/components/sidebar/sidebar-navigation": `const React=globalThis.projectSidebarIdentifierReact;export const SidebarNavItem=({isActive,children})=>React.createElement("div",{"data-sidebar-active":String(isActive)},children);`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const workspaceSlug = "acme";
const accountId = "alice";
const membershipRevision = "membership-1";
const projectRows = [
  { id: "p1", identifier: "ALPHA", name: "Alpha", cycle_view: true, module_view: true },
  { id: "p2", identifier: "BETA", name: "Beta", cycle_view: true, module_view: true },
];

async function renderSidebar(t, isEpic) {
  const fixture = await createQueryRouterFixture(t, "/acme/browse/ALPHA-7");
  const { client, environment } = fixture;
  const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
  const { projectKeys } = await import("../../apps/web/core/queries/projects.ts");
  const { issueDetailKeys } = await import("../../apps/web/core/queries/issue-details.ts");
  const { EIssueServiceType, EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");
  const { ProjectService } = await import("../../apps/web/core/services/project/project.service.ts");
  const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
  const { ProjectNavigation } = await import("../../apps/web/core/components/workspace/sidebar/project-navigation.tsx");

  client.setQueryData(identityKeys.currentUser(), { id: accountId });
  await seedPermissionQueries(client, {
    userId: accountId,
    workspaceSlug,
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    revision: membershipRevision,
    projectRoles: { p1: 20, p2: 20 },
  });

  const scope = {
    accountId,
    sessionGeneration: 0,
    workspaceSlug,
    workspaceRole: EUserWorkspaceRoles.ADMIN,
    membershipRevision,
    projectRoles: [
      ["p1", 20],
      ["p2", 20],
    ],
  };
  client.setQueryData(projectKeys.workspace(scope, "partial"), projectRows);

  const canonicalIssueScope = {
    accountId,
    sessionGeneration: 0,
    workspaceSlug,
    membershipRole: EUserWorkspaceRoles.ADMIN,
    membershipRevision,
    projectId: "p1",
    projectMembershipRole: 20,
    projectRole: 20,
    serviceType: EIssueServiceType.ISSUES,
  };
  const detailRead = waitForQueryState(
    client,
    issueDetailKeys.byIdentifier(canonicalIssueScope, "ALPHA", "7"),
    (state) => state?.status === "success"
  );

  const identifierRequests = [];
  t.mock.method(IssueService.prototype, "retrieveWithIdentifier", async (_slug, identifier, sequenceId) => {
    identifierRequests.push({ identifier, sequenceId });
    return {
      id: isEpic ? "epic-1" : "issue-1",
      project_id: "p1",
      project_identifier: "ALPHA",
      sequence_id: 7,
      is_epic: isEpic,
    };
  });
  t.mock.method(ProjectService.prototype, "getProjectsLite", async () => projectRows);

  const projectNavigation = (projectId) =>
    React.createElement(ProjectNavigation, {
      workspaceSlug,
      projectId,
      additionalNavigationItems: (slug, id) => [
        {
          key: "epics",
          name: "Epics",
          i18n_key: "sidebar.epics",
          href: `/${slug}/projects/${id}/epics`,
          icon: () => null,
          access: [20, 15],
          shouldRender: true,
          sortOrder: 0,
        },
      ],
    });
  const view = () =>
    React.createElement(
      React.Fragment,
      null,
      React.createElement("section", { "data-project": "p1" }, projectNavigation("p1")),
      React.createElement("section", { "data-project": "p2" }, projectNavigation("p2"))
    );
  await fixture.render(view());
  await act(async () => detailRead);

  return { client, environment, identifierRequests };
}

function activeState(container, projectId, href) {
  const section = container.querySelector(`[data-project="${projectId}"]`);
  const link = [...section.querySelectorAll("a")].find((anchor) => anchor.getAttribute("href") === href);
  assert.ok(link, `expected sidebar link ${href}`);
  return link.querySelector("[data-sidebar-active]").getAttribute("data-sidebar-active");
}

await test("sidebar loads and activates only the project matching a work item identifier", async (t) => {
  const { environment, identifierRequests } = await renderSidebar(t, false);
  assert.deepEqual(identifierRequests, [{ identifier: "ALPHA", sequenceId: "7" }]);
  assert.equal(activeState(environment.container, "p1", "/acme/projects/p1/issues"), "true");
  assert.equal(activeState(environment.container, "p1", "/acme/projects/p1/epics"), "false");
  assert.equal(activeState(environment.container, "p2", "/acme/projects/p2/issues"), "false");
  assert.equal(activeState(environment.container, "p2", "/acme/projects/p2/epics"), "false");
});

await test("sidebar activates epics only for the matching canonical epic project", async (t) => {
  const { environment, identifierRequests } = await renderSidebar(t, true);
  assert.deepEqual(identifierRequests, [{ identifier: "ALPHA", sequenceId: "7" }]);
  assert.equal(activeState(environment.container, "p1", "/acme/projects/p1/issues"), "false");
  assert.equal(activeState(environment.container, "p1", "/acme/projects/p1/epics"), "true");
  assert.equal(activeState(environment.container, "p2", "/acme/projects/p2/issues"), "false");
  assert.equal(activeState(environment.container, "p2", "/acme/projects/p2/epics"), "false");
});
