import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { setPositioningGeometry } from "./helpers/positioning.mjs";
import { React, act } from "./helpers/react.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks({ stubEmojiRoot: true });

for (const [name, modulePath, exportName] of [
  ["emoji picker", "../../packages/propel/src/emoji-icon-picker/emoji-picker.tsx", "EmojiPicker"],
  ["reaction picker", "../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx", "EmojiReactionPicker"],
]) {
  for (const { scenario, options, side, align } of [
    { scenario: "honors side and align", options: { side: "left", align: "end" }, side: "left", align: "end" },
    {
      scenario: "retains explicit placement precedence",
      options: { placement: "top-start", side: "left", align: "end" },
      side: "top",
      align: "start",
    },
    { scenario: "defaults to bottom-start", options: {}, side: "bottom", align: "start" },
  ]) {
    void test(`${name} ${scenario}`, async (t) => {
      const environment = await createReactDomTestEnvironment();
      t.after(() => environment.unmount());
      const Component = (await import(modulePath))[exportName];
      function Harness() {
        const [isOpen, handleToggle] = React.useState(false);
        return React.createElement(Component, {
          ...options,
          isOpen,
          handleToggle,
          onChange: () => {},
          label: "Choose emoji",
        });
      }
      await environment.render(React.createElement(Harness));
      const trigger = environment.container.querySelector("button");
      setPositioningGeometry(trigger);
      await act(async () => trigger.click());
      const positioner = document.querySelector('[data-slot="popover-positioner"]');
      assert.ok(positioner);
      assert.equal(positioner.getAttribute("data-side"), side);
      assert.equal(positioner.getAttribute("data-align"), align);
    });
  }
}

for (const [name, modulePath, exportName] of [
  ["emoji", "../../packages/propel/src/emoji-icon-picker/emoji-picker.tsx", "EmojiPicker"],
  ["reaction", "../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx", "EmojiReactionPicker"],
]) {
  for (const closeOnSelect of [true, false]) {
    void test(`${name} selections serialize the emoji and honor closeOnSelect=${closeOnSelect}`, async (t) => {
      const env = await createReactDomTestEnvironment();
      t.after(() => env.unmount());
      const Component = (await import(modulePath))[exportName];
      const changes = [];
      function Harness() {
        const [isOpen, handleToggle] = React.useState(false);
        return React.createElement(Component, {
          isOpen,
          handleToggle,
          label: "Choose emoji",
          onChange: (value) => changes.push(value),
          ...(closeOnSelect ? {} : { closeOnSelect }),
        });
      }
      await env.render(React.createElement(Harness));
      const trigger = env.container.querySelector("button");
      setPositioningGeometry(trigger);
      await act(async () => trigger.click());
      const selection = document.querySelector('[data-testid="choose-emoji"]');
      assert.ok(selection);
      await act(async () => selection.click());
      assert.deepEqual(changes, name === "emoji" ? [{ type: "emoji", value: "128077" }] : ["128077"]);
      assert.equal(trigger.getAttribute("aria-expanded"), String(!closeOnSelect));
      if (!closeOnSelect) {
        await act(async () => {
          selection.focus();
          selection.dispatchEvent(new MouseEvent("mousedown", { bubbles: true }));
          selection.dispatchEvent(new KeyboardEvent("keydown", { key: "Tab", bubbles: true }));
          selection.dispatchEvent(new KeyboardEvent("keydown", { key: "x", bubbles: true }));
          selection.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true }));
        });
        assert.equal(trigger.getAttribute("aria-expanded"), "false");
      }
    });
  }
}

for (const closeOnSelect of [true, false]) {
  void test(`icon tab returns the selected name and color with closeOnSelect=${closeOnSelect}`, async (t) => {
    const env = await createReactDomTestEnvironment();
    t.after(() => env.unmount());
    const { EmojiPicker } = await import("../../packages/propel/src/emoji-icon-picker/emoji-picker.tsx");
    const changes = [];
    function Harness() {
      const [isOpen, handleToggle] = React.useState(false);
      return React.createElement(EmojiPicker, {
        isOpen,
        handleToggle,
        label: "Choose icon",
        onChange: (value) => changes.push(value),
        defaultIconColor: "#123456",
        ...(closeOnSelect ? {} : { closeOnSelect }),
      });
    }
    await env.render(React.createElement(Harness));
    const trigger = env.container.querySelector("button");
    setPositioningGeometry(trigger);
    await act(async () => trigger.click());
    const iconTab = [...document.querySelectorAll('[role="tab"]')].find((tab) => tab.textContent === "Icon");
    await act(async () => iconTab.click());
    const icon = [...document.querySelectorAll('[role="tabpanel"] button[aria-label]')].find((button) =>
      /^archive$/i.test(button.getAttribute("aria-label"))
    );
    assert.ok(icon);
    await act(async () => icon.click());
    assert.deepEqual(changes, [{ type: "icon", value: { name: icon.getAttribute("aria-label"), color: "#123456" } }]);
    assert.equal(trigger.getAttribute("aria-expanded"), String(!closeOnSelect));
  });
}
