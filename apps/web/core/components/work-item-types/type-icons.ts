import { AlertTriangle, BookOpen, Bug, CheckSquare2, Layers } from "lucide-react";
import { LUCIDE_ICONS_LIST } from "@plane/propel/emoji-icon-picker";

// Keep existing saved identifiers while sharing the application's icon catalog.
export const typeIcons = [
  { name: "task", label: "Task", element: CheckSquare2 },
  { name: "bug", label: "Bug", element: Bug },
  { name: "story", label: "Story", element: BookOpen },
  { name: "epic", label: "Epic", element: Layers },
  { name: "incident", label: "Incident", element: AlertTriangle },
  ...LUCIDE_ICONS_LIST.map((icon) => ({
    name: icon.name,
    element: icon.element,
    label: icon.name.replace(/([a-z])([A-Z])/g, "$1 $2"),
  })),
];

export const typeIconComponents = Object.fromEntries(typeIcons.map((icon) => [icon.name, icon.element]));
