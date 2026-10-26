import { createQueryRouterFixture, settleQuery } from "./helpers/query-router.mjs";
import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { React, act, webRequire } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const state = {};
globalThis.queryAuditState = state;
globalThis.queryAuditReact = React;
registerWebModuleHooks();
const stubs = {
  "@/services/workspace-notification.service":
    "export default new Proxy({}, { get: (_, key) => globalThis.queryAuditState.notifications[key] });",
  "@/components/common/count-chip":
    "export const CountChip = ({count}) => globalThis.queryAuditReact.createElement('output', null, count);",
  "@/components/integration/github/select-repository": `export const SelectRepository = ({label, onChange}) => globalThis.queryAuditReact.createElement('button', {
    onClick: () => onChange({id: 1, name: 'repo', full_name: 'acme/repo', owner: {login: 'acme'}, html_url: 'https://github.com/acme/repo'})
  }, label);`,
  "@/components/integration/slack/select-channel": "export const SelectChannel = () => null;",
  "@/services/project": `export class ProjectService {
    getProjectGithubRepository() { return globalThis.queryAuditState.getRepository(); }
    syncGithubRepository() { return globalThis.queryAuditState.syncRepository(); }
  }`,
  "@plane/propel/toast":
    "export const TOAST_TYPE = {SUCCESS: 'success', ERROR: 'error'}; export const setToast = () => {};",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith(".png?url"))
      return { shortCircuit: true, url: "data:text/javascript,export default 'image.png'" };
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});
const { notificationKeys, notificationCountOptions, notificationMutationOptions } =
  await import("../../apps/web/core/queries/notifications.ts");
const { NotificationsProvider, useNotifications } =
  await import("../../apps/web/core/components/workspace-notifications/provider.tsx");
const { MutationObserver } = webRequire("@tanstack/react-query");

function NotificationProbe() {
  const notifications = useNotifications();
  React.useEffect(() => {
    state.view = notifications;
  }, [notifications]);
  return React.createElement("output", null, notifications.notificationIds.join(","));
}
function notificationScreen(workspaceSlug) {
  return React.createElement(
    NotificationsProvider,
    { workspaceSlug, key: workspaceSlug },
    React.createElement(NotificationProbe)
  );
}
const { NotificationAppSidebarOption } =
  await import("../../apps/web/core/components/workspace-notifications/notification-app-sidebar-option.tsx");
const { IntegrationCard } = await import("../../apps/web/core/components/project/integration-card.tsx");

function deferred() {
  return Promise.withResolvers();
}
const notificationPage = (slug) => ({
  results: [{ id: `${slug}-notification`, workspace: slug }],
  next_cursor: `${slug}-next`,
  next_page_results: true,
});

await test("late notification counts for the previous workspace cannot replace the visible workspace count", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const first = deferred();
  state.notifications = {
    fetchUnreadNotificationsCount: (slug) =>
      slug === "first"
        ? first.promise
        : Promise.resolve({ total_unread_notifications_count: 8, mention_unread_notifications_count: 0 }),
  };
  await fixture.render(React.createElement(NotificationAppSidebarOption, { workspaceSlug: "first" }));
  await fixture.render(React.createElement(NotificationAppSidebarOption, { workspaceSlug: "second" }));
  assert.equal(fixture.environment.container.textContent, "8");
  await act(async () => first.resolve({ total_unread_notifications_count: 3, mention_unread_notifications_count: 0 }));
  await settleQuery();
  assert.equal(fixture.environment.container.textContent, "8");
  assert.equal(fixture.client.getQueryData(notificationKeys.count("second")).total_unread_notifications_count, 8);
});

