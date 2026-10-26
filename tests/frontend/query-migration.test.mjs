import { seedInstanceInfo } from "./helpers/instance-info.mjs";
import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { QueryObserver } from "@tanstack/react-query";

const state = { repositories: [], channels: [], deleteChannel: async () => {} };
globalThis.vectorQueryTest = state;
globalThis.vectorQueryReact = React;
const stubs = {
  "@/services/workspace-notification.service": `export default {
    fetchUnreadNotificationsCount: async (workspace) => ({mention_unread_notifications_count: 0, total_unread_notifications_count: workspace === "acme" ? 3 : 8})
  };`,
  "@/components/common/count-chip":
    "export const CountChip = ({count}) => globalThis.vectorQueryReact.createElement('output', null, count);",
  "@/hooks/use-integration-popup": "export default () => ({ startAuth: async () => {} });",
  "@plane/services":
    "export class APITokenService { async destroy() {} } export class InstanceService { async info() { return {instance:{is_setup_done:true},config:{}}; } }",
  "@plane/i18n": "export const useTranslation = () => ({ t: (key) => key });",
  "@plane/propel/toast": "export const TOAST_TYPE = {ERROR: 'error'}; export const setToast = () => {};",
  "@plane/ui": `const React = globalThis.vectorQueryReact;
    export const AlertModalCore = ({handleSubmit}) => React.createElement('button', {onClick: handleSubmit}, 'Delete');
    export const CustomSearchSelect = ({options, footerOption}) => React.createElement('div', null,
      React.createElement('output', null, options.map(option => option.query).join(',')), footerOption);
    export const Loader = Object.assign(({children}) => children, {Item: () => null});`,
  "@/services/project": `export class ProjectService { async getGithubRepositories(url) {
    globalThis.vectorQueryTest.repositories.push(url);
    const page = Number(new URL(url, 'http://localhost').searchParams.get('page'));
    const other = url.includes('/second/');
    return { total_count: other ? 1 : 3, repositories: other ? [{id: 'other', full_name: 'other/repo'}] :
      page === 1 ? [{id: 'one', full_name: 'acme/one'}, {id: 'two', full_name: 'acme/two'}] :
      [{id: 'three', full_name: 'acme/three'}] };
  } }`,
  "@/services/app_installation.service": `export class AppInstallationService {
    async getSlackChannelDetail() { return [...globalThis.vectorQueryTest.channels]; }
    async removeSlackChannel() { await globalThis.vectorQueryTest.deleteChannel(); globalThis.vectorQueryTest.channels = []; }
  }`,
};
registerWebModuleHooks();
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

await test("notification queries fetch the new workspace instead of reusing the previous workspace count", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { NotificationAppSidebarOption } =
    await import("../../apps/web/core/components/workspace-notifications/notification-app-sidebar-option.tsx");
  await fixture.render(React.createElement(NotificationAppSidebarOption, { workspaceSlug: "acme" }));
  assert.equal(fixture.environment.container.textContent, "3");
  await fixture.render(React.createElement(NotificationAppSidebarOption, { workspaceSlug: "other" }));
  assert.equal(fixture.environment.container.textContent, "8");
  assert.equal(fixture.client.getQueryData(["notifications", "acme", "count"]).total_unread_notifications_count, 3);
});

await test("GitHub pagination appends pages, stops at the total, and resets for a different integration", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { SelectRepository } = await import("../../apps/web/core/components/integration/github/select-repository.tsx");
  state.repositories = [];
  const props = { integration: { id: "first" }, value: null, label: "Repository", onChange() {} };
  await fixture.render(React.createElement(SelectRepository, props));
  assert.equal(fixture.environment.container.querySelector("output").textContent, "acme/one,acme/two");
  await act(async () => fixture.environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("output").textContent, "acme/one,acme/two,acme/three");
  assert.equal(fixture.environment.container.querySelector("button"), null);
  assert.ok(state.repositories[1].endsWith("?page=2"));
  await fixture.render(React.createElement(SelectRepository, { ...props, integration: { id: "second" } }));
  assert.equal(fixture.environment.container.querySelector("output").textContent, "other/repo");
  assert.ok(state.repositories[2].endsWith("?page=1"));
});

