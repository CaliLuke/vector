import type { TWorkItemPropertyKind } from "@plane/types";

export const propertyKinds = [
  "Text",
  "Number",
  "Dropdown",
  "Boolean",
  "Date",
  "Member picker",
  "Release picker",
  "Rich text",
  "URL",
  "Formula",
  "Cascading",
] as const;
export type PropertyKind = TWorkItemPropertyKind;
export type PreviewOption = {
  icon?: string;
  id: string;
  name: string;
  parent: string;
  level: number;
  nextLabel: string;
};
export type PreviewContext = {
  id: string;
  name: string;
  projects: string[];
  types: string[];
  required: boolean | null;
  active: boolean | null;
  defaultValue?: string | null;
  format?: string | null;
  options?: PreviewOption[];
};
export type PreviewProperty = {
  internalName?: string;
  icon?: string;
  id: string;
  name: string;
  description: string;
  kind: PropertyKind;
  active: boolean;
  required: boolean;
  format: string;
  multiple: boolean;
  defaultValue: string;
  options: PreviewOption[];
  levels: { id: string; name: string; required: boolean }[];
  expression: string;
  result: string;
  contexts: PreviewContext[];
  used: boolean;
};
export type PreviewType = {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  active: boolean;
  mandatory: boolean;
  properties: string[];
  level: number;
};
export type PreviewItem = {
  draft?: boolean;
  id: string;
  name: string;
  type: string;
  project: string;
  parent: string;
  description: string;
  values: Record<string, string | string[] | boolean>;
  activity: string[];
};
type PreviewProject = { id: string; name: string; types: string[] };
export const previewPeople = ["Alex Morgan", "Jordan Lee", "Sam Rivera"];
export const previewReleases = ["Atlas 2.4", "Atlas 2.5", "Atlas 3.0"];
export function newProperty(kind: PropertyKind = "Text"): PreviewProperty {
  return {
    id: "",
    name: "",
    description: "",
    kind,
    active: true,
    required: false,
    format: "Single line",
    multiple: false,
    defaultValue: "",
    options: [],
    levels: [
      { id: "level-1", name: "Level 1", required: false },
      { id: "level-2", name: "Level 2", required: false },
    ],
    expression: "",
    result: "Number",
    contexts: [],
    used: false,
  };
}
const option = (id: string, name: string, level = 0, parent = "", nextLabel = ""): PreviewOption => ({
  id,
  name,
  level,
  parent,
  nextLabel,
});
export function createPreviewContent() {
  const properties: PreviewProperty[] = [
    {
      ...newProperty("Dropdown"),
      id: "severity",
      name: "Severity",
      description: "How much does this affect our customers?",
      required: true,
      used: true,
      options: [option("critical", "Critical"), option("major", "Major"), option("minor", "Minor")],
      contexts: [
        { id: "mobile-bug", name: "Mobile bugs", projects: ["mobile"], types: ["bug"], required: false, active: null },
      ],
    },
    {
      ...newProperty("Text"),
      id: "summary",
      name: "Customer impact",
      description: "Describe who is affected.",
      format: "Paragraph",
      used: true,
    },
    { ...newProperty("Number"), id: "effort", name: "Effort", defaultValue: "3", used: true },
    { ...newProperty("Boolean"), id: "blocking", name: "Release blocker", used: true },
    { ...newProperty("Date"), id: "review", name: "Review date", format: "Jan 15, 2025" },
    { ...newProperty("Member picker"), id: "reviewer", name: "Reviewer", multiple: true },
    { ...newProperty("Release picker"), id: "release", name: "Target release", multiple: true },
    {
      ...newProperty("Rich text"),
      id: "reproduction",
      name: "Steps to reproduce",
      description: "Include expected and actual behavior.",
    },
    { ...newProperty("URL"), id: "design", name: "Design reference" },
    { ...newProperty("Formula"), id: "score", name: "Priority score", expression: "{{effort}} * 2", used: true },
    {
      ...newProperty("Cascading"),
      id: "area",
      name: "Product area",
      levels: [
        { id: "product", name: "Product", required: true },
        { id: "component", name: "Component", required: false },
        { id: "surface", name: "Surface", required: false },
      ],
      options: [
        option("web", "Web", 0, "", "Service"),
        option("app", "App"),
        option("api", "API", 1, "web"),
        option("auth", "Authentication", 2, "api"),
        option("ui", "Interface", 1, "web"),
        option("ios", "iOS", 1, "app"),
      ],
    },
    {
      ...newProperty("Text"),
      id: "guidance",
      name: "Triage guidance",
      format: "Read-only",
      defaultValue: "Include the customer impact before assigning a priority.",
    },
  ];
  const types: PreviewType[] = [
    {
      id: "task",
      name: "Task",
      description: "The default type for everyday work.",
      icon: "task",
      color: "#64748b",
      active: true,
      mandatory: true,
      properties: ["effort", "reviewer", "design"],
      level: 0,
    },
    {
      id: "bug",
      name: "Bug",
      description: "Track defects and get things working as expected.",
      icon: "bug",
      color: "#e45470",
      active: true,
      mandatory: false,
      properties: [
        "severity",
        "summary",
        "blocking",
        "reproduction",
        "area",
        "release",
        "review",
        "effort",
        "score",
        "guidance",
      ],
      level: 1,
    },
    {
      id: "story",
      name: "Story",
      description: "A meaningful improvement for someone using the product.",
      icon: "story",
      color: "#8b5cf6",
      active: true,
      mandatory: false,
      properties: ["summary", "effort", "design"],
      level: 1,
    },
    {
      id: "epic",
      name: "Epic",
      description: "A larger outcome delivered through stories and bugs.",
      icon: "epic",
      color: "#d69a23",
      active: true,
      mandatory: false,
      properties: ["summary", "review"],
      level: 2,
    },
    {
      id: "incident",
      name: "Incident",
      description: "Coordinate a response when service is disrupted.",
      icon: "incident",
      color: "#e9693b",
      active: false,
      mandatory: false,
      properties: ["severity", "summary", "reviewer"],
      level: 1,
    },
  ];
  const projects: PreviewProject[] = [
    { id: "platform", name: "Web Platform", types: ["task", "bug", "story", "epic"] },
    { id: "mobile", name: "Mobile App", types: ["task", "bug"] },
  ];
  const items: PreviewItem[] = [
    {
      id: "WEB-142",
      name: "Keep the session when switching devices",
      type: "bug",
      project: "platform",
      parent: "WEB-100",
      description: "People should be able to continue their work across devices without signing in again.",
      values: {
        severity: "major",
        summary: "Returning users have to sign in twice.",
        effort: "3",
        blocking: true,
        area: ["web", "api", "auth"],
        reproduction:
          "1. Sign in on a laptop.\n2. Open the same workspace on a phone.\n\nExpected: your session is restored.",
        release: ["Atlas 2.5"],
      },
      activity: ["Alex Morgan changed Severity from Minor to Major.", "Jordan Lee created this work item."],
    },
    {
      id: "WEB-100",
      name: "A more reliable sign-in experience",
      type: "epic",
      project: "platform",
      parent: "",
      description: "Make returning to work feel effortless.",
      values: { summary: "Reduce interruptions for returning users." },
      activity: [],
    },
    {
      id: "WEB-143",
      name: "Add session renewal coverage",
      type: "task",
      project: "platform",
      parent: "WEB-142",
      description: "Cover session renewal across browser tabs.",
      values: { effort: "2", reviewer: ["Alex Morgan"] },
      activity: [],
    },
  ];
  return { types, properties, projects, items, enabled: true, hierarchyEnabled: true, hierarchyDefaultLevel: 0 };
}
export type PreviewContent = ReturnType<typeof createPreviewContent>;
