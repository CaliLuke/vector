import assert from "node:assert/strict";
import { after, afterEach, test } from "node:test";
import { createReactDomTestEnvironment, React, webRequire } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks({ stubStoreContext: true });

const environment = await createReactDomTestEnvironment();

const { action, observable } = webRequire("mobx");
const { observer } = webRequire("mobx-react");
const { StoreContext } = await import("./helpers/work-item-store-context.mjs");
const { useWorkItemFilterLifecycle } =
  await import("../../apps/web/core/hooks/store/work-item-filters/use-work-item-filter-instance.ts");

const keyFor = (entityType, entityId) => `${entityType}:${entityId}`;

afterEach(async () => {
  await environment.render(null);
});

after(async () => {
  await environment.unmount();
});

function createFilterStore() {
  const filters = observable.map();
  let getOrCreateCalls = 0;

  return {
    filters,
    getOrCreateCalls: () => getOrCreateCalls,
    getFilter: (entityType, entityId) => filters.get(keyFor(entityType, entityId)),
    getOrCreateFilter: action((options) => {
      getOrCreateCalls++;
      const key = keyFor(options.entityType, options.entityId);
      if (!filters.has(key)) filters.set(key, { key, visible: options.showOnMount, configManager: {} });
      else if (options.showOnMount !== undefined) filters.get(key).visible = options.showOnMount;
      return filters.get(key);
    }),
    deleteFilter: action((entityType, entityId) => filters.delete(keyFor(entityType, entityId))),
    setVisible: action((entityType, entityId, visible) => {
      const filter = filters.get(keyFor(entityType, entityId));
      if (filter) filter.visible = visible;
    }),
  };
}

const onExpressionChange = () => {};

function renderFilterProbe(store, entityId, onRender) {
  const initialExpression = {};
  const Probe = observer(() => {
    const filter = useWorkItemFilterLifecycle({
      entityType: "project",
      entityId,
      initialExpression,
      onExpressionChange,
      expressionOptions: {},
      showOnMount: false,
    });
    onRender(store.filters.size);
    return React.createElement("span", { "data-filter-key": filter?.key ?? "pending" });
  });

  return React.createElement(StoreContext.Provider, { value: { workItemFilters: store } }, React.createElement(Probe));
}

test("filter instances are created after render and cleaned up on unmount, including StrictMode effects", async () => {
  const store = createFilterStore();
  const renderedSizes = [];

  const probe = renderFilterProbe(store, "alpha", (size) => renderedSizes.push(size));
  const strictTree = React.createElement(React.StrictMode, null, probe);
  await environment.render(strictTree);

  assert.ok(renderedSizes.includes(0), "the first render should see an untouched store");
  assert.equal(store.filters.size, 1);
  assert.equal(
    environment.container.querySelector("[data-filter-key]").getAttribute("data-filter-key"),
    "project:alpha"
  );

  const callsAfterMount = store.getOrCreateCalls();
  store.setVisible("project", "alpha", true);
  await environment.render(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(StoreContext.Provider, { value: { workItemFilters: store } }, probe.props.children)
    )
  );
  assert.equal(
    store.getOrCreateCalls(),
    callsAfterMount,
    "fresh wrapper objects in expressionOptions on an unrelated context render must not reset the filter"
  );
  assert.equal(store.getFilter("project", "alpha").visible, true, "the user's current visibility must be preserved");

  await environment.render(null);
  assert.equal(store.filters.size, 0, "unmount should clean the entity's filter instance");
});

test("changing entities removes the previous filter and creates the new one", async () => {
  const store = createFilterStore();
  let entityId = "alpha";
  const Probe = observer(() => {
    const filter = useWorkItemFilterLifecycle({
      entityType: "project",
      entityId,
      initialExpression: {},
      onExpressionChange: () => {},
      expressionOptions: {},
      showOnMount: false,
    });
    return React.createElement("span", { "data-filter-key": filter?.key ?? "pending" });
  });
  const render = () =>
    environment.render(
      React.createElement(StoreContext.Provider, { value: { workItemFilters: store } }, React.createElement(Probe))
    );

  await render();
  assert.equal(store.filters.has("project:alpha"), true);

  entityId = "beta";
  await render();

  assert.equal(store.filters.has("project:alpha"), false);
  assert.equal(store.filters.has("project:beta"), true);
  assert.equal(
    environment.container.querySelector("[data-filter-key]").getAttribute("data-filter-key"),
    "project:beta"
  );

  await environment.render(null);
});

test("CountChip forwards its DOM ref, ARIA attributes, and event handlers", async () => {
  const { CountChip } = await import("../../apps/web/core/components/common/count-chip.tsx");
  const ref = React.createRef();
  let clicks = 0;

  await environment.render(
    React.createElement(CountChip, {
      count: 7,
      ref,
      "aria-label": "Seven unread items",
      title: "Unread items",
      onClick: () => clicks++,
    })
  );

  const chip = environment.container.querySelector("[aria-label='Seven unread items']");
  assert.ok(chip instanceof HTMLElement);
  assert.equal(ref.current, chip);
  assert.equal(chip.title, "Unread items");
  chip.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  assert.equal(clicks, 1);

  await environment.render(null);
});
