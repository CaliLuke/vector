import assert from "node:assert/strict";
import { createRequire, registerHooks } from "node:module";
import { fileURLToPath, pathToFileURL } from "node:url";
import test from "node:test";
import { React, createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { createModuleStub, resolvePackageStub } from "./helpers/module-stubs.mjs";
import { registerWebModuleHooks } from "./helpers/web-module-hooks.mjs";

const requireFromWeb = createRequire(new URL("../../apps/web/package.json", import.meta.url));
globalThis.__subIssuesContentMobx = requireFromWeb("mobx");
globalThis.__subIssuesContentMobxUtils = requireFromWeb("mobx-utils");
globalThis.__subIssuesContentObserver = requireFromWeb("mobx-react").observer;
const reactModuleUrls = new Map(
  ["react", "react/jsx-runtime", "react-dom", "react-dom/client"].map((specifier) => [
    specifier,
    pathToFileURL(requireFromWeb.resolve(specifier)).href,
  ])
);
registerWebModuleHooks({ stubStoreContext: true });

const packageStubs = {
  "next/navigation":
    "export const useParams = () => ({ workspaceSlug: 'default' }); export const useRouter = () => ({ push() {}, replace() {} });",
  "mobx-react": "export const observer = globalThis.__subIssuesContentObserver;",
  mobx: "export const action = globalThis.__subIssuesContentMobx.action; export const computed = globalThis.__subIssuesContentMobx.computed; export const makeObservable = globalThis.__subIssuesContentMobx.makeObservable; export const observable = globalThis.__subIssuesContentMobx.observable; export const runInAction = globalThis.__subIssuesContentMobx.runInAction;",
  "mobx-utils": "export const computedFn = globalThis.__subIssuesContentMobxUtils.computedFn;",
  "@plane/i18n": "export const useTranslation = () => ({ t: key => key });",
  "@plane/propel/toast":
    "export const TOAST_TYPE = { ERROR: 'error', SUCCESS: 'success' }; export const setToast = () => {};",
};

function resolveSubIssuesAlias(specifier, context) {
  const sourceBySpecifier = {
    "@/services/issue":
      "export class IssueService { subIssues() { throw new Error('IssueService I/O must be provided by the test'); } } export class IssueArchiveService extends IssueService {}",
    "@/hooks/use-issue-peek-overview-redirection": "export default () => ({ handleRedirection() {} });",
    "@/components/issues/delete-issue-modal": "export const DeleteIssueModal = () => null;",
  };
  if (specifier === "@/lib/store-context") {
    const contextUrl = new URL("./helpers/work-item-store-context.mjs", import.meta.url).href;
    return createModuleStub(
      `export { StoreContext } from ${JSON.stringify(contextUrl)}; export const store = globalThis.__subIssuesContentStore;`
    );
  }
  if (Object.hasOwn(sourceBySpecifier, specifier)) return createModuleStub(sourceBySpecifier[specifier]);
  if (specifier === "../../issue-modal/modal" && context.parentURL?.includes("/sub-issues/content.")) {
    return createModuleStub("export const CreateUpdateIssueModal = () => null;");
  }
  return undefined;
}

function resolveSubIssuesPackage(specifier, context, nextResolve) {
  if (specifier === "@plane/propel/tooltip") {
    const url = new URL("../../packages/propel/src/tooltip/root.tsx", import.meta.url);
    return nextResolve(pathToFileURL(fileURLToPath(url)).href, context);
  }
  if (specifier === "@plane/ui" && context.parentURL?.includes("/sub-issues/")) {
    const files = [
      ["Collapsible", "../../packages/ui/src/collapsible/collapsible.tsx"],
      ["CustomMenu", "../../packages/ui/src/dropdowns/custom-menu.tsx"],
      ["ControlLink", "../../packages/ui/src/control-link/control-link.tsx"],
    ];
    const exports = files
      .map(
        ([name, path]) =>
          `export { ${name} } from ${JSON.stringify(pathToFileURL(fileURLToPath(new URL(path, import.meta.url))).href)};`
      )
      .join("\n");
    return createModuleStub(exports);
  }
  if (
    specifier === "@atlaskit/pragmatic-drag-and-drop-hitbox/tree-item" &&
    context.parentURL?.includes("issue-layouts/utils.")
  ) {
    const url = new URL(
      "../../apps/web/node_modules/@atlaskit/pragmatic-drag-and-drop-hitbox/dist/cjs/tree-item.js",
      import.meta.url
    );
    return nextResolve(pathToFileURL(fileURLToPath(url)).href, context);
  }
  return undefined;
}

registerHooks({
  resolve(specifier, context, nextResolve) {
    if (
      ["react", "react/jsx-runtime", "react-dom", "react-dom/client"].includes(specifier) &&
      context.parentURL?.includes("/packages/")
    ) {
      return nextResolve(reactModuleUrls.get(specifier), context);
    }
    const alias = resolveSubIssuesAlias(specifier, context);
    if (alias) return alias;
    const packageResolution = resolveSubIssuesPackage(specifier, context, nextResolve);
    if (packageResolution) return packageResolution;
    return resolvePackageStub(specifier, packageStubs, context, nextResolve);
  },
});

const { StoreContext } = await import("./helpers/work-item-store-context.mjs");
const { IssueSubIssuesStore } = await import("../../apps/web/core/store/issue/issue-details/sub_issues.store.ts");

test("spreadsheet header icons keep selection props off SVG elements", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { SpreadSheetPropertyIcon, SpreadSheetPropertyIconMap } =
    await import("../../apps/web/core/components/issues/issue-layouts/utils.tsx");
  const iconKeys = Object.keys(SpreadSheetPropertyIconMap);
  await environment.render(
    React.createElement(
      React.Fragment,
      null,
      ...iconKeys.map((iconKey) =>
        React.createElement(SpreadSheetPropertyIcon, {
          key: iconKey,
          iconKey,
          className: "header-icon",
          "aria-label": iconKey,
        })
      )
    )
  );
  for (const iconKey of iconKeys) {
    const svg = environment.container.querySelector(`svg[aria-label="${iconKey}"]`);
    assert.ok(svg, `${iconKey} renders an icon`);
    assert.equal(svg.hasAttribute("iconKey"), false, `${iconKey} does not leak its selector to the DOM`);
    assert.equal(svg.hasAttribute("iconkey"), false);
    assert.equal(svg.getAttribute("aria-label"), iconKey);
    assert.ok(svg.classList.contains("header-icon"));
  }
});

