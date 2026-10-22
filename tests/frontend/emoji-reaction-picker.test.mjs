import assert from "node:assert/strict";
import test from "node:test";
import { act, createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks({ stubEmojiRoot: true });

test("reaction group keeps reaction buttons separate from its keyboard popover trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { EmojiReactionPicker } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx");
    const { EmojiReactionGroup } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.tsx");
    const reactionClicks = [];
    const toggleCalls = [];
    const addCalls = [];

    function Harness() {
      const [isOpen, setIsOpen] = React.useState(false);
      return React.createElement(EmojiReactionPicker, {
        isOpen,
        handleToggle(value) {
          toggleCalls.push(value);
          setIsOpen(value);
        },
        buttonClassName: "picker-trigger",
        onChange() {},
        label: React.createElement(EmojiReactionGroup, {
          reactions: [{ emoji: "👍", count: 1 }],
          onReactionClick(emoji) {
            reactionClicks.push(emoji);
          },
          onAddReaction() {
            addCalls.push(true);
          },
        }),
      });
    }

    await environment.render(React.createElement(Harness));

    const reaction = environment.container.querySelector("button:not(.picker-trigger)");
    const addTrigger = environment.container.querySelector("button.picker-trigger");
    assert.ok(reaction);
    assert.ok(addTrigger);
    assert.equal(environment.container.querySelectorAll("button").length, 2);
    assert.equal(environment.container.querySelector("button button"), null);

    await act(async () => reaction.click());
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.deepEqual(toggleCalls, []);

    async function checkKeyboardActivation(key) {
      await act(async () => {
        addTrigger.dispatchEvent(new KeyboardEvent("keydown", { key, bubbles: true, cancelable: true }));
        // jsdom does not synthesize a native button's keyboard activation click.
        addTrigger.click();
      });
      assert.equal(toggleCalls.at(-1), true);
      assert.equal(addTrigger.getAttribute("aria-expanded"), "true", environment.container.innerHTML);
      assert.deepEqual(addCalls, []);
      await act(async () => addTrigger.click());
      assert.equal(addTrigger.getAttribute("aria-expanded"), "false");
      assert.equal(toggleCalls.at(-1), false);
    }
    await checkKeyboardActivation("Enter");
    await checkKeyboardActivation(" ");
  } finally {
    await environment.unmount();
  }
});

test("a disabled reaction picker disables only its add trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { EmojiReactionPicker } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx");
    const { EmojiReactionGroup } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.tsx");
    const reactionClicks = [];
    const toggleCalls = [];

    await environment.render(
      React.createElement(EmojiReactionPicker, {
        isOpen: false,
        handleToggle: (value) => toggleCalls.push(value),
        disabled: true,
        buttonClassName: "picker-trigger",
        onChange() {},
        label: React.createElement(EmojiReactionGroup, {
          reactions: [{ emoji: "👍", count: 1 }],
          onReactionClick: (emoji) => reactionClicks.push(emoji),
        }),
      })
    );

    const addTrigger = environment.container.querySelector("button.picker-trigger");
    const reaction = environment.container.querySelector("button:not(.picker-trigger)");
    assert.ok(addTrigger?.disabled);
    assert.ok(reaction);
    await act(async () => {
      addTrigger.click();
      reaction.click();
    });

    assert.deepEqual(toggleCalls, []);
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.equal(environment.container.querySelector("button button"), null);
  } finally {
    await environment.unmount();
  }
});

test("a reaction group without an add button does not wrap reaction buttons in a trigger", async () => {
  const environment = await createReactDomTestEnvironment();

  try {
    const { EmojiReactionPicker } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction-picker.tsx");
    const { EmojiReactionGroup } = await import("../../packages/propel/src/emoji-reaction/emoji-reaction.tsx");
    const reactionClicks = [];
    const toggleCalls = [];

    await environment.render(
      React.createElement(EmojiReactionPicker, {
        isOpen: false,
        handleToggle: (value) => toggleCalls.push(value),
        buttonClassName: "picker-trigger",
        onChange() {},
        label: React.createElement(EmojiReactionGroup, {
          reactions: [{ emoji: "👍", count: 1 }],
          onReactionClick: (emoji) => reactionClicks.push(emoji),
          showAddButton: false,
        }),
      })
    );

    const reaction = environment.container.querySelector("button");
    assert.ok(reaction);
    assert.equal(environment.container.querySelector("button.picker-trigger"), null);
    assert.equal(environment.container.querySelector("button button"), null);
    await act(async () => reaction.click());
    assert.deepEqual(reactionClicks, ["👍"]);
    assert.deepEqual(toggleCalls, []);
  } finally {
    await environment.unmount();
  }
});
