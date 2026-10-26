import assert from "node:assert/strict";
import { registerHooks } from "node:module";
import test from "node:test";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { createPageIdentityFixture } from "./helpers/page-identity-fixture.mjs";

registerWebModuleHooks();

const editorModuleUrl = new URL("../../packages/editor/dist/index.js", import.meta.url).href;
const editorModuleAdapter = `data:text/javascript,${encodeURIComponent(`
  export {
    applyUpdates,
    convertBase64StringToBinaryData,
    convertBinaryDataToBase64String,
    getAllDocumentFormatsFromDocumentEditorBinaryData,
    getBinaryDataFromDocumentEditorHTMLString,
  } from ${JSON.stringify(editorModuleUrl)};
`)}`;

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier === "@plane/editor") return { shortCircuit: true, url: editorModuleAdapter };
    return nextResolve(specifier, context);
  },
});

const realEditorModule = await import(editorModuleUrl);
const permissionQueries = await import("../../apps/web/core/queries/permissions.ts");
const identityQueries = await import("../../apps/web/core/queries/identity.ts");

let usePageFallback;

function FallbackHarness(props) {
  usePageFallback(props);
  return null;
}

function ConditionalFallbackHarness({ mounted, ...props }) {
  return mounted ? React.createElement(FallbackHarness, props) : null;
}

function makePage({ locked = false, archived = false, canEdit = true } = {}) {
  const page = {
    id: "page-1",
    name: "Lifecycle page",
    description_html: "<p>Saved text</p>",
    is_locked: locked,
    archived_at: archived ? "2026-10-21T00:00:00Z" : null,
    project_ids: ["project-1"],
    owned_by: canEdit ? "owner" : "another-user",
    access: 1,
  };
  return page;
}

function makeProps(page, { fetchPageDescription, updatePageDescription, documentPayload, realConversion } = {}) {
  let hydratedDocument;
  let hydratedFormats;
  const editorRef = {
    current: {
      setProviderDocument(value) {
        hydratedDocument = realConversion ? realConversion.applyUpdates(value) : value;
      },
      getDocument() {
        if (realConversion) {
          hydratedFormats = realConversion.getAllDocumentFormatsFromDocumentEditorBinaryData(hydratedDocument, false);
          return {
            binary: realConversion.convertBase64StringToBinaryData(hydratedFormats.contentBinaryEncoded),
            html: hydratedFormats.contentHTML,
            json: hydratedFormats.contentJSON,
          };
        }
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
    workspaceSlug: "acme",
    getHydratedDocument: () => (realConversion ? hydratedFormats : hydratedDocument),
  };
}

async function seedPermissionQueries(client, roles = { "project-1": 15 }) {
  const { permissionKeys } = permissionQueries;
  const membership = { is_active: true, role: 15, updated_at: "workspace-revision" };
  client.setQueryData(permissionKeys.workspaceMembership("owner", "acme"), membership);
  client.setQueryData(
    permissionKeys.workspaceProjectRoles("owner", "acme", membership.role, membership.updated_at),
    roles
  );
}

async function mountFallback(t, props, { conditional = false } = {}) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const identityFixture = await createPageIdentityFixture(t);
  await seedPermissionQueries(identityFixture.client);
  usePageFallback ??= (await import("../../apps/web/core/hooks/use-page-fallback.ts")).usePageFallback;
  const render = async (nextProps = props, mounted = true) =>
    identityFixture.render(
      environment,
      conditional
        ? React.createElement(ConditionalFallbackHarness, { mounted, ...nextProps })
        : React.createElement(FallbackHarness, nextProps)
    );
  await render();
  await React.act(async () => new Promise((resolve) => setTimeout(resolve, 0)));
  return { environment, identityFixture, render };
}

async function mountPendingFallback(t, { page = makePage(), conditional = false } = {}) {
  let resolveDescription;
  const effects = { fetches: 0, writes: 0, hydrations: 0 };
  const props = makeProps(page, {
    fetchPageDescription: () => {
      effects.fetches++;
      return new Promise((resolve) => {
        resolveDescription = resolve;
      });
    },
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Unsynced edit</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      effects.writes++;
    },
  });
  const originalSetProviderDocument = props.editorRef.current.setProviderDocument.bind(props.editorRef.current);
  props.editorRef.current.setProviderDocument = (value) => {
    effects.hydrations++;
    originalSetProviderDocument(value);
  };
  const fixture = await mountFallback(t, props, { conditional });

  return {
    ...fixture,
    page,
    props,
    effects,
    get resolveDescription() {
      return resolveDescription;
    },
  };
}

async function assertFallbackHydrationDoesNotWrite(t, page) {
  let writes = 0;
  const props = makeProps(page, {
    documentPayload: { binary: new Uint8Array([1]), html: "<p>Saved text</p>", json: { type: "doc" } },
    updatePageDescription: async () => {
      writes++;
    },
  });

  await mountFallback(t, props);

  assert.deepEqual([...props.getHydratedDocument()], [1, 2, 3]);
  assert.equal(writes, 0, "read-only hydration must not persist the recovered document");
}

