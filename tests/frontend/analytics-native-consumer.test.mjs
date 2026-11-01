import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { createModuleStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, webRequire } from "./helpers/react.mjs";
import { pathToFileURL } from "node:url";
import { createRouterFixture } from "./helpers/router.mjs";
import { waitForDom } from "./helpers/wait-for-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

globalThis.analyticsNativeConsumerReact = React;
globalThis.analyticsNativeConsumerDownloads = [];
const sources = {
  "export-to-csv": `export {generateCsv,mkConfig} from ${JSON.stringify(pathToFileURL(webRequire.resolve("export-to-csv")).href)};export const download=config=>csv=>globalThis.analyticsNativeConsumerDownloads.push({config,csv});`,
  "@plane/i18n": `export const useTranslation=()=>({t:(key)=>key});`,
  "@plane/propel/button": `const React=globalThis.analyticsNativeConsumerReact;export const Button=({children,onClick,type="button"})=>React.createElement("button",{type,onClick},children);`,
  "@plane/propel/emoji-icon-picker": `const React=globalThis.analyticsNativeConsumerReact;export const Logo=()=>React.createElement("span");`,
  "@plane/propel/icons": `const React=globalThis.analyticsNativeConsumerReact;export const ProjectIcon=()=>React.createElement("span");export const SearchIcon=()=>React.createElement("span");export const CloseIcon=()=>React.createElement("span");`,
  "@plane/propel/table": `const React=globalThis.analyticsNativeConsumerReact;export const Table=({children,...props})=>React.createElement("table",props,children);export const TableBody=({children,...props})=>React.createElement("tbody",props,children);export const TableCell=({children,...props})=>React.createElement("td",props,children);export const TableHead=({children,...props})=>React.createElement("th",props,children);export const TableHeader=({children,...props})=>React.createElement("thead",props,children);export const TableRow=({children,...props})=>React.createElement("tr",props,children);`,
  "@plane/propel/empty-state": `const React=globalThis.analyticsNativeConsumerReact;export const EmptyStateCompact=()=>React.createElement("div",{"data-testid":"empty-state"});`,
  "@plane/propel/charts/area-chart": `const React=globalThis.analyticsNativeConsumerReact;export const AreaChart=({data})=>React.createElement("output",{"data-testid":"guest-chart"},String(data.length));`,
  "@plane/ui": `const React=globalThis.analyticsNativeConsumerReact;export const Avatar=()=>React.createElement("span");export const Loader=()=>React.createElement("span");`,
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (Object.hasOwn(sources, specifier)) return createModuleStub(sources[specifier]);
    if (specifier === "./insight-card" && context.parentURL?.includes("/components/analytics/total-insights.tsx"))
      return createModuleStub(
        `const React=globalThis.analyticsNativeConsumerReact;export default function InsightCard({data}){return React.createElement("output",{"data-testid":"insight-total"},String(data??""))}`
      );
    if (specifier === "./loader" && context.parentURL?.includes("/components/analytics/insight-table/root.tsx"))
      return createModuleStub(`export const TableLoader=()=>null;`);
    return nextResolve(specifier, context);
  },
});

const { QueryClient, QueryClientProvider } = await import("@tanstack/react-query");
const { identityKeys } = await import("../../apps/web/core/queries/identity.ts");
const { APIService } = await import("../../apps/web/core/services/api.service.ts");
const { AnalyticsStateProvider } = await import("../../apps/web/core/providers/analytics-state-provider.tsx");
const TotalInsights = (await import("../../apps/web/core/components/analytics/total-insights.tsx")).default;
const WorkItemsInsightTable = (
  await import("../../apps/web/core/components/analytics/work-items/workitems-insight-table.tsx")
).default;
const CreatedVsResolved = (await import("../../apps/web/core/components/analytics/work-items/created-vs-resolved.tsx"))
  .default;

