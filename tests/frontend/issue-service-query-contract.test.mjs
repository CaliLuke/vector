import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { WorkspaceService } = await import("../../apps/web/core/services/workspace.service.ts");

const cases = [
  { queries: undefined, endpoint: "issues", name: "omitted filters" },
  { queries: { expand: true }, endpoint: "issues", name: "a boolean expansion flag" },
  { queries: { expand: "issue_relation,labels" }, endpoint: "issues-detail", name: "expanded relations" },
  { queries: { expand: "issue_relation", group_by: "state" }, endpoint: "issues", name: "grouped relations" },
];
for (const { queries, endpoint, name } of cases) {
  void test(`issue requests with ${name} preserve routing and cancellation`, async (t) => {
    const service = new IssueService();
    const response = { results: [], total_count: 0 };
    const requests = [];
    t.mock.method(service, "get", async (...request) => {
      requests.push(request);
      return { data: response };
    });
    const signal = new AbortController().signal;
    assert.equal(await service.getIssues("workspace", "project", queries, { signal }), response);
    const [url, params, config] = requests.at(-1);
    assert.equal(url, `/api/workspaces/workspace/projects/project/${endpoint}/`);
    assert.equal(params.params, queries);
    assert.equal(config.signal, signal);
  });
}

void test("workspace lists tolerate non-string expansion flags without losing their query parameters", async (t) => {
  const service = new WorkspaceService();
  const queries = { expand: true };
  const requests = [];
  t.mock.method(service, "get", async (...request) => {
    requests.push(request);
    return { data: { results: [] } };
  });
  await service.getViewIssues("workspace", queries);
  assert.equal(requests[0][0], "/api/workspaces/workspace/issues/");
  assert.equal(requests[0][1].params, queries);
});
