import { useState } from "react";
import { useTypePreview } from "./preview-provider";
import { CascadeChangeSummary } from "./cascade-change-summary";
import { propertyForEditor } from "./property-adapter";
import type { PreviewProperty } from "./preview-content";
import type { PropertyDefinition } from "@/services/work-item-properties.service";
import { PropertyRemovalDialog, type PropertyRemovalPersistence } from "./property-removal-dialog";

export type PropertyEditPersistence = { revision: number; persistence: PropertyRemovalPersistence };

function PropertyEditReview({
  property,
  original,
  review,
  onCancel,
  onClose,
}: {
  property: PropertyDefinition;
  original: PreviewProperty;
  review: PropertyEditPersistence;
  onCancel: () => void;
  onClose: () => void;
}) {
  const { announce } = useTypePreview();
  return (
    <PropertyRemovalDialog
      id={property.id}
      name={property.title}
      definition={property}
      changeSummary={<CascadeChangeSummary before={original} after={propertyForEditor(property, original.used)} />}
      revision={review.revision}
      persistence={review.persistence}
      onClose={onCancel}
      onCommitted={() => {
        announce(`${property.title} updated.`);
        onClose();
      }}
    />
  );
}

export function usePropertyEditReview(
  original: PreviewProperty,
  review: PropertyEditPersistence | undefined,
  onClose: () => void
) {
  const [proposed, setProposed] = useState<PropertyDefinition>();
  const dialog =
    proposed && review ? (
      <PropertyEditReview
        property={proposed}
        original={original}
        review={review}
        onCancel={() => setProposed(undefined)}
        onClose={onClose}
      />
    ) : undefined;
  return [dialog, setProposed] as const;
}