const ACCOUNT = "analytics-consumer-account";
const WORKSPACE = "analytics-consumer-workspace";
const MEMBERSHIP_REVISION = "membership-v1";
const PROJECT = "analytics-project";
const transport = {
  projectRoles: { [PROJECT]: 15 },
  failAnalyticsReads: false,
  requests: [],
};
function mockAnalyticsTransport(t) {
  t.mock.method(APIService.prototype, "get", async function (path) {
    transport.requests.push(path);
    if (path.endsWith("/workspace-members/me/"))
      return { data: { is_active: true, role: 15, updated_at: MEMBERSHIP_REVISION } };
    if (path.endsWith("/project-roles/")) return { data: transport.projectRoles };
    if (path.endsWith("/projects/details/")) return { data: [{ id: PROJECT, name: "Analytics Project" }] };
    if (path.includes("/advance-analytics")) {
      if (transport.failAnalyticsReads) throw { response: { data: new Error("analytics transport failed") } };
      if (path.includes("advance-analytics-stats"))
        return {
          data: transport.stats ?? [
            {
              project_id: PROJECT,
              project__name: "Analytics Project",
              backlog_work_items: 1,
              started_work_items: 2,
              un_started_work_items: 3,
              completed_work_items: 42,
              cancelled_work_items: 0,
            },
          ],
        };
      if (path.includes("advance-analytics-charts")) return { data: { data: [{ key: "2026-10-25", count: 3 }] } };
      return { data: { completed_work_items: 42 } };
    }
    throw new Error(`Unexpected analytics integration request: ${path}`);
  });
}

test.after(() => {
  delete globalThis.analyticsNativeConsumerReact;
});

function App({ client, projectId, isPeekView = false, showChart = false, showPeekTable = false }) {
  const [route] = React.useState(() => createRouterFixture(`/${WORKSPACE}/analytics/`));
  return React.createElement(
    QueryClientProvider,
    { client },
    React.createElement(
      route.Provider,
      null,
      React.createElement(
        AnalyticsStateProvider,
        { workspaceSlug: WORKSPACE, projectId, isPeekView, isActive: true, instanceKey: "native-consumer" },
        React.createElement(
          React.Fragment,
          null,
          React.createElement(TotalInsights, { analyticsType: "work-items", peekView: isPeekView }),
          showChart ? React.createElement(CreatedVsResolved) : null,
          !isPeekView || showPeekTable ? React.createElement(WorkItemsInsightTable) : null
        )
      )
    )
  );
}

function newClient() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, refetchOnWindowFocus: false } } });
  client.setQueryData(identityKeys.currentUser(), { id: ACCOUNT });
  return client;
}

void test("real permission hooks load totals and export, then a failed refresh hides retained results", async (t) => {
  mockAnalyticsTransport(t);
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = newClient();
  transport.projectRoles = { [PROJECT]: 15 };
  transport.failAnalyticsReads = false;
  transport.requests = [];
  await environment.render(React.createElement(App, { client }));
  await React.act(async () =>
    waitForDom(
      environment.container,
      () =>
        Array.from(environment.container.querySelectorAll('[data-testid="insight-total"]')).some(
          (element) => element.textContent === "42"
        ),
      "Timed out waiting for analytics totals"
    )
  );
  await React.act(async () =>
    waitForDom(
      environment.container,
      () =>
        Array.from(environment.container.querySelectorAll("button")).some((button) =>
          button.textContent?.includes("exporter.csv.short_description")
        ),
      "Timed out waiting for analytics export"
    )
  );
  assert.ok(transport.requests.some((path) => path.endsWith("/workspace-members/me/")));
  assert.ok(transport.requests.some((path) => path.endsWith("/project-roles/")));
  assert.ok(environment.container.querySelector("table tbody tr"));
  globalThis.analyticsNativeConsumerDownloads.length = 0;
  await React.act(async () =>
    [...environment.container.querySelectorAll("button")]
      .find((button) => button.textContent.includes("exporter.csv.short_description"))
      .click()
  );
  const [download] = globalThis.analyticsNativeConsumerDownloads;
  assert.equal(download.config.filename, `${WORKSPACE}-analytics`);
  assert.match(String(download.csv), /Analytics Project/);
  assert.match(String(download.csv), /"1","2","3","42","0"/);

  transport.failAnalyticsReads = true;
  await React.act(async () => {
    await client.invalidateQueries({ queryKey: ["workspace-analytics"] });
    await waitForDom(
      environment.container,
      () => environment.container.querySelector('[role="alert"]'),
      "Timed out waiting for analytics query error"
    );
  });
  assert.ok(
    client
      .getQueryCache()
      .findAll({ queryKey: ["workspace-analytics"] })
      .some((query) => query.state.data)
  );
  assert.equal(environment.container.querySelector('[data-testid="insight-total"]'), null);
  await React.act(async () =>
    waitForDom(
      environment.container,
      () =>
        !Array.from(environment.container.querySelectorAll("button")).some((button) =>
          button.textContent?.includes("exporter.csv.short_description")
        ),
      "Timed out waiting for analytics export to hide after failed refresh"
    )
  );
  assert.equal(
    Array.from(environment.container.querySelectorAll("button")).some((button) =>
      button.textContent?.includes("exporter.csv.short_description")
    ),
    false
  );
  client.clear();
});

