import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act, webRequire } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";

registerWebModuleHooks();
const fixture = { React, modal: undefined, writes: [], navigations: [], save: undefined };
globalThis.intakeAcceptNavigationFixture = fixture;
const state = "globalThis.intakeAcceptNavigationFixture";
const empty = (name) => `export const ${name} = () => null;`;
const stubs = {
  "@tanstack/react-router": `export const useRouter = () => ({ navigate: (to) => { ${state}.navigations.push(to); return Promise.resolve(); } });`,
  "@plane/i18n": `export const useTranslation = () => ({ t: (key) => key });`,
  "@/hooks/use-current-user": `export const useCurrentUser = () => ({ data: ${state}.user });`,
  "@/hooks/use-permissions": `export const useProjectPermissions = () => ({ can: () => Boolean(${state}.user) });`,
  "@/hooks/use-projects": `export const useProjectDetail = () => ({ data: { id: "project", identifier: "TEST" } });`,
  "@/hooks/use-intake": `export const useIntakeMutation = () => ({ mutateAsync: (input) => { ${state}.writes.push(input); return ${state}.save(input); } });`,
  "@/components/issues/issue-modal/modal": `const React = ${state}.React; export function CreateUpdateIssueModal(props) { ${state}.modal = props; return props.isOpen ? React.createElement("div", { "data-testid": "accept-editor" }, "Edit accepted work item") : null; }`,
  "@/components/issues/issue-update-status": empty("NameDescriptionUpdateStatus"),
  "@/components/issues/work-item-selection-modal": empty("WorkItemSelectionModal"),
  "../inbox-issue-status": empty("InboxIssueStatus"),
  "../modals/decline-issue-modal": empty("DeclineIssueModal"),
  "../modals/delete-issue-modal": empty("DeleteInboxIssueModal"),
  "../modals/snooze-issue-modal": empty("InboxIssueSnoozeModal"),
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});
const { EInboxIssueStatus, EInboxIssueCurrentTab } = webRequire("@plane/types");
const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");

async function setup(t, anonymous = false) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({ defaultOptions: { queries: { gcTime: Infinity, retry: false } } });
  client.setQueryData(identityKeys.currentUser(), anonymous ? undefined : { id: "alice" });
  t.after(async () => {
    await env.unmount();
    client.clear();
  });
  const { InboxIssueActionsHeader } =
    await import("../../apps/web/core/components/inbox/content/inbox-issue-header.tsx");
  fixture.user = anonymous ? undefined : { id: "alice" };
  fixture.writes = [];
  fixture.navigations = [];
  fixture.save = async () => ({});
  await env.render(
    React.createElement(
      QueryClientProvider,
      { client },
      React.createElement(InboxIssueActionsHeader, {
        workspaceSlug: "acme",
        projectId: "project",
        inboxIssue: {
          id: "intake-1",
          status: EInboxIssueStatus.PENDING,
          issue: { id: "item-1", project_id: "project", sequence_id: 1, created_by: anonymous ? undefined : "alice" },
        },
        navigation: { tab: EInboxIssueCurrentTab.OPEN, issueIds: ["item-1", "item-2"] },
        isSubmitting: "saved",
        isMobileSidebar: false,
        setIsMobileSidebar() {},
        onDeleteAccepted() {},
        isNotificationEmbed: false,
      })
    )
  );
  if (anonymous) return env;
  const accept = [...env.container.querySelectorAll("button")].find((button) =>
    button.textContent.includes("inbox_issue.actions.accept")
  );
  assert.ok(accept);
  await act(async () => accept.click());
  assert.ok(env.container.querySelector('[data-testid="accept-editor"]'));
  return env;
}

await test("acceptance keeps the editor open until its subsequent work-item save succeeds", async (t) => {
  const env = await setup(t);
  await act(async () => fixture.modal.beforeFormSubmit());
  assert.equal(fixture.writes[0].data.status, EInboxIssueStatus.ACCEPTED);
  assert.ok(env.container.querySelector('[data-testid="accept-editor"]'));
  assert.deepEqual(fixture.navigations, []);
  await act(async () => fixture.modal.onSubmit({ id: "item-1" }));
  assert.equal(env.container.querySelector('[data-testid="accept-editor"]'), null);
  assert.equal(fixture.navigations.length, 1);
  assert.match(fixture.navigations[0].href, /inboxIssueId=item-2/);
});

await test("rejected intake acceptance leaves the editor and current route intact", async (t) => {
  const env = await setup(t);
  fixture.save = async () => {
    throw new Error("acceptance rejected");
  };
  await act(async () => assert.rejects(fixture.modal.beforeFormSubmit(), /acceptance rejected/));
  assert.ok(env.container.querySelector('[data-testid="accept-editor"]'));
  assert.deepEqual(fixture.navigations, []);
});

await test("an unavailable account cannot inherit creator-only delete controls from a missing creator", async (t) => {
  const env = await setup(t, true);
  const menuButton = env.container.querySelector('button[aria-haspopup="menu"]');
  assert.ok(menuButton, "the mobile action menu should be mounted");
  await act(async () => menuButton.click());
  assert.equal(document.querySelector('[role="menuitem"]'), null, "no write action is available without an account");
  assert.deepEqual(fixture.writes, []);
});
