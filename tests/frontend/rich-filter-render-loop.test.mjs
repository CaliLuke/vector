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
