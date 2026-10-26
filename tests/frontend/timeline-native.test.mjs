import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
globalThis.timelineNativeReact = React;
globalThis.timelineNativeProfileStart = 0;
function changeStartOfWeek() {
  globalThis.timelineNativeProfileStart = 1;
}
const stubs = {
  "@/components/gantt-chart": `
    import { useGanttChartState } from "@/components/gantt-chart/contexts/chart-state";
    import { blockDateUpdatesAfterDrag } from "@/components/gantt-chart/state";
    import { useGanttResizable } from "@/components/gantt-chart/helpers/blockResizables/use-gantt-resizable";
    const h=globalThis.timelineNativeReact.createElement;
    const React=globalThis.timelineNativeReact;
    export const GanttChartHeader=({currentView,handleChartView,handleToday})=>h("nav",null,h("button",{"data-testid":"change-view",onClick:()=>handleChartView("month")},currentView),h("button",{"data-testid":"today",onClick:handleToday},"today"));
    function ResizableProbe({blockId,containerRef,updateBlockDates}){
      const state=useGanttChartState();
      const block=state.blocksById[blockId];
      const element=React.useRef(null);
      const {handleBlockDrag}=useGanttResizable(block,element,containerRef,updateBlockDates);
      React.useEffect(()=>{if(block?.position&&element.current){element.current.style.marginLeft=block.position.marginLeft+"px";element.current.style.width=block.position.width+"px";}},[block]);
      return h("button",{ref:element,"data-testid":"move-block",onClick:()=>handleBlockDrag({button:0,clientX:250},"move")},"start move");
    }
    export const GanttChartMainContent=({blockIds,title,ganttContainerRef,updateCurrentViewRenderPayload,updateBlockDates})=>{
      const state=useGanttChartState();
      const block=state.blocksById[blockIds[0]];
      return h("section",{"data-chart":title},h("div",{"data-testid":"scroll-container",ref:ganttContainerRef}),
        h("output",{"data-testid":"view-data"},state.viewData?.key ?? "uninitialized"),
        h("output",{"data-testid":"start-day"},String(state.viewData?.data.startDate.getDay() ?? "none")),
        h("output",{"data-testid":"active"},state.activeBlockId ?? "none"),
        h("output",{"data-testid":"dragging"},String(state.isDragging)),
        h("output",{"data-testid":"position"},block?.position ? block.position.marginLeft+":"+block.position.width : "none"),
        h("output",{"data-testid":"dates"},blockDateUpdatesAfterDrag(block,state.viewData,false).map(update=>update.start_date+"/"+update.target_date).join(",")),
        h("button",{"data-testid":"start-drag",onClick:()=>state.setIsDragging(true)},"drag"),
        h("button",{"data-testid":"activate",onClick:()=>state.setActiveBlockId(blockIds[0])},"activate"),
        h("button",{"data-testid":"shift",onClick:()=>state.setBlockPosition(blockIds[0],{marginLeft:block.position.marginLeft+7,width:block.position.width+5})},"shift"),
        h("button",{"data-testid":"extend-left",onClick:()=>updateCurrentViewRenderPayload("left",state.currentView)},"extend left"),
        h(ResizableProbe,{blockId:blockIds[0],containerRef:ganttContainerRef,updateBlockDates})
      );
    };
  `,
  "@/hooks/use-current-user-profile": `export const useCurrentUserProfile=()=>({data:{start_of_the_week:globalThis.timelineNativeProfileStart}});`,
  "@plane/propel/toast": `export const TOAST_TYPE={ERROR:"error"}; export const setToast=message=>{globalThis.timelineNativeToasts.push(message);};`,
};
registerHooks({
  resolve: (specifier, context, nextResolve) => resolvePackageStub(specifier, stubs, context, nextResolve),
});

const { TimeLineTypeContext } = await import("../../apps/web/core/components/gantt-chart/contexts/index.tsx");
const { GANTT_TIMELINE_TYPE } = await import("../../packages/types/dist/index.mjs");
const { ChartViewRoot } = await import("../../apps/web/core/components/gantt-chart/chart/root.tsx");
const { GanttViewPreferenceProvider } = await import("../../apps/web/core/providers/gantt-view-preferences.tsx");
const { blockDateUpdatesAfterDrag, projectGanttBlocks, shiftBlockPositions } =
  await import("../../apps/web/core/components/gantt-chart/state.ts");

const chartData = {
  key: "week",
  i18n_title: "week",
  data: {
    startDate: new Date(2026, 0, 1),
    currentDate: new Date(2026, 0, 8),
    endDate: new Date(2026, 0, 14),
    approxFilterRange: 4,
    dayWidth: 60,
  },
};
const items = [
  {
    id: "issue-a",
    name: "Issue A",
    project_id: "project-a",
    sort_order: 1,
    start_date: "2026-01-02",
    target_date: "2026-01-03",
  },
  { id: "issue-b", name: "Issue B", project_id: "project-a", sort_order: 2, start_date: "2026-01-04" },
];

