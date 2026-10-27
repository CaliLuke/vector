import assert from "node:assert/strict";
import test from "node:test";
import { registerHooks } from "node:module";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
const { MilestoneActivityMessage, milestoneActivityContent } =
  await import("../../apps/web/core/components/core/milestone-activity-message.tsx");
const { milestoneActivityDetails } = await import("../../apps/web/core/components/core/activity-milestone.tsx");

await test("milestone history renders addition and deletion from title snapshots without a target link", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    milestoneActivityDetails.milestone.message({ new_value: "Original release", old_value: null }, false)
  );
  assert.equal(document.body.textContent, "set the milestone to Original release.");
  assert.equal(document.querySelector("a"), null);
  await env.render(
    React.createElement(MilestoneActivityMessage, { activity: { new_value: null, old_value: "Deleted release" } })
  );
  assert.equal(document.body.textContent, "removed the milestone Deleted release.");
  assert.equal(document.querySelector("a"), null);
});

await test("milestone title snapshots are plain text in activity and notification content", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const title = '<img src="bad" onerror="bad()">';
  await env.render(React.createElement(MilestoneActivityMessage, { activity: { new_value: title } }));
  assert.equal(document.body.textContent, `set the milestone to ${title}.`);
  assert.equal(document.querySelector("img"), null);
  assert.deepEqual(milestoneActivityContent("", title), {
    action: "removed the milestone",
    value: title,
    showConnector: false,
  });
});

await test("notification renderer routes milestone additions and removals without treating titles as markup", async (t) => {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      return resolvePackageStub(
        specifier,
        {
          "@/components/editor/lite-text":
            "export function LiteTextEditor() { throw new Error('milestone title must not use rich text'); }",
        },
        context,
        nextResolve
      );
    },
  });
  const { NotificationContent } =
    await import("../../apps/web/core/components/workspace-notifications/sidebar/notification-card/content.tsx");
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const notification = {
    triggered_by_details: { display_name: "Writer" },
    data: { issue_activity: { field: "milestone", verb: "updated", new_value: "None", old_value: "" } },
  };
  await env.render(
    React.createElement(NotificationContent, {
      notification,
      workspaceId: "workspace",
      workspaceSlug: "workspace",
      projectId: "project",
    })
  );
  assert.equal(document.body.textContent, "Writer set the milestone to None.");
  notification.data.issue_activity = {
    field: "milestone",
    verb: "updated",
    new_value: "",
    old_value: "<b>Deleted title</b>",
  };
  await env.render(
    React.createElement(NotificationContent, {
      notification,
      workspaceId: "workspace",
      workspaceSlug: "workspace",
      projectId: "project",
    })
  );
  assert.equal(document.body.textContent, "Writer removed the milestone <b>Deleted title</b>.");
  assert.equal(document.querySelector("b"), null);
});

await test("work-item property activity routes milestone history to its explicit renderer", async (t) => {
  registerHooks({
    resolve(specifier, context, nextResolve) {
      return resolvePackageStub(
        specifier,
        {
          "./helpers/activity-block": "export const IssueActivityBlockComponent = ({children}) => children;",
        },
        context,
        nextResolve
      );
    },
  });
  const { IssuePropertyActivity } =
    await import("../../apps/web/core/components/issues/issue-detail/issue-activity/activity/actions/property.tsx");
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  await env.render(
    React.createElement(IssuePropertyActivity, {
      activity: { field: "milestone", old_value: "Removed release", new_value: null },
      ends: "bottom",
    })
  );
  assert.equal(document.body.textContent, "removed the milestone Removed release.");
});
