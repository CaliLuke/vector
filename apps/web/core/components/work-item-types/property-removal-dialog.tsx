import { useId, useState, type ReactNode } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useQuery } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import type {
  PropertyDefinition,
  ReviewedPropertyChange,
  PropertyRemovalReview,
} from "@/services/work-item-properties.service";
import { typeErrorMessage } from "@/services/work-item-types.service";
import { Field, inputStyle, PreviewDialog } from "./controls";

export type PropertyRemovalPersistence = {
  scope: string;
  accountId: string | undefined;
  preview: (input: ReviewedPropertyChange, signal?: AbortSignal) => Promise<PropertyRemovalReview>;
  commit: (input: ReviewedPropertyChange) => Promise<unknown>;
  refreshRevision: () => Promise<number>;
};

export function PropertyRemovalDialog({
  id,
  name,
  typeId,
  revision,
  persistence,
  onClose,
  definition,
  onCommitted,
  changeSummary,
}: {
  id: string;
  name: string;
  typeId?: string;
  revision: number;
  persistence: PropertyRemovalPersistence;
  onClose: () => void;
  definition?: PropertyDefinition;
  onCommitted?: () => void;
  changeSummary?: ReactNode;
}) {
  const dialogId = useId();
  const [reviewRevision, setReviewRevision] = useState(revision);
  const [refreshing, setRefreshing] = useState(false);
  const [uncertain, setUncertain] = useState<ReviewedPropertyChange>();
  const form = useForm({
    defaultValues: { permanent: false, confirmation: "" },
    validators: {
      onSubmit: ({ value }): string | undefined =>
        requiresConfirmation(value.permanent, definition, review.data) && value.confirmation !== "Confirm"
          ? "Type Confirm to continue."
          : undefined,
    },
    onSubmit: async ({ formApi }) => {
      if (!review.data || review.isFetching) return;
      const submitted = uncertain ?? { ...command, review_id: review.data.id };
      try {
        await persistence.commit(submitted);
        setUncertain(undefined);
        (onCommitted ?? onClose)();
      } catch (cause) {
        setUncertain(isDefiniteRemovalRejection(cause) ? undefined : submitted);
        formApi.setErrorMap({ onSubmit: typeErrorMessage(cause) });
      }
    },
  });
  const permanent = useStore(form.store, (state) => state.values.permanent);
  const pending = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const command: ReviewedPropertyChange = definition
    ? { action: "update", definition, revision: reviewRevision }
    : { action: typeId ? "detach" : "delete", id, type_id: typeId, revision: reviewRevision, permanent };
  const review = useQuery({
    queryKey: ["property-removal-review", persistence.scope, persistence.accountId, dialogId, command],
    queryFn: ({ signal }) => persistence.preview(command, signal),
    staleTime: Infinity,
    gcTime: 0,
    retry: false,
    refetchOnWindowFocus: false,
    refetchOnReconnect: false,
  });
  const refresh = async () => {
    setRefreshing(true);
    form.setErrorMap({ onSubmit: undefined });
    try {
      const next = await persistence.refreshRevision();
      if (definition && next !== reviewRevision)
        throw new Error("Configuration changed. Close and reopen this property before reapplying your changes.");
      if (next === reviewRevision) await review.refetch();
      else setReviewRevision(next);
    } catch (cause) {
      form.setErrorMap({ onSubmit: typeErrorMessage(cause) });
    } finally {
      setRefreshing(false);
    }
  };
  const busy = pending || refreshing;
  return (
    <PreviewDialog
      title={definition ? `Review changes to ${name}` : `${typeId ? "Detach" : "Delete"} ${name}?`}
      onClose={() => {
        if (!busy) onClose();
      }}
      footer={
        <>
          <Button variant="secondary" disabled={busy} onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="submit"
            form={dialogId}
            variant="error-fill"
            disabled={busy || review.isFetching || !review.data || !!review.error}
          >
            {reviewActionLabel(!!definition, !!typeId, !!uncertain)}
          </Button>
        </>
      }
    >
      <form
        id={dialogId}
        className="space-y-5 text-13 text-secondary"
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit();
        }}
      >
        <PropertyRemovalImpact loading={review.isFetching} review={review.error ? undefined : review.data} />
        {changeSummary}
        <PropertyRemovalFeedback
          error={error ?? review.error}
          uncertain={!!uncertain}
          busy={busy}
          onRefresh={() => void refresh()}
        />
        {definition ? (
          <p>
            Selections that no longer belong to the updated configuration will be saved to each affected work item's
            description. Other values stay in their fields.
          </p>
        ) : typeId ? (
          <p>Values will be saved to each work item's description before the property is detached.</p>
        ) : (
          <form.Field name="permanent">
            {(field) => (
              <fieldset disabled={busy || !!uncertain} className="space-y-3">
                <legend className="sr-only">How should property values be removed?</legend>
                <label className="flex gap-2">
                  <input
                    type="radio"
                    name={`${dialogId}-preservation`}
                    checked={!field.state.value}
                    onChange={() => {
                      field.handleChange(false);
                      form.setErrorMap({ onSubmit: undefined });
                    }}
                  />
                  Save values to work item descriptions
                </label>
                <label className="flex gap-2">
                  <input
                    type="radio"
                    name={`${dialogId}-preservation`}
                    checked={field.state.value}
                    onChange={() => {
                      field.handleChange(true);
                      form.setErrorMap({ onSubmit: undefined });
                    }}
                  />
                  Remove current values without preserving
                </label>
                <p className="text-12 text-tertiary">
                  Existing history remains available. This does not erase historical versions or audit records.
                </p>
              </fieldset>
            )}
          </form.Field>
        )}
        {requiresConfirmation(permanent, definition, review.data) && (
          <Field label="Type Confirm to continue">
            {(fieldId) => (
              <form.Field name="confirmation">
                {(field) => (
                  <input
                    id={fieldId}
                    className={inputStyle}
                    value={field.state.value}
                    disabled={busy || !!uncertain}
                    onBlur={field.handleBlur}
                    onChange={(event) => field.handleChange(event.target.value)}
                    autoComplete="off"
                  />
                )}
              </form.Field>
            )}
          </Field>
        )}
      </form>
    </PreviewDialog>
  );
}

