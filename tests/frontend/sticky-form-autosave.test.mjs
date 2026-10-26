import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { pathToFileURL } from "node:url";
import { React, webRequire } from "./helpers/react.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

registerWebModuleHooks();
const reactUrl = pathToFileURL(webRequire.resolve("react")).href;
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@tanstack/react-router") {
      return createModuleStub(`export const useLocation = ({ select }) => select({ pathname: "/acme/stickies/" });`);
    }
    if (specifier === "@/hooks/use-workspaces") {
      return createModuleStub(`export const useWorkspaceBySlug = () => ({ data: { id: "workspace-id" } });`);
    }
    if (specifier === "@/components/stickies/provider") {
      return createModuleStub(`export const useStickies = () => globalThis.stickyFormStore;`);
    }
    if (specifier === "./use-operations" && context.parentURL?.includes("/stickies/sticky/root.tsx")) {
      return createModuleStub(
        `export const useStickyOperations = () => ({
          stickyOperations: globalThis.stickyFormOperations,
          sessionKey: "alice/0",
          isCurrent: () => globalThis.stickyFormCurrent !== false,
        });`
      );
    }
    if (specifier === "@/components/editor/sticky-editor") {
      return createModuleStub(`
        import React from "${reactUrl}";
        export const StickyEditor = React.forwardRef(function StickyEditor(props, ref) {
          return React.createElement("div", null,
            React.createElement("textarea", {
              id: "sticky-description",
              defaultValue: props.initialValue,
              onChange: event => props.onChange({}, event.target.value),
            }),
            React.createElement("output", { id: "sticky-form-value" }, props.initialValue),
            React.createElement("button", { id: "sticky-color", onClick: () => props.handleColorChange({ background_color: "pink" }) }, "Color"),
            React.createElement("button", { id: "sticky-toolbar-delete", onClick: props.handleDelete }, "Delete toolbar")
          );
        });
      `);
    }
    return resolvePackageStub(
      specifier,
      {
        "@plane/utils/common": `export const cn = (...values) => values.filter(Boolean).join(" ");`,
        "@plane/utils/string": `export const isCommentEmpty = value => !value || value === "<p></p>";`,
        "lucide-react": `export const Minimize2 = () => null;`,
        "@plane/ui": `
          import React from "${reactUrl}";
          export const AlertModalCore = ({ isOpen, handleClose, handleSubmit, isSubmitting, content }) => isOpen
            ? React.createElement("section", { id: "sticky-delete-dialog" }, content,
              React.createElement("button", { id: "cancel-delete", onClick: handleClose }, "Cancel"),
              React.createElement("button", { id: "confirm-delete", disabled: isSubmitting, onClick: handleSubmit }, "Delete"))
            : null;
        `,
      },
      context,
      nextResolve
    );
  },
});

async function editDescription(environment, value) {
  await React.act(async () => {
    const editor = environment.container.querySelector("textarea");
    Object.getOwnPropertyDescriptor(window.HTMLTextAreaElement.prototype, "value").set.call(editor, value);
    editor.dispatchEvent(new Event("input", { bubbles: true }));
  });
}

async function waitForDebounce() {
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 550)));
}