function Chart(props) {
  return React.createElement(ChartViewRoot, {
    border: false,
    title: props.title,
    loaderTitle: "items",
    blockIds: [props.item.id],
    blockItems: [props.item],
    blockUpdateHandler: () => {},
    updateBlockDates: (updates) => {
      globalThis.timelineNativeUpdates = updates;
      if (globalThis.timelineNativeRejectUpdates) return Promise.reject(new Error("update failed"));
      return Promise.resolve();
    },
    blockToRender: () => null,
    sidebarToRender: () => null,
    enableBlockLeftResize: false,
    enableBlockRightResize: false,
    enableBlockMove: false,
    enableReorder: false,
    enableAddBlock: false,
    enableSelection: false,
    enableDependency: false,
    bottomSpacing: false,
    showAllBlocks: true,
    showToday: false,
  });
}

function TimelineHarness() {
  const [showA, setShowA] = React.useState(true);
  const [showB, setShowB] = React.useState(false);
  const [session, setSession] = React.useState(0);
  const [, setProfileRevision] = React.useState(0);
  return React.createElement(
    React.Fragment,
    null,
    React.createElement("button", { "data-testid": "toggle-a", onClick: () => setShowA((show) => !show) }, "toggle A"),
    React.createElement("button", { "data-testid": "toggle-b", onClick: () => setShowB((show) => !show) }, "toggle B"),
    React.createElement(
      "button",
      { "data-testid": "new-session", onClick: () => setSession((value) => value + 1) },
      "session"
    ),
    React.createElement(
      "button",
      {
        "data-testid": "change-start-week",
        onClick: () => {
          changeStartOfWeek();
          setProfileRevision((value) => value + 1);
        },
      },
      "start week"
    ),
    React.createElement(
      GanttViewPreferenceProvider,
      { key: session },
      React.createElement(
        TimeLineTypeContext.Provider,
        { value: GANTT_TIMELINE_TYPE.ISSUE },
        React.createElement(
          React.Fragment,
          null,
          showA ? React.createElement(Chart, { title: "A", item: items[0] }) : null,
          showB
            ? React.createElement(
                TimeLineTypeContext.Provider,
                { value: GANTT_TIMELINE_TYPE.MODULE },
                React.createElement(Chart, { title: "B", item: items[1] })
              )
            : null
        )
      )
    )
  );
}

function click(environment, id) {
  return React.act(async () => environment.container.querySelector(`[data-testid="${id}"]`).click());
}

await test("Gantt view choice lasts across chart remounts while chart state resets per mount/session", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(TimelineHarness));
  await React.act(async () => {});
  assert.equal(environment.container.querySelector('[data-testid="start-day"]').textContent, "0");
  await click(environment, "change-start-week");
  assert.equal(environment.container.querySelector('[data-testid="start-day"]').textContent, "1");
  await click(environment, "change-view");
  assert.equal(environment.container.querySelector('[data-testid="change-view"]').textContent, "month");
  await React.act(async () => {});
  const originalPosition = environment.container.querySelector('[data-testid="position"]').textContent;
  assert.notEqual(originalPosition, "none");
  await click(environment, "shift");
  assert.notEqual(environment.container.querySelector('[data-testid="position"]').textContent, originalPosition);
  await click(environment, "activate");
  await click(environment, "start-drag");
  assert.equal(environment.container.querySelector('[data-testid="active"]').textContent, "issue-a");
  assert.equal(environment.container.querySelector('[data-testid="dragging"]').textContent, "true");

  await click(environment, "toggle-a");
  await click(environment, "toggle-a");
  assert.equal(environment.container.querySelector('[data-testid="change-view"]').textContent, "month");
  assert.equal(environment.container.querySelector('[data-testid="active"]').textContent, "none");
  assert.equal(environment.container.querySelector('[data-testid="dragging"]').textContent, "false");
  assert.equal(environment.container.querySelector('[data-testid="position"]').textContent, originalPosition);

  await click(environment, "new-session");
  assert.equal(environment.container.querySelector('[data-testid="change-view"]').textContent, "week");
});

await test("concurrent Gantt charts keep active and drag state separate", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(TimelineHarness));
  await click(environment, "toggle-b");
  const charts = environment.container.querySelectorAll("section");
  assert.equal(charts.length, 2);
  const buttons = charts[0].querySelectorAll("button");
  await React.act(async () => buttons[0].click());
  await React.act(async () => buttons[1].click());
  assert.equal(charts[0].querySelector('[data-testid="dragging"]').textContent, "true");
  assert.equal(charts[1].querySelector('[data-testid="dragging"]').textContent, "false");
  assert.equal(charts[0].querySelector('[data-testid="active"]').textContent, "issue-a");
  assert.equal(charts[1].querySelector('[data-testid="active"]').textContent, "none");
});

