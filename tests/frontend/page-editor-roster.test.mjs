import assert from "node:assert/strict";
import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { act, React } from "./helpers/react.mjs";
import { registerPermissionUiStubs } from "./helpers/permission-ui-fixture.mjs";

const rosterState = { query: { isSuccess: false, isError: false }, membersById: undefined };
globalThis.pageEditorRosterState = rosterState;
const exportState = {
  query: { isSuccess: false, isError: false, isFetching: false, refetch: async () => {} },
  retries: 0,
  conversionError: false,
  closes: 0,
  toasts: [],
  react: React,
};
globalThis.pageEditorExportState = exportState;
const hookUrl = new URL("../../apps/web/core/hooks/use-parse-editor-content.ts", import.meta.url).href;
registerPermissionUiStubs("/use-parse-editor-content.ts", {
  "@/hooks/use-member-rosters": `const state=globalThis.pageEditorRosterState;export const useWorkspaceMemberRoster=()=>({...state,isError:state.query.isError??false,isLoading:state.query.isLoading??false,retry:async()=>{}});`,
  "@/hooks/use-additional-editor-mention": `export const useAdditionalEditorMention=()=>({parseAdditionalEditorContent:()=>undefined});`,
  "@plane/utils/file": "export const getBase64Image=async value=>value;",
  "@plane/utils/editor/common": 'export const getEditorAssetSrc=()=>"";',
});
registerPermissionUiStubs("/export-page-modal.tsx", {
  "@tanstack/react-router": `export const useParams=()=>({workspaceSlug:"acme",projectId:"p1"});`,
  "@react-pdf/renderer": `export const pdf=()=>({toBlob:async()=>new Blob()});`,
  "@tanstack/react-form": `const state=globalThis.pageEditorExportState;export function useForm(options){state.form??={isSubmitting:false,isSubmitSuccessful:false,errors:[],values:options.defaultValues};state.options=options;const Field=({name,children})=>children({state:{value:state.form.values[name]},handleChange:value=>{state.form.values[name]=value;}});return{store:state.form,Field,handleSubmit:async()=>{state.form.isSubmitting=true;state.form.isSubmitSuccessful=false;try{const error=state.options.validators?.onSubmit?.({value:state.form.values});if(error){state.form.errors=[error];return;}state.form.errors=[];await state.options.onSubmit({value:state.form.values});state.form.isSubmitSuccessful=true;}catch(error){state.form.errors=[error];throw error;}finally{state.form.isSubmitting=false;}}};}export const useStore=(store,selector)=>selector(store);`,
  "@plane/propel/button": `const React=globalThis.pageEditorExportState.react;export const Button=({children,variant,size,loading,...props})=>React.createElement("button",props,children);`,
  "@plane/propel/toast": `const state=globalThis.pageEditorExportState;export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=toast=>state.toasts.push(toast);`,
  "@plane/ui": `const React=globalThis.pageEditorExportState.react;export const CustomSelect=Object.assign(({children,label})=>React.createElement("div",null,label,children),{Option:({children})=>React.createElement("span",null,children)});export const EModalPosition={CENTER:"center"};export const EModalWidth={SM:"sm"};export const ModalCore=({children})=>React.createElement(React.Fragment,null,children);`,
  "@/components/editor/pdf": `export const PDFDocument=()=>null;`,
  "@/hooks/use-parse-editor-content": `const state=globalThis.pageEditorExportState;export const useParseEditorContent=()=>({memberRosterQuery:state.query,memberRosterIsLoading:state.query.isLoading??false,memberRosterIsError:state.query.isError??false,retryMemberRoster:()=>state.query.refetch(),replaceCustomComponentsFromHTMLContent:async({htmlContent})=>{if(state.conversionError)throw new Error("conversion failed");return htmlContent;},replaceCustomComponentsFromMarkdownContent:({markdownContent})=>{if(state.conversionError)throw new Error("conversion failed");return markdownContent;}});`,
});
const { useParseEditorContent } = await import(hookUrl);
const modalUrl = new URL("../../apps/web/core/components/pages/modals/export-page-modal.tsx", import.meta.url).href;
const { ExportPageModal } = await import(modalUrl);

