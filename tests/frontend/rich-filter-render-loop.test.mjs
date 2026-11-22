import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();

await test("a filter select catalog loads once across parent rerenders", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { useFiltersOperatorConfigs } =
    await import("../../apps/web/core/hooks/rich-filters/use-filters-operator-configs.ts");
  const { MultiSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx");
  let optionsLoads = 0;
  function Probe() {
    const operatorConfigs = useFiltersOperatorConfigs({ workspaceSlug: "acme" });
    const config = React.useMemo(
      () => ({
        type: "multi_select",
        ...operatorConfigs,
        getOptions: async () => {
          optionsLoads++;
          return [{ id: "high", value: "high", label: "High" }];
        },
      }),
      [operatorConfigs]
    );
    const [revision, setRevision] = React.useState(0);
    return React.createElement(
      React.Fragment,
      null,
      React.createElement(MultiSelectFilterValueInput, {
        config,
        condition: { id: "priority", property: "priority", operator: "in", value: ["high"] },
        onChange: () => {},
      }),
      React.createElement("button", { "data-testid": "rerender", onClick: () => setRevision(revision + 1) }, "Refresh")
    );
  }

  await environment.render(React.createElement(Probe));
  await act(async () => new Promise((resolve) => setImmediate(resolve)));
  assert.equal(optionsLoads, 1);
  assert.match(environment.container.textContent, /High/);

  await act(async () => environment.container.querySelector('[data-testid="rerender"]').click());
  assert.equal(optionsLoads, 1, "rerendering the filter config must not restart its option load");
});

await test("a state filter never drops loaded labels between commits", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { getStateFilterConfig } = await import("../../packages/utils/dist/index.js");
  const { MultiSelectFilterValueInput } =
    await import("../../apps/web/core/components/rich-filters/filter-value-input/select/multi.tsx");
  const commits = [];
  function Probe({ todoLabel }) {
    const config = getStateFilterConfig("state_id")({
      isEnabled: true,
      allowedOperators: new Set(["in"]),
      states: [
        { id: "backlog", name: "Backlog" },
        { id: "todo", name: todoLabel },
      ],
    });
    React.useLayoutEffect(() => {
      const button = environment.container.querySelector("button");
      commits.push({ labels: button.textContent, disabled: button.disabled });
    });
    return React.createElement(MultiSelectFilterValueInput, {
      config: config.supportedOperatorConfigsMap.get("in"),
      condition: { type: "condition", id: "status", property: "state_id", operator: "in", value: ["backlog", "todo"] },
      onChange: () => {},
    });
  }

  await environment.render(React.createElement(Probe, { todoLabel: "Todo" }));
  await environment.render(React.createElement(Probe, { todoLabel: "Ready" }));
  assert.deepEqual(commits, [
    { labels: "Backlog,Todo", disabled: false },
    { labels: "Backlog,Ready", disabled: false },
  ]);
});
