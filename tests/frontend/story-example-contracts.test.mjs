import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
registerWebModuleHooks();
for (const [folder, variants, sizes] of [
  ["button", 7, 4],
  ["badge", 5, 3],
  ["icon-button", 6, 4],
]) {
  void test(`${folder} galleries preserve all variants, sizes and disabled states`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const stories = await import(`../../packages/propel/src/${folder}/${folder}.stories.tsx`);
    await env.render(React.createElement(stories.AllVariants.render));
    assert.equal(env.container.querySelector("h3").textContent, "Primary Variants");
    assert.equal(env.container.querySelector(".flex").children.length, variants);
    await env.render(React.createElement(stories.AllSizes.render));
    assert.equal(env.container.querySelector(".flex").children.length, sizes);
    if (folder !== "badge") {
      await env.render(React.createElement(stories.AllStates.render));
      const buttons = [...env.container.querySelectorAll("button")];
      assert.equal(buttons.length, 3);
      assert.equal(buttons[0].disabled, false);
      assert.equal(buttons[2].disabled, true);
    }
    const Component = stories.default.component;
    await env.render(React.createElement(Component, { ...stories.default.args, ...stories.Default.args }));
    assert.ok(env.container.firstElementChild);
    if (folder !== "icon-button") {
      await env.render(React.createElement(Component, stories.WithPrependIcon.args));
      assert.equal(env.container.textContent, "With Prepend Icon");
      assert.equal(env.container.querySelector("svg").getAttribute("width"), "16");
      assert.equal(env.container.querySelector("svg").classList.contains("lucide-plus"), true);
      await env.render(React.createElement(Component, stories.WithAppendIcon.args));
      assert.equal(env.container.textContent, "With Append Icon");
      assert.equal(env.container.querySelector("svg").classList.contains("lucide-chevron-right"), true);
    }
  });
}
for (const [file, size] of [
  ["circular-spinner", "32px"],
  ["circular-bar-spinner", "24px"],
]) {
  void test(`${file} color examples preserve labels, accessible loading output and size`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const stories = await import(`../../packages/propel/src/spinners/${file}.stories.tsx`);
    await env.render(React.createElement(stories.ColorVariations.render));
    assert.deepEqual(
      [...env.container.querySelectorAll("p")].map((node) => node.textContent),
      ["Blue", "Green", "Red", "Purple", "Orange"]
    );
    assert.equal(env.container.querySelectorAll("output").length, 5);
    assert.deepEqual(
      [...env.container.querySelectorAll("svg")].map((node) => node.getAttribute("height")),
      [size, size, size, size, size]
    );
    assert.equal(env.container.querySelector("svg").classList.contains("text-blue-500"), true);
  });
}
void test("empty-state controls preserve distinct asset choices and detailed description support", async () => {
  const compact = (await import("../../packages/propel/src/empty-state/compact-empty-state.stories.tsx")).default;
  const detailed = (await import("../../packages/propel/src/empty-state/detailed-empty-state.stories.tsx")).default;
  assert.equal(compact.argTypes.assetKey.options.includes("estimate"), true);
  assert.equal(detailed.argTypes.assetKey.options.includes("dashboard"), true);
  assert.equal(detailed.argTypes.description.control, "text");
  assert.equal(compact.argTypes.description, undefined);
  assert.equal(compact.argTypes.className.control, "text");
  assert.equal(detailed.argTypes.actions.control, false);
  assert.equal(compact.argTypes.asset.control, false);
});