void test("locked page fallback hydrates the document without issuing a description write", async (t) => {
  await assertFallbackHydrationDoesNotWrite(t, makePage({ locked: true }));
});

void test("archived page fallback hydrates the document without issuing a description write", async (t) => {
  await assertFallbackHydrationDoesNotWrite(t, makePage({ archived: true }));
});

void test("read-only page fallback hydrates the document without issuing a description write", async (t) => {
  await assertFallbackHydrationDoesNotWrite(t, makePage({ canEdit: false }));
});

void test("read-only page fallback round-trips real document content without writing", async (t) => {
  const sourceBinary = realEditorModule.getBinaryDataFromDocumentEditorHTMLString(
    "<p>Saved <strong>formatted</strong> text</p>"
  );
  const page = makePage({ canEdit: false });
  let writes = 0;
  const props = makeProps(page, {
    fetchPageDescription: async () =>
      sourceBinary.buffer.slice(sourceBinary.byteOffset, sourceBinary.byteOffset + sourceBinary.byteLength),
    realConversion: realEditorModule,
    updatePageDescription: async () => {
      writes++;
    },
  });

  await mountFallback(t, props);

  const hydrated = props.getHydratedDocument();
  assert.equal(hydrated.contentJSON.type, "doc");
  assert.deepEqual(
    hydrated.contentJSON.content.map(({ type, content }) => ({
      type,
      content: content?.map(({ type: nodeType, text, marks }) => ({
        type: nodeType,
        text,
        marks: marks?.map(({ type: markType }) => markType) ?? [],
      })),
    })),
    [
      {
        type: "paragraph",
        content: [
          { type: "text", text: "Saved ", marks: [] },
          { type: "text", text: "formatted", marks: ["bold"] },
          { type: "text", text: " text", marks: [] },
        ],
      },
    ]
  );
  assert.match(hydrated.contentHTML, /<strong>formatted<\/strong>/);
  assert.equal(writes, 0, "read-only hydration must not persist the recovered document");
});

void test("page locked while fallback fetch is pending is hydrated but not persisted", async (t) => {
  const fixture = await mountPendingFallback(t);
  assert.equal(typeof fixture.resolveDescription, "function", "fallback fetch should be in flight");

  fixture.page.description_html = "<p>Updated during hydration</p>";
  await fixture.render();
  assert.equal(fixture.effects.fetches, 1, "a rerender should not start a duplicate fallback request");

  fixture.page.is_locked = true;
  await React.act(async () => {
    fixture.resolveDescription(new Uint8Array([4, 5]).buffer);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.deepEqual([...fixture.props.getHydratedDocument()], [4, 5]);
  assert.equal(fixture.effects.writes, 0);
});

void test("fallback does not hydrate or write after same-account session generation changes", async (t) => {
  const fixture = await mountPendingFallback(t);
  assert.equal(typeof fixture.resolveDescription, "function");

  identityQueries.advanceIdentitySession(fixture.identityFixture.client);
  await React.act(async () => {
    fixture.resolveDescription(new Uint8Array([4, 5]).buffer);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.equal(fixture.effects.hydrations, 0, "a superseded session cannot hydrate the current editor");
  assert.equal(fixture.effects.writes, 0);
});

void test("fallback does not hydrate or write after the page unmounts during its fetch", async (t) => {
  const fixture = await mountPendingFallback(t, { conditional: true });
  assert.equal(typeof fixture.resolveDescription, "function");

  await fixture.render(fixture.props, false);
  await React.act(async () => {
    fixture.resolveDescription(new Uint8Array([4, 5]).buffer);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.equal(fixture.effects.hydrations, 0, "an unmounted page cannot hydrate a stale editor");
  assert.equal(fixture.effects.writes, 0, "an unmounted page cannot persist fallback content");
});

void test("fallback uses latest project roles after its description fetch", async (t) => {
  const fixture = await mountPendingFallback(t);
  const { permissionKeys } = permissionQueries;
  await React.act(async () => {
    fixture.identityFixture.client.setQueryData(
      permissionKeys.workspaceProjectRoles("owner", "acme", 15, "workspace-revision"),
      {}
    );
    await new Promise((resolve) => setTimeout(resolve, 0));
  });
  await React.act(async () => {
    fixture.resolveDescription(new Uint8Array([4, 5]).buffer);
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.deepEqual([...fixture.props.getHydratedDocument()], [4, 5]);
  assert.equal(fixture.effects.writes, 0, "a role removed during fetch prevents the fallback write");
});

void test("editable fallback continues to persist the recovered document", async (t) => {
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
  assert.equal(savedDocuments[0].description_binary, "AQ==");
});
