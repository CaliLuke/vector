import assert from "node:assert/strict";
import test from "node:test";

import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";

const stubs = {
  "@tanstack/react-router": `import {React} from "${import.meta.resolve("./helpers/react.mjs")}";export const Link=({to,children,...props})=>React.createElement("a",{...props,href:to},children);`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/icons": `export const EditIcon=()=>null;export const CloseIcon=()=>null;`,
  "@plane/propel/tooltip": `import {React} from "${import.meta.resolve("./helpers/react.mjs")}";export const Tooltip=({children})=>React.createElement(React.Fragment,null,children);`,
  "@plane/utils/common": `export const cn=(...values)=>values.filter(value=>typeof value==="string").join(" ");`,
  "@/providers/issue-detail-ui-state": `export const useIssueDetailUiState=()=>({isParentIssueModalOpen:null,toggleParentIssueModal:()=>{}});`,
  "@/hooks/use-projects": `export const useWorkspaceProjectDetails=()=>({projects:globalThis.parentSelectFixture.projects});`,
  "@/hooks/use-platform-os": `export const usePlatformOS=()=>({isMobile:false});`,
  "@/components/issues/issue-detail/issue-identifier": `import {React} from "${import.meta.resolve("./helpers/react.mjs")}";export const IssueIdentifier=props=>{globalThis.parentSelectFixture.identifiers.push(props);return React.createElement("span",{"data-testid":"parent-identifier"},props.projectIdentifier+"-"+props.issueSequenceId);};`,
  "../work-item-selection-modal": `export const WorkItemSelectionModal=()=>null;`,
};
registerWebTestModules(stubs);

const { IssueParentSelect } = await import("../../apps/web/core/components/issues/issue-detail/parent-select.tsx");

function fixture() {
  return {
    projects: [{ id: "parent-project", identifier: "PARENT" }],
    identifiers: [],
  };
}

async function render(t, initialProps) {
  const environment = await createReactDomTestEnvironment();
  t.after(() => environment.unmount());
  const view = {
    environment,
    render: (nextProps = initialProps) => environment.render(React.createElement(IssueParentSelect, nextProps)),
  };
  await view.render();
  return view;
}

function parentProps(parentIssue, hasParentRelation, workItemLink) {
  return {
    issueId: "child",
    projectId: "child-project",
    workspaceSlug: "workspace",
    parentIssue,
    hasParentRelation,
    workItemLink,
    handleParentIssue: async () => {},
    handleRemoveSubIssue: async () => {},
  };
}

await test("parent selection waits for valid parent identity before rendering a link", async (t) => {
  globalThis.parentSelectFixture = fixture();
  const view = await render(t, parentProps({}, true, undefined));

  assert.equal(view.environment.container.querySelector("a"), null);
  assert.equal(view.environment.container.querySelector('[data-testid="parent-identifier"]'), null);
  assert.doesNotMatch(view.environment.container.textContent, /undefined/);
  assert.match(view.environment.container.textContent, /common\.parent/);

  await view.render(
    parentProps(
      { id: "parent", project_id: "parent-project", sequence_id: 17, name: "Parent task" },
      true,
      "/workspace/projects/parent-project/issues/parent"
    )
  );

  assert.equal(
    view.environment.container.querySelector("a")?.getAttribute("href"),
    "/workspace/projects/parent-project/issues/parent"
  );
  assert.equal(view.environment.container.querySelector('[data-testid="parent-identifier"]')?.textContent, "PARENT-17");
  assert.doesNotMatch(view.environment.container.textContent, /undefined/);
});

await test("empty expanded parent without a parent id renders the add-parent action", async (t) => {
  globalThis.parentSelectFixture = fixture();
  const view = await render(t, parentProps({}, false, undefined));

  assert.equal(view.environment.container.querySelector("a"), null);
  assert.equal(view.environment.container.querySelector('[data-testid="parent-identifier"]'), null);
  assert.match(view.environment.container.textContent, /issue\.add\.parent/);
  assert.doesNotMatch(view.environment.container.textContent, /undefined/);
});
