import assert from "node:assert/strict";
import test from "node:test";
import { React } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";

import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const { Card, ECardDirection, ECardSpacing, ECardVariant } = await import("../../packages/ui/dist/index.js");

await test("the shared Card forwards its ref and native props while custom classes override defaults", async (t) => {
  const env = await createReactDomTestEnvironment();
  t.after(() => env.unmount());
  const ref = React.createRef();
  await env.render(
    React.createElement(
      Card,
      {
        ref,
        role: "region",
        "aria-label": "Release summary",
        direction: ECardDirection.ROW,
        spacing: ECardSpacing.LG,
        variant: ECardVariant.WITHOUT_SHADOW,
        className: "p-2",
      },
      "Ready to ship"
    )
  );
  const card = env.container.querySelector('[role="region"]');
  assert.equal(ref.current, card);
  assert.equal(card.getAttribute("aria-label"), "Release summary");
  assert.equal(card.textContent, "Ready to ship");
  assert.ok(card.classList.contains("flex-row"));
  assert.ok(card.classList.contains("p-2"));
  assert.equal(card.classList.contains("p-6"), false);
  assert.equal(card.classList.contains("hover:shadow-raised-200"), false);
});
