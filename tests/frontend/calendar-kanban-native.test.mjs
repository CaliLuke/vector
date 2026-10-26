import assert from "node:assert/strict";
import test from "node:test";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

registerWebModuleHooks();

const {
  CalendarStateProvider,
  KanbanDragStateProvider,
  canDragDropForGrouping,
  useCalendarState,
  useKanbanDragState,
  useKanbanDragOwner,
} = await import("../../apps/web/core/providers/issue-layout-ui-state.tsx");
const { useCurrentUserStartOfWeek } = await import("../../apps/web/core/hooks/use-current-user-profile.ts");
const { currentUserOptions } = await import("../../apps/web/core/queries/identity.ts");
const { userProfileOptions } = await import("../../apps/web/core/queries/user-profile.ts");
const { useState } = React;
const { getCalendarState, shiftCalendarDate } =
  await import("../../apps/web/core/components/issues/issue-layouts/calendar/calendar-state.ts");

const initialDates = {
  activeMonthDate: new Date(2026, 9, 1),
  activeWeekDate: new Date(2026, 9, 24),
};

function createClient(startOfWeek = 0) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: Infinity } } });
  client.setQueryData(currentUserOptions().queryKey, { id: "viewer" });
  client.setQueryData(userProfileOptions("viewer").queryKey, { user: "viewer", start_of_the_week: startOfWeek });
  return client;
}

function CalendarProbe({ routeId }) {
  const state = useCalendarState();
  const startOfWeek = useCurrentUserStartOfWeek();
  const calendar = getCalendarState(state.activeMonthDate, state.activeWeekDate, startOfWeek);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { "data-testid": "month" }, state.activeMonthDate.toISOString().slice(0, 10)),
    React.createElement(
      "output",
      { "data-testid": "week" },
      `${calendar.weekDateRange.startDate}/${calendar.weekDateRange.endDate}`
    ),
    React.createElement("output", { "data-testid": "route" }, routeId),
    React.createElement(
      "button",
      {
        onClick: () =>
          state.updateCalendarDates({ activeMonthDate: shiftCalendarDate(state.activeMonthDate, 1, "month") }),
      },
      "Next month"
    ),
    React.createElement(
      "button",
      {
        onClick: () =>
          state.updateCalendarDates({ activeWeekDate: shiftCalendarDate(state.activeWeekDate, 1, "week") }),
      },
      "Next week"
    )
  );
}

function DragOwner({ name }) {
  const { isDragging } = useKanbanDragState();
  const { beginDrag, endDrag } = useKanbanDragOwner();
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("output", { "data-testid": `dragging-${name}` }, String(isDragging)),
    React.createElement("button", { "data-testid": `start-${name}`, onClick: beginDrag }, `Start ${name}`),
    React.createElement("button", { "data-testid": `end-${name}`, onClick: endDrag }, `End ${name}`)
  );
}

function DragBoardProbe() {
  const [showA, setShowA] = useState(true);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement(
      "button",
      { "data-testid": "toggle-a", onClick: () => setShowA((current) => !current) },
      "Toggle A"
    ),
    showA ? React.createElement(DragOwner, { name: "A" }) : null,
    React.createElement(DragOwner, { name: "B" })
  );
}

await test("calendar navigation persists across routes and derives weeks from the current profile preference", async (t) => {
  const environment = await createReactDomTestEnvironment();
  const client = createClient();
  t.after(() => client.clear());
  t.after(() => environment.unmount());

  const render = async (routeId = "calendar") =>
    environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(CalendarStateProvider, { initialDates }, React.createElement(CalendarProbe, { routeId }))
      )
    );

  await render();
  assert.equal(environment.container.querySelector('[data-testid="week"]').textContent, "2026-10-18/2026-10-24");
  await React.act(async () => environment.container.querySelector("button").click());
  assert.equal(environment.container.querySelector('[data-testid="month"]').textContent, "2026-11-01");
  await React.act(async () => environment.container.querySelectorAll("button")[1].click());
  assert.equal(environment.container.querySelector('[data-testid="week"]').textContent, "2026-10-25/2026-10-31");

  await React.act(async () => {
    client.setQueryData(userProfileOptions("viewer").queryKey, { user: "viewer", start_of_the_week: 1 });
  });
  await render("another-route");
  assert.equal(environment.container.querySelector('[data-testid="month"]').textContent, "2026-11-01");
  assert.equal(environment.container.querySelector('[data-testid="route"]').textContent, "another-route");
  assert.equal(environment.container.querySelector('[data-testid="week"]').textContent, "2026-10-26/2026-11-01");
});

await test("calendar state resets when the session-keyed provider subtree remounts", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const client = createClient();
  t.after(() => client.clear());
  const renderSession = async (generation) =>
    environment.render(
      React.createElement(
        QueryClientProvider,
        { client },
        React.createElement(
          CalendarStateProvider,
          { key: generation, initialDates },
          React.createElement(CalendarProbe, { routeId: "calendar" })
        )
      )
    );

  await renderSession(1);
  await React.act(async () => environment.container.querySelector("button").click());
  assert.equal(environment.container.querySelector('[data-testid="month"]').textContent, "2026-11-01");
  await renderSession(2);
  assert.equal(environment.container.querySelector('[data-testid="month"]').textContent, "2026-10-01");
});

await test("Kanban drag ownership clears on active-card teardown without clearing another card's drag", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());

  const render = () =>
    environment.render(React.createElement(KanbanDragStateProvider, null, React.createElement(DragBoardProbe)));
  await render();
  const dragState = () => environment.container.querySelector('[data-testid="dragging-B"]').textContent;
  await React.act(async () => environment.container.querySelector('[data-testid="start-A"]').click());
  assert.equal(dragState(), "true", "the board shows a drag while card A owns it");
  await React.act(async () => environment.container.querySelector('[data-testid="toggle-a"]').click());
  assert.equal(dragState(), "false", "unmounting active card A clears the drag overlay");

  await React.act(async () => environment.container.querySelector('[data-testid="toggle-a"]').click());
  await React.act(async () => environment.container.querySelector('[data-testid="start-B"]').click());
  await React.act(async () => environment.container.querySelector('[data-testid="toggle-a"]').click());
  assert.equal(dragState(), "true", "unmounting card A does not clear card B's active drag");
  await React.act(async () => environment.container.querySelector('[data-testid="end-B"]').click());
  assert.equal(dragState(), "false", "card B can release its own drag");

  await environment.unmount();

  const remounted = await createReactDomTestEnvironment();
  t.after(() => remounted.unmount());
  await remounted.render(
    React.createElement(KanbanDragStateProvider, null, React.createElement(DragOwner, { name: "C" }))
  );
  assert.equal(remounted.container.querySelector('[data-testid="dragging-C"]').textContent, "false");
  assert.equal(canDragDropForGrouping("state", "priority"), true);
  assert.equal(canDragDropForGrouping("type_id", undefined), false);
  assert.equal(canDragDropForGrouping("state", "type_id"), false);
});
