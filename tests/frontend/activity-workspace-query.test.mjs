import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createQueryDomFixture } from "./helpers/query-dom.mjs";
import { React } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { seedWorkspaceQuery } from "./helpers/workspace-query-fixture.mjs";

registerWebModuleHooks();
globalThis.activityWorkspaceReact = React;
registerHooks({
  resolve: (specifier, context, nextResolve) =>
    resolvePackageStub(
      specifier,
      {
        "./activity-block": `const React=globalThis.activityWorkspaceReact;export function ActivityBlockComponent({children}){return React.createElement("div",null,children);}`,
      },
      context,
      nextResolve
    ),
});

await test("activity cycle links use the workspace selected from native Query data", async (t) => {
  const { env: environment, client, render } = await createQueryDomFixture(t);
  await seedWorkspaceQuery(client, { id: "workspace-id", slug: "acme" });

  const { ActivityItem } = await import("../../apps/web/core/components/common/activity/activity-item.tsx");
  await render(
    React.createElement(ActivityItem, {
      activity: {
        field: "cycles",
        verb: "added",
        workspace: "workspace-id",
        project: "project-id",
        new_identifier: "cycle-id",
        new_value: "Cycle 7",
      },
    })
  );

  const link = environment.container.querySelector("a");
  assert.ok(link);
  assert.equal(link.getAttribute("href"), "/acme/projects/project-id/cycles/cycle-id");
  assert.equal(link.textContent, "Cycle 7");
});