await test("a failed mark-read request restores only its workspace counts after navigation", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const request = deferred();
  state.notifications = {
    fetchUnreadNotificationsCount: async () => ({
      total_unread_notifications_count: 8,
      mention_unread_notifications_count: 2,
    }),
    markNotificationAsRead: () => request.promise,
  };
  await fixture.client.fetchQuery(notificationCountOptions("first"));
  await fixture.client.fetchQuery(notificationCountOptions("second"));
  const notification = { id: "n1", is_mentioned_notification: true, read_at: null };
  const mutation = new MutationObserver(
    fixture.client,
    notificationMutationOptions(fixture.client, "first", notification.id)
  );
  const pending = mutation.mutate({ notification, field: "read_at", value: "2026-01-01" });
  await settleQuery();
  assert.deepEqual(fixture.client.getQueryData(notificationKeys.count("first")), {
    total_unread_notifications_count: 8,
    mention_unread_notifications_count: 1,
  });
  await fixture.render(React.createElement(NotificationAppSidebarOption, { workspaceSlug: "second" }));
  request.reject(new Error("offline"));
  await assert.rejects(pending, /offline/);
  for (const slug of ["first", "second"]) {
    assert.deepEqual(fixture.client.getQueryData(notificationKeys.count(slug)), {
      total_unread_notifications_count: 8,
      mention_unread_notifications_count: 2,
    });
  }
});

await test("GitHub sync replaces an in-flight initial repository query and ignores its late result", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const initial = deferred();
  let synced = false;
  state.getRepository = () =>
    synced ? Promise.resolve([{ repo_detail: { owner: "acme", name: "repo" } }]) : initial.promise;
  state.syncRepository = async () => {
    synced = true;
  };
  await fixture.render(
    React.createElement(IntegrationCard, {
      integration: { id: "github", integration_detail: { provider: "github", title: "GitHub" } },
    })
  );
  assert.equal(fixture.environment.container.querySelector("button").textContent, "Select Repository");
  await act(async () => fixture.environment.container.querySelector("button").click());
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("button").textContent, "acme/repo");
  initial.resolve([]);
  await settleQuery();
  assert.equal(fixture.environment.container.querySelector("button").textContent, "acme/repo");
});

await test("notification pagination retains the requested workspace cursor after a late response", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const first = deferred();
  const requests = [];
  let pendingFirst = true;
  state.notifications = {
    fetchUnreadNotificationsCount: async () => ({
      total_unread_notifications_count: 1,
      mention_unread_notifications_count: 0,
    }),
    fetchNotifications: (slug, params) => {
      requests.push({ slug, cursor: params.cursor });
      return slug === "A" && pendingFirst ? first.promise : Promise.resolve(notificationPage(slug));
    },
  };
  await fixture.render(notificationScreen("A"));
  await fixture.render(notificationScreen("B"));
  assert.equal(fixture.environment.container.textContent, "B-notification");
  await act(async () => first.resolve(notificationPage("A")));
  await settleQuery();
  assert.equal(fixture.environment.container.textContent, "B-notification");
  pendingFirst = false;
  await fixture.render(notificationScreen("A"));
  await act(async () => state.view.list.fetchNextPage());
  assert.deepEqual(requests.at(-1), { slug: "A", cursor: "A-next" });
});

await test("changing notification filters discards the superseded list response", async (t) => {
  const fixture = await createQueryRouterFixture(t);
  const first = deferred();
  state.notifications = {
    fetchUnreadNotificationsCount: async () => ({
      total_unread_notifications_count: 1,
      mention_unread_notifications_count: 0,
    }),
    fetchNotifications: (_slug, params) =>
      params.archived
        ? Promise.resolve({ results: [{ id: "archived", workspace: "A", archived_at: "2026-01-01" }] })
        : first.promise,
  };
  await fixture.render(notificationScreen("A"));
  await act(async () => state.view.updateFilters("archived", true));
  await settleQuery();
  assert.equal(fixture.environment.container.textContent, "archived");
  await act(async () => first.resolve({ results: [{ id: "obsolete", workspace: "A" }] }));
  await settleQuery();
  assert.equal(fixture.environment.container.textContent, "archived");
});
