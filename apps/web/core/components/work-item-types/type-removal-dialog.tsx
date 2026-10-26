import { useId, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { Form } from "@plane/ui";
import type { WorkItemType } from "@/services/work-item-types.service";
import { typeErrorMessage } from "@/services/work-item-types.service";
import type { ReviewedTypeRemoval, TypeRemovalReview } from "@/services/work-item-type-removal.service";
import { Field, inputStyle, PreviewDialog } from "./controls";
import { canReviewTypeRemovalAgain, useTypeRemoval } from "./use-type-removal";

type Props = {
  workspaceSlug: string;
  type: Pick<WorkItemType, "id" | "name">;
  types: WorkItemType[];
  revision: number;
  refreshRevision: () => Promise<number>;
  onClose: () => void;
};
export function TypeRemovalDialog({ workspaceSlug, type, types, revision, refreshRevision, onClose }: Props) {
  const formId = useId();
  const removal = useTypeRemoval(workspaceSlug, type.id);
  const [initialReplacement] = useState(() => removal.rejected?.command.replacement_id ?? "");
  const [reviewed, setReviewed] = useState<ReviewedTypeRemoval>();
  const [reviewRevision, setReviewRevision] = useState(revision);
  const form = useForm({
    defaultValues: { replacement: initialReplacement },
    validators: {
      onSubmit: ({ value }): string | undefined =>
        !value.replacement && !removal.retained ? "Choose an active replacement type." : undefined,
    },
    onSubmit: async ({ value, formApi }) => {
      formApi.setErrorMap({ onSubmit: undefined });
      try {
        const attempt = removal.retained ?? reviewed;
        if (attempt?.impact.id) {
          await removal.commit(attempt);
          removal.acknowledge();
          onClose();
        } else {
          const command = { revision: reviewRevision, type_id: type.id, replacement_id: value.replacement };
          const impact = await removal.preview(command);
          setReviewed({ command: { ...command, review_id: impact.id }, impact });
        }
      } catch (cause) {
        if (!removal.isCurrent()) return;
        if (canReviewTypeRemovalAgain(cause)) setReviewed(undefined);
        formApi.setErrorMap({ onSubmit: typeErrorMessage(cause) });
      }
    },
  });
  const refreshMutation = useMutation({ mutationFn: refreshRevision, retry: false });
  const pending = useStore(form.store, (state) => state.isSubmitting);
  const error = useStore(form.store, (state) => state.errorMap.onSubmit);
  const replacement = useStore(form.store, (state) => state.values.replacement);
  const impact = (removal.retained ?? reviewed)?.impact;
  const busy = pending || removal.busy || refreshMutation.isPending;
  const choices = types.filter((candidate) => candidate.id !== type.id && candidate.active);
  const refresh = async () => {
    form.setErrorMap({ onSubmit: undefined });
    try {
      const next = await refreshMutation.mutateAsync();
      if (!removal.isCurrent()) return;
      setReviewRevision(next);
      setReviewed(undefined);
    } catch (cause) {
      if (removal.isCurrent()) form.setErrorMap({ onSubmit: typeErrorMessage(cause) });
    }
  };
  return (
    <PreviewDialog
      title={`Delete ${type.name}?`}
      onClose={() => {
        if (!busy) onClose();
      }}
      footer={
        <TypeRemovalFooter
          formId={formId}
          busy={busy}
          retained={!!removal.retained}
          replacement={replacement}
          impact={impact}
          onClose={onClose}
        />
      }
    >
      <Form id={formId} onSubmit={() => form.handleSubmit()} className="space-y-4 text-13 text-secondary">
        <p>
          Choose the type for its work items and private drafts. This also includes archived and trashed work items.
        </p>
        <Field label="Replacement type">
          {(id) => (
            <select
              id={id}
              className={inputStyle}
              disabled={busy || !!removal.retained}
              value={removal.retained?.command.replacement_id ?? replacement}
              onChange={(event) => {
                form.setFieldValue("replacement", event.target.value);
                form.setErrorMap({ onSubmit: undefined });
                setReviewed(undefined);
              }}
            >
              <option value="">Choose a type</option>
              {choices.map((choice) => (
                <option key={choice.id} value={choice.id}>
                  {choice.name}
                </option>
              ))}
            </select>
          )}
        </Field>
        {!choices.length && <p role="alert">Activate a replacement type before deleting this one.</p>}
        {impact && <TypeRemovalImpact review={impact} />}
        <p>
          Values shared with the replacement stay in their fields. Other values are saved to each item's description.
          This does not import or activate the replacement in projects.
        </p>
        <TypeRemovalFeedback
          error={error ?? removal.error}
          retained={!!removal.retained}
          blocked={!!impact?.blocked_reason}
          busy={busy}
          refresh={() => void refresh()}
        />
      </Form>
    </PreviewDialog>
  );
}
function TypeRemovalImpact({ review }: { review: TypeRemovalReview }) {
  return (
    <div className="space-y-2">
      <p>
        {removalCount(review.items, "work item")} and {removalCount(review.drafts, "draft")} across{" "}
        {removalCount(review.projects, "project")} will be reassigned. Values will be preserved in{" "}
        {removalCount(review.preserved, "description")}.
      </p>
      {review.blocked_reason && (
        <p role="alert" className="text-danger-primary">
          {review.blocked_reason}
        </p>
      )}
    </div>
  );
}

function TypeRemovalFooter({
  formId,
  busy,
  retained,
  replacement,
  impact,
  onClose,
}: {
  formId: string;
  busy: boolean;
  retained: boolean;
  replacement: string;
  impact?: TypeRemovalReview;
  onClose: () => void;
}) {
  return (
    <>
      <Button variant="secondary" disabled={busy} onClick={onClose}>
        Cancel
      </Button>
      <Button
        type="submit"
        form={formId}
        variant={impact?.id ? "error-fill" : "primary"}
        disabled={busy || (!retained && (!replacement || !!impact?.blocked_reason))}
      >
        {retained ? "Retry deletion" : impact?.id ? "Delete type" : "Review deletion"}
      </Button>
    </>
  );
}
function TypeRemovalFeedback({
  error,
  retained,
  blocked,
  busy,
  refresh,
}: {
  error: unknown;
  retained: boolean;
  blocked: boolean;
  busy: boolean;
  refresh: () => void;
}) {
  return (
    <>
      {!!error && (
        <p role="alert" className="text-danger-primary">
          {typeof error === "string" ? error : typeErrorMessage(error)}
        </p>
      )}
      {retained && <p>Retry to check the result of the original deletion.</p>}
      {!retained && (!!error || blocked) && (
        <Button variant="secondary" disabled={busy} onClick={refresh}>
          Refresh review
        </Button>
      )}
    </>
  );
}

function removalCount(count: number, label: string) {
  return `${count} ${label}${count === 1 ? "" : "s"}`;
}