function PropertyRemovalImpact({ loading, review }: { loading: boolean; review?: PropertyRemovalReview }) {
  if (loading) return <output>Checking affected work items…</output>;
  if (!review) return null;
  return (
    <p>
      {review.items} work items, {review.types} types, and {review.projects} projects are affected. This includes
      private drafts and archived or trashed work items.
    </p>
  );
}
function isDefiniteRemovalRejection(cause: unknown) {
  if (!cause || typeof cause !== "object") return false;
  const response = "response" in cause ? cause.response : cause;
  if (!response || typeof response !== "object" || !("status" in response)) return false;
  return [400, 401, 403, 404, 409, 422, 429].includes(Number(response.status));
}

function PropertyRemovalFeedback({
  error,
  uncertain,
  busy,
  onRefresh,
}: {
  error: unknown;
  uncertain: boolean;
  busy: boolean;
  onRefresh: () => void;
}) {
  if (!error) return null;
  return (
    <div className="space-y-2">
      <p role="alert" className="text-danger-primary">
        {typeof error === "string" ? error : typeErrorMessage(error)}
      </p>
      {uncertain ? (
        <p>Retry to check the result of the submitted change.</p>
      ) : (
        <Button variant="secondary" disabled={busy} onClick={onRefresh}>
          Refresh review
        </Button>
      )}
    </div>
  );
}

function requiresConfirmation(permanent: boolean, definition?: PropertyDefinition, review?: PropertyRemovalReview) {
  return permanent || (!!definition && (review?.items ?? 0) > 0);
}
function reviewActionLabel(update: boolean, detach: boolean, uncertain: boolean) {
  if (uncertain) return update ? "Retry changes" : "Retry removal";
  if (update) return "Save changes";
  return detach ? "Detach property" : "Delete property";
}