void test("real project guest scope allows charts but denies member-only insights", async (t) => {
  mockAnalyticsTransport(t);
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = newClient();
  transport.projectRoles = { [PROJECT]: 5 };
  transport.failAnalyticsReads = false;
  transport.requests = [];
  await environment.render(React.createElement(App, { client, projectId: PROJECT, isPeekView: true, showChart: true }));
  await React.act(async () =>
    waitForDom(
      environment.container,
      () => environment.container.querySelector('[data-testid="guest-chart"]'),
      "Timed out waiting for the guest chart"
    )
  );
  assert.match(environment.container.textContent, /You do not have access to these analytics/);
  assert.ok(transport.requests.some((path) => path.includes(`/projects/${PROJECT}/advance-analytics-charts`)));
  assert.equal(
    transport.requests.some((path) => path.includes("advance-analytics/")),
    false
  );
  client.clear();
});

void test("project analytics renders assignee identities and exports their displayed numeric rows", async (t) => {
  mockAnalyticsTransport(t);
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = newClient();
  t.after(() => client.clear());
  transport.projectRoles = { [PROJECT]: 15 };
  transport.failAnalyticsReads = false;
  const counts = {
    backlog_work_items: 1,
    started_work_items: 2,
    un_started_work_items: 3,
    completed_work_items: 4,
    cancelled_work_items: 0,
  };
  transport.stats = [
    { ...counts, assignee_id: "alice", display_name: "Alice" },
    { ...counts, assignee_id: "bob", display_name: "Bob", avatar_url: "https://example.test/avatar.png" },
    { ...counts, assignee_id: null },
  ];
  t.after(() => {
    transport.stats = undefined;
  });
  await environment.render(
    React.createElement(App, { client, projectId: PROJECT, isPeekView: true, showPeekTable: true })
  );
  await React.act(async () =>
    waitForDom(
      environment.container,
      () => environment.container.textContent.includes("Alice"),
      "Assignee analytics loads"
    )
  );
  assert.match(environment.container.textContent, /Bob/);
  assert.match(environment.container.textContent, /Unassigned/);
  globalThis.analyticsNativeConsumerDownloads.length = 0;
  await React.act(async () =>
    [...environment.container.querySelectorAll("button")]
      .find((button) => button.textContent.includes("exporter.csv.short_description"))
      .click()
  );
  const [download] = globalThis.analyticsNativeConsumerDownloads;
  assert.match(String(download.csv), /Alice/);
  assert.match(String(download.csv), /Bob/);
  assert.match(String(download.csv), /"1","2","3","4","0"/);
});

void test("a missing project membership denies charts without sending an analytics request", async (t) => {
  mockAnalyticsTransport(t);
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const client = newClient();
  t.after(() => client.clear());
  transport.projectRoles = {};
  transport.requests = [];
  transport.failAnalyticsReads = false;
  await environment.render(React.createElement(App, { client, projectId: PROJECT, isPeekView: true, showChart: true }));
  await React.act(async () =>
    waitForDom(
      environment.container,
      () => (environment.container.textContent.match(/You do not have access to these analytics/g) ?? []).length === 2,
      "Both analytics surfaces deny the missing project membership"
    )
  );
  assert.equal(environment.container.querySelector('[data-testid="guest-chart"]'), null);
  assert.equal(
    transport.requests.some((path) => path.includes("advance-analytics")),
    false
  );
});