await test("sticky form preserves newer drafts across accepted and rejected autosaves", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { StickyNote } = await import("../../apps/web/core/components/stickies/sticky/root.tsx");
  const firstSave = Promise.withResolvers();
  const secondSave = Promise.withResolvers();
  const updates = [];
  let failSecondSave = true;
  globalThis.stickyFormStore = {
    stickies: new Map([["sticky-1", { id: "sticky-1", description_html: "<p>Initial</p>", background_color: "gray" }]]),
  };
  globalThis.stickyFormOperations = {
    update: (id, payload) => {
      updates.push({ id, payload });
      if (updates.length === 1) return firstSave.promise;
      if (updates.length === 2) return secondSave.promise;
      if (updates.length === 3 && failSecondSave) return Promise.reject(new Error("offline"));
      return Promise.resolve();
    },
    remove: async () => {},
  };

  await environment.render(React.createElement(StickyNote, { workspaceSlug: "acme", stickyId: "sticky-1" }));
  await editDescription(environment, "<p>Older draft</p>");
  await waitForDebounce();
  assert.deepEqual(updates, [{ id: "sticky-1", payload: { description_html: "<p>Older draft</p>" } }]);

  await editDescription(environment, "<p>Newer draft</p>");
  await waitForDebounce();
  assert.equal(updates.length, 1, "a newer draft waits for the active Form submission");
  await React.act(async () => {
    firstSave.reject(new Error("stale failure"));
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  assert.equal(updates.length, 2);
  assert.deepEqual(updates[1], { id: "sticky-1", payload: { description_html: "<p>Newer draft</p>" } });
  await React.act(async () => {
    secondSave.resolve();
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  assert.equal(environment.container.querySelector("#sticky-form-value").textContent, "<p>Newer draft</p>");

  assert.equal(environment.container.querySelector("#sticky-form-value").textContent, "<p>Newer draft</p>");

  await editDescription(environment, "<p>Retry draft</p>");
  await waitForDebounce();
  assert.equal(updates.length, 3);
  assert.deepEqual(updates[2], { id: "sticky-1", payload: { description_html: "<p>Retry draft</p>" } });
  assert.match(environment.container.querySelector('[role="alert"]').textContent, /Could not save this sticky/);
  await React.act(async () => environment.container.querySelector("button.underline").click());
  await React.act(async () => Promise.resolve());
  assert.equal(updates.length, 4);
  assert.equal(environment.container.querySelector('[role="alert"]'), null);
  assert.equal(environment.container.querySelector("#sticky-form-value").textContent, "<p>Retry draft</p>");
});

function pendingStickyFixture() {
  const writes = [];
  const first = Promise.withResolvers();
  globalThis.stickyFormCurrent = true;
  globalThis.stickyFormStore = {
    stickies: new Map([["sticky-1", { id: "sticky-1", description_html: "<p>Initial</p>", background_color: "gray" }]]),
  };
  globalThis.stickyFormOperations = {
    update: (_id, patch) => {
      writes.push(patch.description_html);
      return writes.length === 1 ? first.promise : Promise.resolve();
    },
    remove: async () => {},
  };
  return { writes, first };
}

await test("same-session unmount flushes the latest sticky draft after an active save", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { StickyNote } = await import("../../apps/web/core/components/stickies/sticky/root.tsx");
  const { writes, first } = pendingStickyFixture();
  await environment.render(React.createElement(StickyNote, { workspaceSlug: "acme", stickyId: "sticky-1" }));
  await editDescription(environment, "<p>First</p>");
  await waitForDebounce();
  await editDescription(environment, "<p>Last</p>");
  await environment.unmount();
  await React.act(async () => {
    first.resolve();
    await new Promise(setImmediate);
  });
  assert.deepEqual(writes, ["<p>First</p>", "<p>Last</p>"]);
});

await test("session replacement discards the queued sticky draft after unmount", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { StickyNote } = await import("../../apps/web/core/components/stickies/sticky/root.tsx");
  const { writes, first } = pendingStickyFixture();
  await environment.render(React.createElement(StickyNote, { workspaceSlug: "acme", stickyId: "sticky-1" }));
  await editDescription(environment, "<p>First</p>");
  await waitForDebounce();
  await editDescription(environment, "<p>Discard old account draft</p>");
  globalThis.stickyFormCurrent = false;
  await environment.unmount();
  await React.act(async () => {
    first.resolve();
    await new Promise(setImmediate);
  });
  assert.deepEqual(writes, ["<p>First</p>"]);
});

async function openStickyDelete(environment) {
  await React.act(async () => environment.container.querySelector("#sticky-toolbar-delete").click());
  await React.act(async () => environment.container.querySelector("#confirm-delete").click());
}

await test("sticky deletion cancels the draft queued behind an active autosave", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { StickyNote } = await import("../../apps/web/core/components/stickies/sticky/root.tsx");
  const { writes, first } = pendingStickyFixture();
  const deletion = Promise.withResolvers();
  const removes = [];
  let closes = 0;
  globalThis.stickyFormOperations.remove = async (id) => {
    await first.promise;
    removes.push(id);
    await deletion.promise;
  };
  await environment.render(
    React.createElement(StickyNote, { workspaceSlug: "acme", stickyId: "sticky-1", onClose: () => closes++ })
  );
  await editDescription(environment, "<p>Active</p>");
  await waitForDebounce();
  await editDescription(environment, "<p>Discard when deleted</p>");
  await waitForDebounce();
  await openStickyDelete(environment);
  await React.act(async () => environment.container.querySelector("#cancel-delete").click());
  assert.ok(
    environment.container.querySelector("#sticky-delete-dialog"),
    "pending deletion cannot dismiss its confirmation"
  );
  assert.equal(closes, 0);
  await React.act(async () => {
    first.resolve();
    await new Promise(setImmediate);
  });
  assert.deepEqual(removes, ["sticky-1"]);
  assert.deepEqual(writes, ["<p>Active</p>"]);
  await React.act(async () => {
    deletion.resolve();
    await new Promise(setImmediate);
  });
  assert.equal(closes, 1);
  assert.equal(environment.container.querySelector("#sticky-delete-dialog"), null);
  assert.deepEqual(writes, ["<p>Active</p>"]);
});

await test("failed sticky deletion retains the draft, resumes autosave, and allows retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { StickyNote } = await import("../../apps/web/core/components/stickies/sticky/root.tsx");
  const { writes, first } = pendingStickyFixture();
  let removals = 0;
  let closes = 0;
  globalThis.stickyFormOperations.remove = async () => {
    if (++removals === 1) throw new Error("offline");
  };
  await environment.render(
    React.createElement(StickyNote, { workspaceSlug: "acme", stickyId: "sticky-1", onClose: () => closes++ })
  );
  await editDescription(environment, "<p>Active</p>");
  await waitForDebounce();
  await editDescription(environment, "<p>Keep after failed delete</p>");
  await waitForDebounce();
  await openStickyDelete(environment);
  assert.equal(closes, 0);
  assert.ok(environment.container.querySelector("#sticky-delete-dialog [role='alert']"));
  assert.equal(
    environment.container.querySelector("#sticky-form-value").textContent,
    "<p>Keep after failed delete</p>"
  );
  await React.act(async () => {
    first.resolve();
    await new Promise(setImmediate);
  });
  assert.deepEqual(writes, ["<p>Active</p>", "<p>Keep after failed delete</p>"]);
  await React.act(async () => environment.container.querySelector("#confirm-delete").click());
  assert.equal(removals, 2);
  assert.equal(closes, 1);
  assert.equal(environment.container.querySelector("#sticky-delete-dialog"), null);
});
