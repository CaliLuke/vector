import { fileURLToPath } from "node:url";
import { seedInstanceInfo } from "./instance-info.mjs";
import { registerHooks } from "node:module";
import { QueryClient, QueryClientProvider, notifyManager, useMutation } from "@tanstack/react-query";
import { createReactDomTestEnvironment } from "./react-dom.mjs";
import { React } from "./react.mjs";
import { registerWebModuleHooks } from "./web-module-hooks.mjs";
import { resolvePackageStub } from "./module-stubs.mjs";
import { observeNativeForms } from "./native-form-observer.mjs";
import { seedWorkspaceQuery } from "./workspace-query-fixture.mjs";
registerWebModuleHooks();
export const formSources = Object.fromEntries(
  Object.entries({
    issue: "apps/web/core/components/issues/issue-modal/form.tsx",
    intake: "apps/web/core/components/inbox/modals/create-modal/create-root.tsx",
    project: "apps/web/core/components/projects/create/root.tsx",
    quickAdd: "apps/web/core/components/issues/issue-layouts/quick-add/form/root.tsx",
    page: "apps/web/core/components/pages/modals/page-form.tsx",
    estimate: "apps/web/core/components/estimates/create/modal.tsx",
    state: "apps/web/core/components/project-states/create-update/form.tsx",
    snooze: "apps/web/core/components/inbox/modals/snooze-issue-modal.tsx",
    invitation: "apps/web/core/components/workspace/invitation-selection-form.tsx",
  }).map(([name, file]) => [name, fileURLToPath(import.meta.resolve("../../../" + file))])
);
export const formObserver = observeNativeForms(Object.values(formSources), {
  "@tanstack/react-form": ["useForm"],
  "./form-state": ["useIssueDraftForm"],
  "@/components/issues/issue-modal/form-state": ["useIssueDraftForm"],
});
notifyManager.setScheduler(queueMicrotask);
export const state = {};
globalThis.issueFormWorkflow = state;
globalThis.issueFormReact = React;
globalThis.issueFormUseMutation = useMutation;
const stubs = {
  "@/services/work-item-schema.service": `export const workItemSchemaService={
    schema:async(_slug,input)=>globalThis.issueFormWorkflow.schema(input),
    preview:async(_slug,input,values)=>globalThis.issueFormWorkflow.preview(input,values)
  };`,
  "@/services/work-item-types.service": `export const typeErrorMessage=error=>error.message;export const workItemTypesService={
    projectCatalog:async()=>globalThis.issueFormWorkflow.types,
    identities:async()=>globalThis.issueFormWorkflow.types,
    catalog:async()=>globalThis.issueFormWorkflow.types
  };`,
  "@tanstack/react-router": `export const useParams=()=>({workspaceSlug:"workspace",projectId:"first"});export const useLocation=({select}={})=>{const location={pathname:"/workspace/projects/first"};return select?select(location):location;};export const useRouterState=({select}={})=>{const state={location:{href:"/workspace/projects/first"}};return select?select(state):state;};export const useRouter=()=>({navigate:async()=>{},state:{location:{href:"/workspace/projects/first"}}});`,
  "@plane/i18n": `export const useTranslation=()=>({t:key=>key});`,
  "@plane/propel/toast": `export const TOAST_TYPE={SUCCESS:"success",ERROR:"error"};export const setToast=()=>{};`,
  "@/hooks/use-platform-os": `export const usePlatformOS=()=>({isMobile:false});`,
  "@/hooks/context/use-issue-modal": `export const useIssueModal=()=>({allowedProjectIds:["first","second"],selectedParentIssue:null,setSelectedParentIssue:()=>{}});`,
  "@/hooks/use-intake": `const useMutation=globalThis.issueFormUseMutation;
    export const useIntakeMutation=(workspace,project)=>useMutation({mutationFn: async input=>({saved:await globalThis.issueFormWorkflow.intake.create(workspace,project,input.data)}),retry:false});`,
  "@/components/common/access-field": `export const AccessField=()=>null;`,
  "@/hooks/store/use-issue-detail": `export const useIssueDetail=()=>({issue:{getIssueById:()=>undefined}});`,
  "@/hooks/use-project-states":
    "export const useProjectStates=()=>({states:[],projectStates:[],getStateById:()=>undefined,getProjectStateIds:()=>[],getProjectDefaultStateId:()=>undefined,getProjectIntakeStateIds:()=>[],getIntakeStateById:()=>undefined,accessPending:false,accessError:false,accessDenied:false,retryAccess:async()=>{},refetch:async()=>({})});",
  "@/hooks/use-estimates": `export const useEstimateEditorKey=(workspace,project)=>workspace+"/"+project; export const useEstimateMutations=(workspace,project)=>({mutation:{mutateAsync:({data})=>globalThis.issueFormWorkflow.createEstimate(workspace,project,data)}});`,
  "@/components/editor/rich-text": `const React=globalThis.issueFormReact;export const RichTextEditor=React.forwardRef(({id,initialValue,onChange,editable,editorProps},ref)=>{
    const [value,setValue]=React.useState(initialValue);
    const listeners=React.useRef(new Set());
    const api=React.useMemo(()=>({
      isEditorReadyToDiscard:()=>globalThis.issueFormWorkflow.editorReadiness.get(id) ?? globalThis.issueFormWorkflow.editorReady !== false,
      focus:()=>document.getElementById(id)?.focus(),
      clearEditor:()=>setValue(""),setEditorValueAtCursorPosition:()=>{},
      setEditorValue:next=>{globalThis.issueFormWorkflow.editorWrites.push({id,value:next});setValue(next);for(const listener of listeners.current)listener();},
      onStateChange:listener=>{listeners.current.add(listener);return()=>listeners.current.delete(listener);},
      emit:()=>{for(const listener of listeners.current)listener();}
    }),[id]);
    React.useImperativeHandle(ref,()=>api,[api]);
    React.useEffect(()=>{globalThis.issueFormWorkflow.editors.set(id,api);return()=>{globalThis.issueFormWorkflow.editors.delete(id);};},[id,api]);
    return React.createElement("textarea",{...editorProps?.attributes,id,name:"description_html",value,disabled:!editable,onChange:event=>{setValue(event.target.value);api.emit();onChange({},event.target.value);}});
  });`,
  "@/components/core/modals/gpt-assistant-popover": `export const GptAssistantPopover=()=>null;`,
  "@/components/issues/issue-detail/issue-identifier": `export const IssueIdentifier=()=>null;`,
  "@/components/issues/work-item-selection-modal": `export const WorkItemSelectionModal=()=>null;`,
  "@/components/project/create/header": `export default ()=>null;`,
  "@/components/project/project-network-select": `export const ProjectNetworkSelect=()=>null;`,
  "@/helpers/cover-image.helper": `export const getRandomCoverImage=()=>"/static-cover.jpg";export const getCoverImageType=()=>"local_static";export const uploadCoverImage=async()=>"/uploaded-cover.jpg";`,
  "@/components/issues/create-issue-toast-action-items": `export const CreateIssueToastActionItems=()=>null;export const notifyIssueCreated=()=>{};`,
};
for (const [specifier, name, field, multiple] of [
  ["@/components/dropdowns/project/dropdown", "ProjectDropdown", "project_id", false],
  ["@/components/dropdowns/state/dropdown", "StateDropdown", "state_id", false],
  ["@/components/dropdowns/intake-state/dropdown", "IntakeStateDropdown", "state_id", false],
  ["@/components/dropdowns/priority", "PriorityDropdown", "priority", false],
  ["@/components/dropdowns/member/dropdown", "MemberDropdown", "assignee_ids", true],
  ["@/components/dropdowns/module/dropdown", "ModuleDropdown", "module_ids", true],
  ["@/components/issues/select", "IssueLabelSelect", "label_ids", true],
  ["@/components/dropdowns/cycle", "CycleDropdown", "cycle_id", false],
]) {
  stubs[specifier] =
    `const React=globalThis.issueFormReact;export const ${name}=({value,onChange,disabled})=>React.createElement("input",{name:${JSON.stringify(field)},value:${multiple ? '(value??[]).join(",")' : 'value??""'},disabled,onChange:event=>onChange(${multiple ? 'event.target.value.split(",").filter(Boolean)' : "event.target.value"})});`;
}
stubs["@/components/editor/rich-text/editor"] = stubs["@/components/editor/rich-text"];
stubs["@/components/dropdowns/date"] = "export const DateDropdown=()=>null;";
stubs["@/components/dropdowns/estimate"] = "export const EstimateDropdown=()=>null;";
registerHooks({
  resolve(specifier, context, nextResolve) {
    if (specifier.endsWith("create-issue-toast-action-items"))
      specifier = "@/components/issues/create-issue-toast-action-items";
    return resolvePackageStub(specifier, stubs, context, nextResolve);
  },
});
export async function setup(t) {
  const env = await createReactDomTestEnvironment();
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: Infinity }, mutations: { retry: false } },
  });
  seedInstanceInfo(client, { has_llm_configured: false });
  await seedWorkspaceQuery(client);
  const { permissionKeys } = await import("../../../apps/web/core/queries/permissions.ts");
  client.setQueryData(permissionKeys.workspaceMembership("workspace-query-test-user", "workspace"), {
    is_active: true,
    role: 20,
    updated_at: "v1",
  });
  client.setQueryData(permissionKeys.workspaceProjectRoles("workspace-query-test-user", "workspace", 20, "v1"), {
    first: 20,
    second: 20,
  });
  const { projectKeys } = await import("../../../apps/web/core/queries/projects.ts");
  const { ProjectService } = await import("../../../apps/web/core/services/project/project.service.ts");
  const projectScope = {
    accountId: "workspace-query-test-user",
    sessionGeneration: 0,
    workspaceSlug: "workspace",
    workspaceRole: 20,
    membershipRevision: "v1",
    projectRoles: [
      ["first", 20],
      ["second", 20],
    ],
  };
  const projects = ["first", "second"].map((id) => ({
    id,
    identifier: id.toUpperCase(),
    cycle_view: true,
    module_view: true,
  }));
  client.setQueryData(projectKeys.workspace(projectScope, "full"), projects);
  for (const project of projects)
    client.setQueryData(
      projectKeys.detail({ ...projectScope, projectId: project.id, projectRole: 20, projectMembershipRole: 20 }),
      project
    );
  t.mock.method(ProjectService.prototype, "getProjects", async () => projects);
  t.mock.method(ProjectService.prototype, "getProject", async (_workspace, id) =>
    projects.find((project) => project.id === id)
  );
  t.after(async () => {
    await env.render(null);
    client.clear();
    await env.unmount();
  });
  state.editorReady = true;
  state.editorWrites = [];
  state.editorReadiness = new Map();
  state.editors = new Map();
  state.types = { enabled: false, types: [] };
  state.schema = async () => ({ enabled: false, type_id: "", properties: [], members: [], releases: [], computed: {} });
  state.preview = async () => ({ computed: {} });
  return {
    ...env,
    client,
    render: (element) => env.render(React.createElement(QueryClientProvider, { client }, element)),
  };
}
export function issueProps(overrides = {}) {
  return {
    projectId: "first",
    isDraft: false,
    issueTitleRef: React.createRef(),
    isCreateMoreToggleEnabled: false,
    onCreateMoreToggleChange: () => {},
    onAssetUpload: () => {},
    onClose: () => {},
    onSubmit: async () => {},
    onPublish: async (payload) => state.move("workspace", "draft", payload),
    data: { description_html: "<p>Original</p>" },
    ...overrides,
  };
}