function PageEditorRosterProbe() {
  const parsed = useParseEditorContent({ workspaceSlug: "acme" });
  React.useEffect(() => {
    globalThis.pageEditorParsedResult = parsed;
  }, [parsed]);
  return React.createElement("output", null, parsed.memberRosterQuery.isSuccess ? "ready" : "waiting");
}

const getParsed = () => globalThis.pageEditorParsedResult;

await test("page export parsing waits for roster success and preserves unknown mention identity", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const originalDOMParser = globalThis.DOMParser;
  globalThis.DOMParser = environment.container.ownerDocument.defaultView.DOMParser;
  t.after(() => {
    if (originalDOMParser) globalThis.DOMParser = originalDOMParser;
    else delete globalThis.DOMParser;
  });
  globalThis.pageEditorParsedResult = undefined;
  await environment.render(React.createElement(PageEditorRosterProbe));
  assert.throws(
    () =>
      getParsed().replaceCustomComponentsFromMarkdownContent({
        markdownContent: '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>',
      }),
    /must load before exporting/
  );
  assert.deepEqual(
    getParsed().getEditorMetaData(
      '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>'
    ).user_mentions,
    [{ id: "ada", display_name: "ada", url: "http://localhost/acme/profile/ada" }]
  );
  assert.deepEqual(getParsed().getEditorMetaData("<p>No mentions</p>").user_mentions, []);
  rosterState.query = { isSuccess: false, isError: true };
  await environment.render(React.createElement(PageEditorRosterProbe));
  assert.throws(
    () =>
      getParsed().replaceCustomComponentsFromMarkdownContent({
        markdownContent: '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>',
      }),
    /must load before exporting/
  );
  assert.deepEqual(getParsed().getEditorMetaData("<p>No mentions</p>").user_mentions, []);

  rosterState.query = { isSuccess: true, isError: false };
  rosterState.membersById = new Map([["ada", { member: { id: "ada", display_name: "Ada" } }]]);
  await environment.render(React.createElement(PageEditorRosterProbe));
  assert.equal(environment.container.textContent, "ready");
  const mixedMentionMetadata = getParsed().getEditorMetaData(
    '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>' +
      '<mention-component entity_identifier="ada" entity_name="issue"></mention-component>' +
      '<mention-component entity_identifier="ada" entity_name="project"></mention-component>' +
      '<mention-component entity_identifier="ada"></mention-component>'
  );
  assert.deepEqual(
    mixedMentionMetadata.user_mentions.map(({ id, display_name }) => ({ id, display_name })),
    [
      { id: "ada", display_name: "Ada" },
      { id: "ada", display_name: "Ada" },
    ]
  );
  const resolvedHtml = await getParsed().replaceCustomComponentsFromHTMLContent({
    htmlContent: '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>',
  });
  assert.match(resolvedHtml, /@Ada/);
  assert.match(
    getParsed().replaceCustomComponentsFromMarkdownContent({
      markdownContent: '<mention-component entity_identifier="ada" entity_name="user_mention"></mention-component>',
    }),
    /\[Ada\]\(http:\/\/localhost\/acme\/profile\/ada\)/
  );

  rosterState.membersById = new Map();
  await environment.render(React.createElement(PageEditorRosterProbe));
  assert.match(
    getParsed().replaceCustomComponentsFromMarkdownContent({
      markdownContent:
        '<mention-component entity_identifier="unknown-user" entity_name="user_mention"></mention-component>',
    }),
    /\[unknown-user\]\(http:\/\/localhost\/acme\/profile\/unknown-user\)/
  );
  const html = await getParsed().replaceCustomComponentsFromHTMLContent({
    htmlContent: '<mention-component entity_identifier="unknown-user" entity_name="user_mention"></mention-component>',
  });
  assert.match(html, /@unknown-user/);
  const metadata = getParsed().getEditorMetaData(
    '<mention-component entity_identifier="unknown-user" entity_name="user_mention"></mention-component>'
  );
  assert.deepEqual(
    metadata.user_mentions.map(({ id, display_name }) => ({ id, display_name })),
    [{ id: "unknown-user", display_name: "unknown-user" }]
  );
});