function deferred() {
  let resolve;
  const promise = new Promise((resolvePromise) => {
    resolve = resolvePromise;
  });
  return { promise, resolve };
}

function makeIssueDetail() {
  const issuesMap = {};
  const issueSubIssuesStore = new IssueSubIssuesStore(
    {
      rootIssueStore: {
        issues: {
          issuesMap,
          addIssue: (issues) => issues.forEach((issue) => (issuesMap[issue.id] = issue)),
          updateIssue: (issueId, updates) => Object.assign((issuesMap[issueId] ??= { id: issueId }), updates),
          getIssuesByIds: (issueIds) => issueIds.map((issueId) => issuesMap[issueId]).filter(Boolean),
        },
      },
    },
    "issues"
  );
  const issueDetail = {
    subIssues: issueSubIssuesStore,
    issue: { getIssueById: (issueId) => issuesMap[issueId] },
    toggleCreateIssueModal: () => {},
    toggleDeleteIssueModal: () => {},
    createSubIssues: async () => {},
    fetchSubIssues: issueSubIssuesStore.fetchSubIssues,
    updateSubIssue: async () => {},
    deleteSubIssue: async () => {},
    removeSubIssue: async () => {},
  };
  globalThis.__subIssuesContentMobx.runInAction(() => {
    issueSubIssuesStore.filters.subIssueFilters["parent-id"] = {
      displayProperties: {},
      filters: {},
      displayFilters: {},
    };
  });
  const project = { getProjectById: () => ({ id: "project-id", identifier: "TEST" }) };
  return {
    issueDetail,
    issueSubIssuesStore,
    issuesMap,
    storeValue: { issue: { issueDetail }, projectRoot: { project }, state: { getStateById: () => undefined } },
  };
}

test("duplicate StrictMode sub-item fetch completions keep the rendered child list visible", async (t) => {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const { issueSubIssuesStore, storeValue } = makeIssueDetail();
  globalThis.__subIssuesContentStore = storeValue;
  const { SubIssuesCollapsibleContent } =
    await import("../../apps/web/core/components/issues/issue-detail-widgets/sub-issues/content.tsx");
  const requests = [deferred(), deferred()];
  let requestIndex = 0;
  issueSubIssuesStore.issueService = {
    subIssues: () => requests[requestIndex++].promise,
  };
  await environment.render(
    React.createElement(
      React.StrictMode,
      null,
      React.createElement(
        StoreContext.Provider,
        { value: storeValue },
        React.createElement(SubIssuesCollapsibleContent, {
          workspaceSlug: "default",
          projectId: "project-id",
          parentIssueId: "parent-id",
          disabled: false,
        })
      )
    )
  );

  assert.equal(requestIndex, 2, "StrictMode should replay the initial effect and start duplicate requests");
  const response = {
    sub_issues: [
      { id: "child-1", name: "First child", project_id: "project-id", sequence_id: 1, sub_issues_count: 0 },
      { id: "child-2", name: "Second child", project_id: "project-id", sequence_id: 2, sub_issues_count: 0 },
    ],
    state_distribution: { backlog: ["child-1", "child-2"] },
  };
  await React.act(async () => {
    requests[0].resolve(response);
    requests[1].resolve(response);
    await Promise.all(requests.map(({ promise }) => promise));
    await new Promise((resolve) => setTimeout(resolve, 0));
  });

  assert.equal(environment.container.querySelector("a#issue-child-1")?.textContent, "First child");
  assert.equal(environment.container.querySelector("a#issue-child-2")?.textContent, "Second child");
  assert.deepEqual(issueSubIssuesStore.subIssuesByIssueId("parent-id"), ["child-1", "child-2"]);
  assert.deepEqual(issueSubIssuesStore.subIssueHelpersByIssueId("parent-id_root").issue_visibility, ["parent-id"]);
  assert.deepEqual(issueSubIssuesStore.subIssueHelpersByIssueId("parent-id_root").preview_loader, []);
});
