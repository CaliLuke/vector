import { registerWebTestModules } from "./helpers/web-test-modules.mjs";
import assert from "node:assert/strict";

import test from "node:test";
import { createReactDomTestEnvironment } from "./helpers/react-dom.mjs";
import { React } from "./helpers/react.mjs";
import { createRouterFixture } from "./helpers/router.mjs";

globalThis.archiveHeaderReact = React;
const stubs = {
  "@plane/propel/icons":
    "const Icon=()=>null; export const ArchiveIcon=Icon,CycleIcon=Icon,ModuleIcon=Icon,WorkItemsIcon=Icon;",
  "@plane/propel/tooltip": "export const Tooltip=({children})=>children;",
  "@plane/ui": `const Part=({children})=>children;
    export const Header=Object.assign(Part,{LeftItem:Part});
    export const Breadcrumbs=Object.assign(Part,{Item:({component})=>component});
    export const Avatar=Part,AvatarGroup=Part;`,
  "@/components/common/breadcrumb-link": `const React=globalThis.archiveHeaderReact;
    export const BreadcrumbLink=({href,label})=>React.createElement('a',{href},label);`,
  "@/components/breadcrumbs/common": `const React=globalThis.archiveHeaderReact;
    export const CommonProjectBreadcrumbs=({workspaceSlug,projectId})=>React.createElement('output',null,workspaceSlug+'/'+projectId);`,
  "@/hooks/use-issue-list-count":
    "export const useIssueListCount=()=>({count:0,isPending:false,isError:false,retry:async()=>{}});",
  "@/hooks/use-projects": `export const useRouteProject=()=>({isPending:false});export const useProjectDetail=()=>({project:undefined,accessDenied:false,accessError:false,isError:false,scope:{},retryAccess:async()=>{}});`,
  "@/hooks/use-platform-os": "export const usePlatformOS=()=>({isMobile:false});",
};
registerWebTestModules(stubs);

await Promise.all(
  ["issues", "cycles", "modules"].map((activeTab) =>
    test(`project ${activeTab} archives read params without requiring the active-project layout`, async (t) => {
      const environment = await createReactDomTestEnvironment();
      t.after(async () => {
        await environment.unmount();
      });
      const { ProjectArchivesHeader } =
        await import("../../apps/web/app/(all)/[workspaceSlug]/(projects)/projects/(detail)/[projectId]/archives/header.tsx");
      const fixture = createRouterFixture(`/acme/projects/project-1/archives/${activeTab}/`);
      await fixture.render(environment, React.createElement(ProjectArchivesHeader, { activeTab }));
      assert.equal(environment.container.querySelector("output")?.textContent, "acme/project-1");
      assert.equal(
        environment.container.querySelector("a[href]")?.getAttribute("href"),
        "/acme/projects/project-1/archives/issues"
      );
    })
  )
);
