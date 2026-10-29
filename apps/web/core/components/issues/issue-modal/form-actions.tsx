import type { RefObject, ReactNode } from "react";
import { Button } from "@plane/propel/button";
import { ToggleSwitch } from "@plane/ui/button";
import { useTranslation } from "@plane/i18n";
import type { IssueDraftForm } from "./form-state";

type Props = {
  form: IssueDraftForm;
  isNew: boolean;
  submitError: ReactNode;
  showActionButtons: boolean;
  isCreateMoreToggleEnabled: boolean;
  onCreateMoreToggleChange: (value: boolean) => void;
  close: () => void;
  discardTabIndex: number | undefined;
  submitTabIndex: number | undefined;
  moveToIssue: boolean;
  publishing: boolean;
  setPublishing: (value: boolean) => void;
  submitBtnRef: RefObject<HTMLButtonElement | null>;
  isSubmitting: boolean;
  primaryButtonText: { default: string; loading: string };
};
export function IssueFormActions(props: Props) {
  const {
    form,
    isNew,
    submitError,
    showActionButtons,
    isCreateMoreToggleEnabled,
    onCreateMoreToggleChange,
    close,
    discardTabIndex,
    submitTabIndex,
    moveToIssue,
    publishing,
    setPublishing,
    submitBtnRef,
    isSubmitting,
    primaryButtonText,
  } = props;
  const { t } = useTranslation();
  return (
    <div className="rounded-b-lg bg-surface-1 px-4 pb-3">
      {submitError && (
        <p role="alert" className="text-13 text-danger-primary">
          {submitError}
        </p>
      )}
      {showActionButtons && (
        <div className="flex items-center justify-end gap-4 border-t-[0.5px] border-subtle pt-6 pb-3">
          {isNew && (
            <label className="inline-flex cursor-pointer items-center gap-1.5">
              <ToggleSwitch value={isCreateMoreToggleEnabled} onChange={onCreateMoreToggleChange} size="sm" />
              <span className="text-caption-sm-regular">{t("create_more")}</span>
            </label>
          )}
          <Button variant="secondary" size="lg" type="button" onClick={close} tabIndex={discardTabIndex}>
            {t("discard")}
          </Button>
          <Button
            variant={moveToIssue ? "secondary" : "primary"}
            size="lg"
            type={publishing ? "button" : "submit"}
            onClick={publishing ? () => setPublishing(false) : undefined}
            ref={submitBtnRef}
            loading={isSubmitting}
            disabled={isSubmitting}
            tabIndex={submitTabIndex}
          >
            {publishing ? "Back to draft" : isSubmitting ? primaryButtonText.loading : primaryButtonText.default}
          </Button>
          {moveToIssue && (
            <Button
              variant="primary"
              type="button"
              size="lg"
              disabled={isSubmitting}
              onClick={() => {
                form.setErrorMap({ onSubmit: undefined });
                if (!publishing) setPublishing(true);
                else void form.handleSubmit({ action: "move" }).catch(() => {});
              }}
            >
              {publishing ? "Publish work item" : t("add_to_project")}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}
