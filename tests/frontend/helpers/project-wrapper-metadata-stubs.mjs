/** Existing metadata fixtures unrelated to project access and preference behavior. */
export const projectWrapperMetadataStubs = {
  "@/hooks/use-estimates": `export const useProjectEstimates=()=>({});`,
  "@/hooks/use-labels": `export const useProjectLabels=()=>({});`,
  "@/hooks/use-cycles": `export const useProjectCycles=()=>({});`,
  "@/hooks/use-modules": `export const useProjectModules=()=>({});`,
  "@/hooks/use-project-states":
    "export const useProjectStates=()=>({states:[],projectStates:[],getStateById:()=>undefined,getProjectStateIds:()=>[],getProjectDefaultStateId:()=>undefined,getProjectIntakeStateIds:()=>[],getIntakeStateById:()=>undefined,accessPending:false,accessError:false,accessDenied:false,retryAccess:async()=>{},refetch:async()=>({})});",
};
