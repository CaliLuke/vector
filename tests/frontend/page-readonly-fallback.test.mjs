import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment, React } from "./helpers/react-dom.mjs";

registerWebModuleHooks();

const editorStub = `data:text/javascript,${encodeURIComponent(`
  export const convertBinaryDataToBase64String = () => "encoded-binary";
  export const getBinaryDataFromDocumentEditorHTMLString = () => new Uint8Array([0]);
`)}`;

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/editor") return { shortCircuit: true, url: editorStub };
    return nextResolve(specifier, context);
  },
});

let usePageFallback;

function FallbackHarness(props) {
  usePageFallback(props);
  return null;
}

function makePage({ locked = false, archived = false, canEdit = true } = {}) {
  return {
    name: "Lifecycle page",
    description_html: "<p>Saved text</p>",
    is_locked: locked,
    archived_at: archived ? "2026-10-21T00:00:00Z" : null,
    get isContentEditable() {
      return !this.is_locked && !this.archived_at && canEdit;
    },
  };
}

function makeProps(page, { fetchPageDescription, updatePageDescription, documentPayload } = {}) {
  let hydratedDocument;
  const editorRef = {
    current: {
      setProviderDocument(value) {
        hydratedDocument = value;
      },
      getDocument() {
        return documentPayload;
      },
    },
  };

  return {
    editorRef,
    fetchPageDescription: fetchPageDescription ?? (async () => new Uint8Array([1, 2, 3]).buffer),
    collaborationState: { stage: { kind: "disconnected" } },
    updatePageDescription: updatePageDescription ?? (async () => {}),
    page,
    getHydratedDocument: () => hydratedDocument,
  };
}

async function mountFallback(t, props) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  usePageFallback ??= (await import("../../apps/web/core/hooks/use-page-fallback.ts")).usePageFallback;
  await environment.render(React.createElement(FallbackHarness, props));
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  return environment;
}

test("locked page fallback hydrates the document without issuing a description write", async (t) => {
  const page = makePage({ locked: true });
  let writes = 0;
  const props = makeProps(page, {
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Saved text</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      writes++;
    },
  });

  await mountFallback(t, props);

  assert.deepEqual([...props.getHydratedDocument()], [1, 2, 3]);
  assert.equal(writes, 0);
});

test("archived page fallback hydrates the document without issuing a description write", async (t) => {
  const page = makePage({ archived: true });
  let writes = 0;
  const props = makeProps(page, {
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Saved text</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      writes++;
    },
  });

  await mountFallback(t, props);

  assert.deepEqual([...props.getHydratedDocument()], [1, 2, 3]);
  assert.equal(writes, 0);
});

test("read-only page fallback hydrates the document without issuing a description write", async (t) => {
  const page = makePage({ canEdit: false });
  let writes = 0;
  const props = makeProps(page, {
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Saved text</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      writes++;
    },
  });

  await mountFallback(t, props);

  assert.deepEqual([...props.getHydratedDocument()], [1, 2, 3]);
  assert.equal(writes, 0);
});

test("page locked while fallback fetch is pending is hydrated but not persisted", async (t) => {
  const page = makePage();
  let resolveDescription;
  let fetches = 0;
  let writes = 0;
  const props = makeProps(page, {
    fetchPageDescription: () => {
      fetches++;
      return new Promise((resolve) => {
        resolveDescription = resolve;
      });
    },
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Unsynced edit</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      writes++;
    },
  });

  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  usePageFallback ??= (await import("../../apps/web/core/hooks/use-page-fallback.ts")).usePageFallback;
  await environment.render(React.createElement(FallbackHarness, props));
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  assert.equal(typeof resolveDescription, "function", "fallback fetch should be in flight");

  page.description_html = "<p>Updated during hydration</p>";
  await environment.render(React.createElement(FallbackHarness, props));
  assert.equal(fetches, 1, "a rerender should not start a duplicate fallback request");

  page.is_locked = true;
  await React.act(async () => {
    resolveDescription(new Uint8Array([4, 5]).buffer);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.deepEqual([...props.getHydratedDocument()], [4, 5]);
  assert.equal(writes, 0);
});

test("editable fallback continues to persist the recovered document", async (t) => {
  const page = makePage();
  const savedDocuments = [];
  const props = makeProps(page, {
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Unsynced edit</p>", json: { type: "doc" } },
    updatePageDescription: async (document) => {
      savedDocuments.push(document);
    },
  });

  await mountFallback(t, props);

  assert.equal(savedDocuments.length, 1);
  assert.equal(savedDocuments[0].description_html, "<p>Unsynced edit</p>");
  assert.equal(savedDocuments[0].description_binary, "encoded-binary");
});