await test("issue and module charts retain separate selected views", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(TimelineHarness));
  await click(environment, "toggle-b");
  const changeViewButtons = environment.container.querySelectorAll('[data-testid="change-view"]');
  await React.act(async () => changeViewButtons[0].click());
  assert.deepEqual(
    [...environment.container.querySelectorAll('[data-testid="change-view"]')].map((button) => button.textContent),
    ["month", "week"]
  );
  await React.act(async () => changeViewButtons[1].click());
  assert.deepEqual(
    [...environment.container.querySelectorAll('[data-testid="change-view"]')].map((button) => button.textContent),
    ["month", "month"]
  );
});

await test("resizing/moving saves current dates after a left range expansion", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  globalThis.timelineNativeUpdates = undefined;
  globalThis.timelineNativeRejectUpdates = false;
  globalThis.timelineNativeToasts = [];
  await environment.render(React.createElement(TimelineHarness));
  await React.act(async () => environment.container.querySelector('[data-testid="move-block"]').click());
  await React.act(async () => document.dispatchEvent(new MouseEvent("mousemove", { clientX: 310 })));
  assert.equal(environment.container.querySelector('[data-testid="dragging"]').textContent, "true");
  await click(environment, "extend-left");
  await React.act(async () => document.dispatchEvent(new MouseEvent("mouseup")));
  assert.deepEqual(globalThis.timelineNativeUpdates, [
    { id: "issue-a", meta: { project_id: "project-a" }, start_date: "2026-01-03", target_date: "2026-01-04" },
  ]);
});

await test("a rejected block-date update reports the existing error toast and ends dragging", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  globalThis.timelineNativeToasts = [];
  globalThis.timelineNativeRejectUpdates = true;
  await environment.render(React.createElement(TimelineHarness));
  await React.act(async () => environment.container.querySelector('[data-testid="move-block"]').click());
  await React.act(async () => document.dispatchEvent(new MouseEvent("mousemove", { clientX: 310 })));
  await React.act(async () => document.dispatchEvent(new MouseEvent("mouseup")));
  assert.equal(environment.container.querySelector('[data-testid="dragging"]').textContent, "false");
  assert.deepEqual(globalThis.timelineNativeToasts, [
    { type: "error", title: "Error", message: "Something went wrong while updating block dates" },
  ]);
  globalThis.timelineNativeRejectUpdates = false;
});

await test("drag geometry keeps its saved dates when the chart extends left", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(TimelineHarness));
  const initialDates = environment.container.querySelector('[data-testid="dates"]').textContent;
  await click(environment, "start-drag");
  const initialPosition = environment.container.querySelector('[data-testid="position"]').textContent;
  await click(environment, "extend-left");
  assert.equal(environment.container.querySelector('[data-testid="dates"]').textContent, initialDates);
  assert.notEqual(environment.container.querySelector('[data-testid="position"]').textContent, initialPosition);
});

await test("each mounted Gantt chart scrolls its own container", async (t) => {
  t.mock.timers.enable({ apis: ["setTimeout"] });
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  await environment.render(React.createElement(TimelineHarness));
  await click(environment, "toggle-b");
  t.mock.timers.runAll();
  const charts = environment.container.querySelectorAll("section");
  const firstScroll = charts[0].querySelector('[data-testid="scroll-container"]');
  const secondScroll = charts[1].querySelector('[data-testid="scroll-container"]');
  firstScroll.scrollLeft = 0;
  secondScroll.scrollLeft = 0;
  await React.act(async () => environment.container.querySelectorAll('[data-testid="today"]')[0].click());
  t.mock.timers.runAll();
  assert.notEqual(firstScroll.scrollLeft, 0);
  assert.equal(secondScroll.scrollLeft, 0);
});

await test("pure block projection and drag conversion preserve inclusive dates and half-block rules", () => {
  const blocks = projectGanttBlocks(["issue-a", "issue-b"], items, chartData, {});
  assert.deepEqual(blocks["issue-a"].position, { marginLeft: 60, width: 120 });
  assert.deepEqual(blocks["issue-b"].position, { marginLeft: 180, width: 60 });
  assert.deepEqual(blockDateUpdatesAfterDrag(blocks["issue-a"], chartData, false), [
    {
      id: "issue-a",
      meta: { project_id: "project-a" },
      start_date: "2026-01-02",
      target_date: "2026-01-03",
    },
  ]);
  const half = blockDateUpdatesAfterDrag(blocks["issue-b"], chartData, false)[0];
  assert.equal(half.start_date, "2026-01-04");
  assert.equal(half.target_date, undefined);
  assert.deepEqual(shiftBlockPositions({ x: { marginLeft: 4, width: 8 } }, 20), {
    x: { marginLeft: 24, width: 8 },
  });
});