await test("disconnecting a Slack channel waits for deletion before refreshing the typed cache", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { SelectChannel } = await import("../../apps/web/core/components/integration/slack/select-channel.tsx");
  state.channels = [{ id: "channel", project: "p1" }];
  let finishDelete;
  state.deleteChannel = () =>
    new Promise((resolve) => {
      finishDelete = resolve;
    });
  await fixture.render(React.createElement(SelectChannel, { integration: { id: "slack" } }));
  const toggle = fixture.environment.container.querySelector('[role="switch"]');
  assert.equal(toggle.getAttribute("aria-checked"), "true");
  await act(async () => toggle.click());
  await settleQuery();
  assert.equal(toggle.disabled, true);
  assert.equal(toggle.getAttribute("aria-checked"), "true");
  await act(async () => finishDelete());
  await settleQuery();
  assert.equal(toggle.getAttribute("aria-checked"), "false");
  assert.deepEqual(fixture.client.getQueryData(["SLACK_CHANNEL_INFO", "acme", "p1", "slack"]), []);
});

await test("a failed Slack deletion keeps the channel connected and can be retried", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { SelectChannel } = await import("../../apps/web/core/components/integration/slack/select-channel.tsx");
  state.channels = [{ id: "channel", project: "p1" }];
  state.deleteChannel = async () => {
    throw new Error("offline");
  };
  await fixture.render(React.createElement(SelectChannel, { integration: { id: "slack" } }));
  const toggle = fixture.environment.container.querySelector('[role="switch"]');
  await act(async () => toggle.click());
  await settleQuery();
  assert.equal(toggle.getAttribute("aria-checked"), "true");
  assert.equal(toggle.disabled, false);
  assert.deepEqual(fixture.client.getQueryData(["SLACK_CHANNEL_INFO", "acme", "p1", "slack"]), [
    { id: "channel", project: "p1" },
  ]);
});

await test("deleting an API token cancels an older list request before updating the cache", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { DeleteApiTokenModal } = await import("../../apps/web/core/components/api-token/delete-token-modal.tsx");
  fixture.client.setQueryData(["API_TOKENS_LIST"], [{ id: "deleted" }, { id: "kept" }]);
  let resolveList;
  const pendingList = fixture.client.fetchQuery({
    queryKey: ["API_TOKENS_LIST"],
    queryFn: () =>
      new Promise((resolve) => {
        resolveList = resolve;
      }),
  });
  await fixture.render(React.createElement(DeleteApiTokenModal, { isOpen: true, tokenId: "deleted", onClose() {} }));
  await act(async () => fixture.environment.container.querySelector("button").click());
  await settleQuery();
  resolveList([{ id: "deleted" }, { id: "kept" }]);
  await pendingList;
  assert.deepEqual(fixture.client.getQueryData(["API_TOKENS_LIST"]), [{ id: "kept" }]);
});

await test("deleting a token while its list first loads refetches the list instead of leaving it pending", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  seedInstanceInfo(fixture.client);
  const { DeleteApiTokenModal } = await import("../../apps/web/core/components/api-token/delete-token-modal.tsx");
  let resolveOriginalList;
  const list = new QueryObserver(fixture.client, {
    queryKey: ["API_TOKENS_LIST"],
    queryFn: () =>
      resolveOriginalList
        ? Promise.resolve([{ id: "kept" }])
        : new Promise((resolve) => {
            resolveOriginalList = resolve;
          }),
  });
  const unsubscribe = list.subscribe(() => {});
  t.after(unsubscribe);
  await fixture.render(React.createElement(DeleteApiTokenModal, { isOpen: true, tokenId: "deleted", onClose() {} }));
  await act(async () => fixture.environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(list.getCurrentResult().status, "success");
  assert.deepEqual(list.getCurrentResult().data, [{ id: "kept" }]);
  resolveOriginalList([{ id: "deleted" }, { id: "kept" }]);
  await settleQuery();
  assert.deepEqual(list.getCurrentResult().data, [{ id: "kept" }]);
});
