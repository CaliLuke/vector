import assert from "node:assert/strict";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

const { parentCatalogueFixture } = await import("./helpers/parent-catalogue-fixture.mjs");
const { IssueService } = await import("../../apps/web/core/services/issue/issue.service.ts");
const { ModuleService } = await import("../../apps/web/core/services/module.service.ts");
const { currentIssueListMutationOwner, issueListMutation } =
  await import("../../apps/web/core/queries/issue-list-mutations.ts");

async function verifyDomain(domain) {
  await test(`${domain} assignment requires verified native project access before its transport call`, async (t) => {
    const { client, rolesKey } = await parentCatalogueFixture(t);
    client
      .getQueryCache()
      .find({ queryKey: rolesKey, exact: true })
      .setState({
        status: "error",
        error: new Error("permission refresh failed"),
      });
    let writes = 0;
    const service = domain === "cycle" ? IssueService.prototype : ModuleService.prototype;
    const method = domain === "cycle" ? "addIssueToCycle" : "addIssuesToModule";
    t.mock.method(service, method, async () => {
      writes++;
    });
    const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true);
    const mutation =
      domain === "cycle"
        ? { kind: "cycle-add", cycleId: "c1", issueIds: ["issue"] }
        : { kind: "module-add", moduleId: "m1", issueIds: ["issue"] };

    await assert.rejects(issueListMutation(client, owner, mutation), /Work item access changed/);
    assert.equal(writes, 0);
  });

  await test(`${domain} assignment keeps its accepted result when ancillary refresh fails`, async (t) => {
    const { client } = await parentCatalogueFixture(t);
    const calls = [];
    const errors = t.mock.method(console, "error", () => {});
    const service = domain === "cycle" ? IssueService.prototype : ModuleService.prototype;
    const method = domain === "cycle" ? "addIssueToCycle" : "addIssuesToModule";
    t.mock.method(service, method, async (...args) => {
      calls.push(args);
    });
    const owner = currentIssueListMutationOwner(client, "acme", "p1", "issues", () => true, {
      isCurrent: () => true,
      refreshAssignments: async (kind, ids) => {
        calls.push(["refresh", kind, [...ids]]);
        throw new Error("parent metadata offline");
      },
    });
    const mutation =
      domain === "cycle"
        ? { kind: "cycle-add", cycleId: "c1", issueIds: ["issue"] }
        : { kind: "module-add", moduleId: "m1", issueIds: ["issue"] };

    await issueListMutation(client, owner, mutation);
    await new Promise((resolve) => setImmediate(resolve));

    assert.deepEqual(
      calls,
      domain === "cycle"
        ? [
            ["acme", "p1", "c1", { issues: ["issue"] }],
            ["refresh", "cycle", ["c1"]],
          ]
        : [
            ["acme", "p1", "m1", { issues: ["issue"] }],
            ["refresh", "module", ["m1"]],
          ]
    );
    assert.equal(errors.mock.callCount(), 1, "failed ancillary refresh remains observable");
  });
}

await verifyDomain("cycle");
await verifyDomain("module");
