import assert from "node:assert/strict";
import test from "node:test";
import { React, act, createPermissionUiFixture, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { waitForReactCondition } from "./helpers/react-dom.mjs";

globalThis.calendarMenuTest = { peeks: [] };
registerPermissionUiStubs("/calendar/issue-block.tsx", {
  "@/hooks/use-projects": 'export const useWorkspaceProjectDetails=()=>({projects:[{id:"p1",identifier:"PRJ"}]});',
  "@/hooks/use-project-states": 'export const useRouteProjectStates=()=>({states:[{id:"active",color:"#123456"}]});',
  "@/hooks/use-issue-list-type": 'export const useIssueListType=()=>"project";',
  "@/hooks/use-issue-filter-preferences":
    "export const useIssueFilterPreferences=()=>({issueFilters:{displayProperties:{key:false}}});",
  "@/providers/issue-detail-ui-state": "export const useIssueDetailUiState=()=>({getIsIssuePeeked:()=>false});",
  "@/hooks/use-issue-peek-overview-redirection":
    "export default ()=>({handleRedirection:(...args)=>globalThis.calendarMenuTest.peeks.push(args)});",
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
  "@/components/issues/issue-detail/issue-identifier": "export const IssueIdentifier=()=>null;",
  "../preview-popover-panel": "export const IssuePreviewPopoverPanel=()=>null;",
});

await test("calendar item actions open by keyboard, restore their trigger and never open the work item", async (t) => {
  const view = await createPermissionUiFixture(t, {
    path: "/acme/projects/p1/issues/",
    workspaceRole: 15,
    projectRoles: { p1: 20 },
  });
  const { CalendarIssueBlock } =
    await import("../../apps/web/core/components/issues/issue-layouts/calendar/issue-block.tsx");
  const { QuickActionMenu } = await import("../../apps/web/core/components/common/quick-action-menu.tsx");
  let actions = 0;
  const quickActions = ({ parentRef, customActionButton, placement }) =>
    React.createElement(QuickActionMenu, {
      parentRef,
      items: [
        {
          key: "edit",
          title: "Edit work item",
          action: () => {
            actions++;
          },
        },
      ],
      customMenuProps: { customButton: customActionButton, placement, closeOnSelect: true },
    });
  const issue = { id: "item", project_id: "p1", sequence_id: 1, name: "Calendar work", state_id: "active" };
  const card = React.createRef();
  await view.render(React.createElement(CalendarIssueBlock, { issue, quickActions, ref: card }));
  const trigger = view.environment.container.querySelector('button[aria-label="Actions"]');
  assert.ok(trigger);
  await act(async () => {
    trigger.focus();
    trigger.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown", bubbles: true, cancelable: true }));
  });
  await waitForReactCondition(() => document.querySelector('[role="menuitem"]'), "keyboard opens the native menu");
  assert.equal(trigger.getAttribute("aria-expanded"), "true");
  await act(async () =>
    document.activeElement.dispatchEvent(
      new KeyboardEvent("keydown", { key: "Escape", bubbles: true, cancelable: true })
    )
  );
  await waitForReactCondition(() => trigger.getAttribute("aria-expanded") === "false", "Escape closes the menu");
  assert.ok(document.activeElement === trigger, "focus returns to the actual opening control");
  await act(async () => trigger.click());
  await waitForReactCondition(() => document.querySelector('[role="menuitem"]'), "menu reopens");
  await act(async () => document.querySelector('[role="menuitem"]').click());
  assert.equal(actions, 1);
  assert.deepEqual(globalThis.calendarMenuTest.peeks, []);
  await act(async () => card.current.click());
  assert.equal(globalThis.calendarMenuTest.peeks.length, 1);
  assert.equal(globalThis.calendarMenuTest.peeks[0][1], issue);
});
