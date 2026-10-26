import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const stubs = {
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/ui": "export const LinearProgressIndicator = () => null; export const Loader = () => null;",
  "@/components/empty-state/simple-empty-state-root": "export const SimpleEmptyState = () => null;",
};
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith("?url")) return createModuleStub('export default "asset";');
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});

await test("cycle analytics uses the API's canonical route for both metrics", async () => {
  const { CycleService } = await import("../../apps/web/core/services/cycle.service.ts");
  const service = new CycleService();
  const requests = [];
  service.get = async (path) => {
    requests.push(path);
    return { data: { total: 4 } };
  };
  assert.deepEqual(await service.workspaceActiveCyclesAnalytics("acme", "project", "cycle", "issues"), { total: 4 });
  await service.workspaceActiveCyclesAnalytics("acme", "project", "cycle", "points");
  assert.deepEqual(requests, [
    "/api/workspaces/acme/projects/project/cycles/cycle/analytics/?type=issues",
    "/api/workspaces/acme/projects/project/cycles/cycle/analytics/?type=points",
  ]);
});

await test("active cycle progress renders stable groups and applies the selected state filter", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const errors = [];
  t.mock.method(console, "error", (...args) => errors.push(args.join(" ")));
  const { ActiveCycleProgress } = await import("../../apps/web/core/components/cycles/active-cycle/progress.tsx");
  const filters = [];
  await environment.render(
    React.createElement(ActiveCycleProgress, {
      workspaceSlug: "acme",
      projectId: "project",
      cycle: {
        total_issues: 4,
        completed_issues: 1,
        cancelled_issues: 0,
        started_issues: 1,
        unstarted_issues: 0,
        backlog_issues: 2,
      },
      handleFiltersUpdate: (conditions) => filters.push(conditions),
    })
  );
  assert.match(environment.container.textContent, /backlog2 Work items/);
  const backlog = [...environment.container.querySelectorAll("span")].find(
    (element) => element.textContent === "backlog"
  );
  await act(async () => backlog.click());
  assert.deepEqual(filters, [[{ property: "state_group", operator: "in", value: ["backlog"] }]]);
  assert.deepEqual(errors, []);
});
