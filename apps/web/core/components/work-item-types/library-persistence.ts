import type { WorkItemType } from "@/services/work-item-types.service";
import type { PreviewType, PreviewProperty } from "./preview-content";
import type { FormulaPersistence } from "./formula-editor";
import type { PropertyRemovalPersistence } from "./property-removal-dialog";
export type TypePersistence = {
  types: WorkItemType[];
  properties: PreviewProperty[];
  attachments: { type_id: string; property_ids: string[] }[];
  projects: { id: string; name: string }[];
  members: { id: string; name: string }[];
  saveProperty: (property: PreviewProperty, creating?: boolean, revision?: number) => Promise<void>;
  attachProperties: (typeId: string, properties: string[], revision?: number) => Promise<void>;
  propertyRemoval: PropertyRemovalPersistence;
  previewFormula: FormulaPersistence["preview"];
  projectCount: number;
  revision: number;
  save: (type: PreviewType, creating?: boolean, revision?: number) => Promise<void>;
  remove: (id: string, revision?: number) => Promise<void>;
};
export function capturePropertyEditors(persistence: TypePersistence | undefined, revision?: number) {
  if (!persistence) return {};
  const capturedRevision = revision ?? persistence.revision;
  return {
    review: { revision: capturedRevision, persistence: persistence.propertyRemoval },
    formula: {
      scope: persistence.propertyRemoval.scope,
      accountId: persistence.propertyRemoval.accountId,
      revision: capturedRevision,
      preview: persistence.previewFormula,
    },
  };
}
