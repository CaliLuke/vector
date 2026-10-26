import assert from "node:assert/strict";
import test from "node:test";
import { pathToFileURL } from "node:url";
import path from "node:path";
const fromRoot = (file) => import(pathToFileURL(path.resolve(process.cwd(), file)).href);
const { React, act, createPermissionUiFixture, settleQuery } = await fromRoot(
  "tests/frontend/helpers/permission-ui-fixture.mjs"
);
const { IssueService } = await fromRoot("apps/web/core/services/issue/index.ts");
const { useWorkItemSubItems } = await fromRoot("apps/web/core/hooks/use-work-item-detail.ts");

await test("StrictMode sub-item observers share one Query read and keep the accepted child visible", async (t) => {
  const fixture = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/P1-1",
    userId: "alice",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  t.after(settleQuery);
  let finish;
  let requestCount = 0;
  t.mock.method(IssueService.prototype, "subIssues", () => {
    requestCount++;
    return new Promise((resolve) => {
      finish = resolve;
    });
  });
  function Probe() {
    const query = useWorkItemSubItems("acme", "p1", "parent");
    return React.createElement("output", null, query.subItems?.map((item) => item.id).join(",") ?? "loading");
  }

  await fixture.render(React.createElement(React.StrictMode, null, React.createElement(Probe)));
  await settleQuery();
  assert.equal(requestCount, 1, "StrictMode observers share the exact sub-item Query");
  await act(async () => {
    finish({ sub_issues: [{ id: "child-1", parent_id: "parent", project_id: "p1" }] });
    await settleQuery();
  });
  assert.equal(fixture.environment.container.querySelector("output").textContent, "child-1");
  await fixture.render(React.createElement(React.StrictMode, null, React.createElement(Probe)));
  assert.equal(requestCount, 1, "a parent rerender keeps the accepted query data visible");
  assert.equal(fixture.environment.container.querySelector("output").textContent, "child-1");
});