await test("page export validates roster readiness and preserves failed submit state for retry", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const documentWindow = environment.container.ownerDocument.defaultView;
  const originalDOMParser = globalThis.DOMParser;
  globalThis.DOMParser = documentWindow.DOMParser;
  const originalCreateObjectURL = URL.createObjectURL?.bind(URL);
  const originalRevokeObjectURL = URL.revokeObjectURL?.bind(URL);
  const anchorClick = documentWindow.HTMLAnchorElement.prototype.click.bind(documentWindow.HTMLAnchorElement.prototype);
  URL.createObjectURL = () => "blob:page-export";
  URL.revokeObjectURL = () => {};
  environment.container.ownerDocument.defaultView.HTMLAnchorElement.prototype.click = () => {};
  t.after(() => {
    if (originalCreateObjectURL) URL.createObjectURL = originalCreateObjectURL;
    else delete URL.createObjectURL;
    if (originalRevokeObjectURL) URL.revokeObjectURL = originalRevokeObjectURL;
    else delete URL.revokeObjectURL;
    documentWindow.HTMLAnchorElement.prototype.click = anchorClick;
    if (originalDOMParser) globalThis.DOMParser = originalDOMParser;
    else delete globalThis.DOMParser;
  });
  exportState.query = {
    isSuccess: false,
    isError: false,
    isFetching: false,
    refetch: async () => {
      exportState.retries += 1;
    },
  };
  exportState.retries = 0;
  exportState.conversionError = false;
  exportState.closes = 0;
  exportState.toasts = [];
  exportState.form = undefined;
  const submit = async () => {
    await act(async () => {
      environment.container
        .querySelector("form")
        .dispatchEvent(new Event("submit", { bubbles: true, cancelable: true }));
      await new Promise((resolve) => setImmediate(resolve));
    });
  };
  const render = () =>
    environment.render(
      React.createElement(ExportPageModal, {
        editorRef: null,
        isOpen: true,
        onClose: () => {
          exportState.closes += 1;
        },
        pageTitle: "Roster export",
      })
    );

  await render();
  let exportButton = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Export"
  );
  assert.equal(exportButton?.disabled, true);
  assert.match(environment.container.textContent, /Loading workspace members/);
  await submit();
  assert.equal(exportState.form.isSubmitSuccessful, false);
  assert.equal(exportState.closes, 0);
  assert.equal(
    exportState.toasts.some((toast) => toast.type === "success"),
    false
  );

  exportState.query = {
    isSuccess: false,
    isError: true,
    isFetching: false,
    refetch: async () => {
      exportState.retries += 1;
    },
  };
  await render();
  assert.ok(environment.container.querySelector('[role="alert"]'));
  const retry = [...environment.container.querySelectorAll("button")].find((button) => button.textContent === "Retry");
  assert.ok(retry);
  await act(async () => retry.click());
  assert.equal(exportState.retries, 1);
  await submit();
  assert.equal(exportState.form.isSubmitSuccessful, false);
  assert.equal(exportState.closes, 0);

  exportState.query = { isSuccess: true, isError: false, isFetching: false, refetch: async () => {} };
  await render();
  exportButton = [...environment.container.querySelectorAll("button")].find(
    (button) => button.textContent === "Export"
  );
  assert.equal(exportButton?.disabled, false);

  exportState.conversionError = true;
  const originalConsoleError = console.error;
  console.error = () => {};
  await submit();
  console.error = originalConsoleError;
  assert.equal(exportState.form.isSubmitSuccessful, false);
  assert.equal(exportState.closes, 0);
  assert.ok(exportState.toasts.some((toast) => toast.type === "error"));
  assert.equal(
    exportState.toasts.some((toast) => toast.type === "success"),
    false
  );

  exportState.conversionError = false;
  await render();
  await submit();
  assert.equal(exportState.form.isSubmitSuccessful, true);
  assert.equal(exportState.closes, 1);
  assert.ok(exportState.toasts.some((toast) => toast.type === "success"));
});
