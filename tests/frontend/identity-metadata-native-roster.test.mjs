import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, createPermissionUiFixture } from "./helpers/permission-ui-fixture.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.identityMetadataTestReact = React;
globalThis.identityMetadataTestState = { workspaces: {} };

registerHooks({
  resolve(specifier, context, nextResolve) {
    const sources = {
      "@tanstack/react-router": `
        const React = globalThis.identityMetadataTestReact;
        export const Link = ({ to, params, children, ...props }) => React.createElement(
          "a",
          { ...props, href: to.replace("$workspaceSlug", params.workspaceSlug).replace("$userId", params.userId) },
          children
        );
      `,
      "@/hooks/use-workspaces": `
        export const useWorkspaceById = (workspaceId) => ({ data: globalThis.identityMetadataTestState.workspaces[workspaceId] });
      `,
      "@/providers/issue-detail-ui-state": `
        export const useIssueDetailUiState = () => ({ toggleIssueLinkModal() {} });
      `,
      "@/hooks/use-issue-resources": `export const useIssueLinks=()=>({links:[{id:"issue-link"}],query:{isError:false},create:async()=>{},update:async()=>{},remove:async()=>{}});`,
      "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
      "@/hooks/use-platform-os": "export const usePlatformOS = () => ({ isMobile: false });",
      "@/components/modules": `
        const React = globalThis.identityMetadataTestReact;
        export const ModulesLinksListItem = ({ workspaceSlug }) => React.createElement(
          "output", { "data-testid": "module-link-scope" }, workspaceSlug ?? "missing"
        );
      `,
      "@plane/propel/icons": "export const PlusIcon = () => null;",
      "@plane/propel/toast": "export const TOAST_TYPE = {}; export const setToast = () => {};",
    };
    if (
      specifier === "@plane/types" &&
      context.parentURL?.endsWith("/apps/web/core/components/issues/issue-detail/links/root.tsx")
    ) {
      return {
        shortCircuit: true,
        url: `data:text/javascript,${encodeURIComponent('export const EIssueServiceType = { ISSUES: "issues" };')}`,
      };
    }
    if (sources[specifier]) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(sources[specifier])}` };
    }
    const localStubs = {
      "./create-update-link-modal": `export const IssueLinkCreateUpdateModal = () => null;`,
      "./link-detail": `
        const React = globalThis.identityMetadataTestReact;
        export function IssueLinkDetail({ workspaceSlug }) {
          return React.createElement("output", { "data-testid": "issue-link-scope" }, workspaceSlug);
        }
      `,
      "../issue-detail-widgets/attachments/helper": `
        export const useAttachmentOperations = () => ({ operations: {}, snapshot: { uploadStatus: [], attachments: [{id:"attachment"}] } });
      `,
      "./attachment-upload": `export const IssueAttachmentUpload = () => null;`,
      "./attachment-detail": `
        const React = globalThis.identityMetadataTestReact;
        export const IssueAttachmentsDetail = ({ workspaceSlug }) => React.createElement(
          "output", { "data-testid": "attachment-scope" }, workspaceSlug
        );
      `,
      "./attachment-upload-details": `export const IssueAttachmentsUploadDetails = () => null;`,
    };
    if (localStubs[specifier] && context.parentURL?.includes("/apps/web/core/components/")) {
      return { shortCircuit: true, url: `data:text/javascript,${encodeURIComponent(localStubs[specifier])}` };
    }
    return nextResolve(specifier, context);
  },
});

const { User } = await import("../../apps/web/core/components/common/activity/user.tsx");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");
const { permissionKeys } = await import("../../apps/web/core/queries/permissions.ts");
const { memberRosterKeys } = await import("../../apps/web/core/queries/member-rosters.ts");
const { EUserWorkspaceRoles } = await import("../../packages/types/dist/index.mjs");

function deferred() {
  let resolve;
  const promise = new Promise((done) => (resolve = done));
  return { promise, resolve };
}

function activity(workspace, actor = "actor-1") {
  return { workspace, actor };
}

function rosterRow(id, displayName, isActive = true) {
  return { is_active: isActive, member: { id, display_name: displayName } };
}

await test("activity identity updates from native workspace roster reads and stays isolated by scope", async (t) => {
  const acmeRead = deferred();
  const betaRead = deferred();
  const acmeStarted = Promise.withResolvers();
  const betaStarted = Promise.withResolvers();
  t.mock.method(WorkspaceService.prototype, "fetchWorkspaceMembers", (slug) => {
    if (slug === "workspace-a") {
      acmeStarted.resolve();
      return acmeRead.promise;
    }
    if (slug === "workspace-b") {
      betaStarted.resolve();
      return betaRead.promise;
    }
    throw new Error(`Unexpected workspace ${slug}`);
  });
  const fixture = await createPermissionUiFixture(t, {
    path: "/workspace-a/projects/p1/issues/",
    userId: "viewer",
    workspaceRole: EUserWorkspaceRoles.MEMBER,
  });
  const state = globalThis.identityMetadataTestState;
  state.workspaces = {
    "workspace-id-a": { slug: "workspace-a" },
    "workspace-id-b": { slug: "workspace-b" },
  };
  fixture.client.setQueryData(permissionKeys.workspaceMembership("viewer", "workspace-b"), {
    is_active: true,
    role: EUserWorkspaceRoles.MEMBER,
    updated_at: "revision-b",
  });

  await fixture.render(React.createElement(User, { activity: activity("workspace-id-a") }));
  await acmeStarted.promise;
  assert.equal(fixture.environment.container.textContent, "Vector", "pending roster keeps the missing-actor fallback");

  await React.act(async () => {
    acmeRead.resolve([rosterRow("actor-1", "Ada in A")]);
    await new Promise((resolve) => setTimeout(resolve, 40));
  });
  const firstLink = fixture.environment.container.querySelector("a");
  assert.equal(firstLink?.textContent, "Ada in A");
  assert.equal(firstLink?.getAttribute("href"), "/workspace-a/profile/actor-1/");

  await fixture.render(React.createElement(User, { activity: activity("workspace-id-b") }));
  await betaStarted.promise;
  assert.equal(fixture.environment.container.textContent, "Vector", "the previous workspace roster is not reused");

  await React.act(async () => {
    betaRead.resolve([rosterRow("actor-1", "Bea in B")]);
    await new Promise((resolve) => setTimeout(resolve, 40));
  });
  const secondLink = fixture.environment.container.querySelector("a");
  assert.equal(secondLink?.textContent, "Bea in B");
  assert.equal(secondLink?.getAttribute("href"), "/workspace-b/profile/actor-1/");

  const workspaceBRosterKey = memberRosterKeys.workspace({
    accountId: "viewer",
    workspaceSlug: "workspace-b",
    membershipRole: EUserWorkspaceRoles.MEMBER,
    membershipRevision: "revision-b",
  });
  fixture.client.setQueryData(workspaceBRosterKey, [rosterRow("actor-1", "Inactive actor", false)]);
  await fixture.render(React.createElement(User, { activity: activity("workspace-id-b") }));
  assert.equal(
    fixture.environment.container.querySelector("a")?.textContent,
    "Inactive actor",
    "an inactive but returned membership preserves its activity attribution"
  );
});

await test("issue and module metadata parents pass their explicit workspace scope to link and attachment rows", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(async () => {
    await environment.unmount();
  });
  const { IssueLinkRoot } = await import("../../apps/web/core/components/issues/issue-detail/links/root.tsx");
  const { IssueAttachmentRoot } = await import("../../apps/web/core/components/issues/attachment/root.tsx");
  const { ModuleLinksList } = await import("../../apps/web/core/components/modules/links/list.tsx");

  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      React.createElement(IssueLinkRoot, {
        workspaceSlug: "workspace-alpha",
        projectId: "project-1",
        issueId: "issue-1",
      }),
      React.createElement(IssueAttachmentRoot, {
        workspaceSlug: "workspace-alpha",
        projectId: "project-1",
        issueId: "issue-1",
      }),
      React.createElement(ModuleLinksList, {
        links: [{ id: "module-link" }],
        workspaceSlug: "workspace-alpha",
        moduleId: "module-1",
        handleEditLink() {},
        handleDeleteLink() {},
      })
    )
  );

  assert.equal(environment.container.querySelector('[data-testid="issue-link-scope"]')?.textContent, "workspace-alpha");
  assert.equal(environment.container.querySelector('[data-testid="attachment-scope"]')?.textContent, "workspace-alpha");
  assert.equal(
    environment.container.querySelector('[data-testid="module-link-scope"]')?.textContent,
    "workspace-alpha"
  );
});
