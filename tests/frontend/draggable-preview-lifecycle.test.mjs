import assert from "node:assert/strict";
import test from "node:test";
import { React, act, registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { registerWebTestModules } from "./helpers/web-test-modules.mjs";

globalThis.dragPreviewReact = React;
registerWebTestModules({
  "@atlaskit/pragmatic-drag-and-drop/adapter/element-adapter": `
    export const draggable=options=>globalThis.dragPreviewFixture.register('sources',options);
    export const dropTargetForElements=options=>globalThis.dragPreviewFixture.register('targets',options);
  `,
  "@atlaskit/pragmatic-drag-and-drop/element/set-custom-native-drag-preview": `export const setCustomNativeDragPreview=options=>globalThis.dragPreviewFixture.previews.push(options);`,
  "@atlaskit/pragmatic-drag-and-drop/element/pointer-outside-of-preview": `export const pointerOutsideOfPreview=()=>()=>({x:0,y:0});`,
  "@plane/i18n": "export const useTranslation=()=>({t:key=>key});",
  "@plane/ui": `export const DropIndicator=()=>null;export const ToggleSwitch=()=>null;`,
});
registerPermissionUiStubs("/manage/widget-item.tsx", {
  "../../home-dashboard-widgets": "export const HOME_WIDGETS_LIST={assigned:{title:'Assigned'}};",
  "./widget-item-drag-handle": `export const WidgetItemDragHandle=({isDragging})=>globalThis.dragPreviewReact.createElement('output',{'data-dragging':''},String(isDragging));`,
});
registerPermissionUiStubs("/favorite-items/root.tsx", {
  "@/hooks/use-favorite-item-details": `const R=globalThis.dragPreviewReact;export const useFavoriteItemDetails=()=>({itemTitle:globalThis.dragPreviewFixture.title,itemLink:'/acme/favorite',itemIcon:R.createElement('span',null)});`,
  "./common": `
    const R=globalThis.dragPreviewReact;
    export const FavoriteItemWrapper=({elementRef,children})=>R.createElement('div',{ref:elementRef},children);
    export const FavoriteItemDragHandle=({isDragging})=>R.createElement('output',{'data-dragging':''},String(isDragging));
    export const FavoriteItemTitle=({href,title,icon})=>R.createElement('a',{href},icon,title);
    export const FavoriteItemQuickAction=()=>null;
  `,
});

async function setup(t, kind) {
  const env = await createReactDomTestEnvironment();
  const state = (globalThis.dragPreviewFixture = {
    sources: [],
    targets: [],
    previews: [],
    roots: new Set(),
    containers: [],
    title: "Current favorite",
    register(list, options) {
      const entry = { options, active: true, cleanups: 0 };
      this[list].push(entry);
      return () => {
        entry.active = false;
        entry.cleanups++;
      };
    },
  });
  t.after(async () => {
    await act(async () => {
      for (const cleanup of state.roots) cleanup();
    });
    for (const container of state.containers) container.remove();
    await env.unmount();
  });
  const Component =
    kind === "widget"
      ? (await import("../../apps/web/core/components/home/widgets/manage/widget-item.tsx")).WidgetItem
      : (await import("../../apps/web/core/components/workspace/sidebar/favorites/favorite-items/root.tsx"))
          .FavoriteRoot;
  const props =
    kind === "widget"
      ? {
          widget: { key: "assigned", is_enabled: true, sort_order: 1 },
          disabled: false,
          isLastChild: true,
          handleDrop: () => {},
          handleToggle: () => {},
        }
      : {
          favorite: { id: "favorite", sort_order: 1 },
          parentId: "parent",
          workspaceSlug: "acme",
          isLastChild: true,
          handleDrop: () => {},
          handleRemoveFromFavorites: () => {},
        };
  const render = (extra = {}) => env.render(React.createElement(Component, { ...props, ...extra }));
  await render();
  const source = () => state.sources.find((entry) => entry.active);
  const target = () => state.targets.find((entry) => entry.active);
  const preview = async () => {
    await act(async () => source().options.onGenerateDragPreview({ nativeSetDragImage: () => {} }));
    const container = document.createElement("div");
    document.body.append(container);
    state.containers.push(container);
    let cleanup;
    await act(async () => {
      cleanup = state.previews.at(-1).render({ container });
    });
    state.roots.add(cleanup);
    return {
      container,
      cleanup: async () => {
        await act(async () => cleanup());
        state.roots.delete(cleanup);
      },
    };
  };
  return { env, state, render, source, target, preview };
}

async function registerPreviewContract(kind, expected) {
  await test(`${kind} drag sources retain their metadata and reclaim their React preview and registrations`, async (t) => {
    const view = await setup(t, kind);
    assert.deepEqual(view.source().options.getInitialData(), expected);
    const preview = await view.preview();
    assert.equal(preview.container.textContent, kind === "widget" ? "assigned" : "Current favorite");
    await preview.cleanup();
    assert.equal(preview.container.textContent, "");
    const source = view.source(),
      target = view.target();
    await view.env.render(null);
    assert.equal(source.cleanups, 1);
    assert.equal(target.cleanups, 1);
    assert.equal(view.state.sources.filter((entry) => entry.active).length, 0);
  });
}

await registerPreviewContract("widget", { id: "assigned", parentId: null, isGroup: false, isChild: false });
await registerPreviewContract("favorite", { id: "favorite", parentId: "parent", isGroup: false, isChild: true });

await test("favorite dragging retains its source and target until unmount", async (t) => {
  const view = await setup(t, "favorite");
  const source = view.source(),
    target = view.target();
  await act(async () => source.options.onDragStart());
  assert.equal(document.querySelector("[data-dragging]").textContent, "true");
  assert.equal(source.cleanups, 0, "starting a drag must not dispose the active source");
  assert.equal(target.cleanups, 0, "starting a drag must not dispose its target");
  await act(async () => source.options.onDrop());
  assert.equal(document.querySelector("[data-dragging]").textContent, "false");
  assert.equal(source.cleanups, 0);
  assert.equal(target.cleanups, 0);
  await view.env.render(null);
  assert.equal(source.cleanups, 1);
  assert.equal(target.cleanups, 1);
});

await test("favorite reparenting refreshes source metadata, target eligibility and the current drop callback", async (t) => {
  const view = await setup(t, "favorite");
  await view.render();
  const calls = [];
  await view.render({ parentId: undefined, handleDrop: (...args) => calls.push(args) });
  assert.deepEqual(view.source().options.getInitialData(), {
    id: "favorite",
    parentId: null,
    isGroup: false,
    isChild: false,
  });
  const target = view.target();
  assert.equal(target.options.canDrop({ source: { data: { id: "group", isGroup: true } } }), true);
  const payload = {
    self: { id: "target" },
    source: { data: { id: "other" } },
    location: { current: { dropTargets: [] } },
  };
  await act(async () => target.options.onDrop(payload));
  assert.deepEqual(calls, [[payload.self, payload.source, payload.location]]);
});

await test("favorite previews use the current title without reconnecting the native drag source", async (t) => {
  const view = await setup(t, "favorite");
  await view.render();
  const source = view.source();
  view.state.title = "Renamed favorite";
  await view.render();
  const preview = await view.preview();
  assert.equal(preview.container.textContent, "Renamed favorite");
  assert.equal(source.cleanups, 0);
  await preview.cleanup();
});

await test("a preview owner without a mounted element does not register a drag source", async (t) => {
  const view = await setup(t, "widget");
  await view.env.render(null);
  const { useDraggablePreview } = await import("../../apps/web/core/hooks/use-draggable-preview.tsx");
  function MissingElementOwner() {
    const elementRef = React.useRef(null);
    useDraggablePreview({
      elementRef,
      getInitialData: () => ({ id: "missing" }),
      preview: "Missing",
      onDraggingChange: () => {},
    });
    return null;
  }
  const registrations = view.state.sources.length;
  await view.env.render(React.createElement(MissingElementOwner));
  assert.equal(view.state.sources.length, registrations);
  assert.equal(
    view.state.sources.some((entry) => entry.active),
    false
  );
});
