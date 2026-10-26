import type { TIssue } from "@plane/types";
import type { ItemSchema } from "@/services/work-item-schema.service";

/** Only displayed rich-text content can reference an upload. Plain text may
 * contain literal markup; deleted or hidden images must not finalize uploads. */
export function itemAssetReferences(values: Partial<TIssue>, schema: ItemSchema | undefined): string[] {
  const sources = [values.description_html];
  for (const property of schema?.properties ?? []) {
    if (property.kind !== "Rich text") continue;
    const value = values.custom_properties?.[property.id];
    if (typeof value === "string") sources.push(value);
  }
  const ids = new Set<string>();
  for (const html of sources) {
    if (!html) continue;
    const template = document.createElement("template");
    template.innerHTML = html;
    for (const image of template.content.querySelectorAll("image-component[src], img[src]")) {
      const src = image.getAttribute("src");
      if (src) ids.add(src);
    }
  }
  return [...ids];
}
