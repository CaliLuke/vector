/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */
import { useEffect, useRef, useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { ESTIMATE_SYSTEMS, estimateCount, MAX_ESTIMATE_POINT_INPUT_LENGTH } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
import { Button } from "@plane/propel/button";
import { ChevronLeftIcon, TrashIcon } from "@plane/propel/icons";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EEstimateSystem, type TEstimateSystemKeys } from "@plane/types";
import { EModalPosition, EModalWidth, Input, ModalCore, Sortable } from "@plane/ui";
import { isEstimatePointValuesRepeated } from "@plane/utils/estimates";
import { useEstimateEditorKey, useEstimateMutations } from "@/hooks/use-estimates";
import { EstimateCreateStageOne } from "./stage-one";

type Props = { workspaceSlug: string; projectId: string; isOpen: boolean; handleClose: () => void };
type EstimateDraft = { system: TEstimateSystemKeys; points: { clientId: string; value: string }[] };
function pointError(value: string, otherValues: string[], system: TEstimateSystemKeys): string | undefined {
  if (!value.trim()) return "project_settings.estimates.validation.empty";
  if (value.length > MAX_ESTIMATE_POINT_INPUT_LENGTH) return "Estimate value is too long.";
  if (isEstimatePointValuesRepeated(otherValues, system, value))
    return "project_settings.estimates.validation.already_exists";
  if (system === EEstimateSystem.CATEGORIES)
    return Number.isNaN(Number(value)) ? undefined : "project_settings.estimates.validation.character";
  if (!Number.isFinite(Number(value))) return "project_settings.estimates.validation.numeric";
  if (Number(value) <= 0) return "project_settings.estimates.validation.min_length";
  return undefined;
}
function validatePoints(draft: EstimateDraft) {
  const fields: Partial<Record<`points[${number}].value`, string>> = {};
  draft.points.forEach((point, index) => {
    const error = pointError(
      point.value,
      draft.points.filter((_item, i) => i !== index).map((item) => item.value),
      draft.system
    );
    if (error) fields[`points[${index}].value`] = error;
  });
  return { fields };
}
export function CreateEstimateModal(props: Props) {
  const key = useEstimateEditorKey(props.workspaceSlug, props.projectId);
  return props.isOpen ? <EstimateForm key={key} {...props} /> : null;
}
function EstimateForm({ workspaceSlug, projectId, handleClose }: Props) {
  const { mutation } = useEstimateMutations(workspaceSlug, projectId);
  const mounted = useRef(false);
  useEffect(() => {
    mounted.current = true;
    return () => {
      mounted.current = false;
    };
  }, []);
  const { t } = useTranslation();
  const [step, setStep] = useState<1 | 2>(1);
  const initial: EstimateDraft = { system: EEstimateSystem.POINTS, points: [] };
  const form = useForm({
    defaultValues: initial,
    validators: {
      onChange: ({ value }) => validatePoints(value),
      onSubmit: ({ value }): string | undefined =>
        value.points.length < estimateCount.min || value.points.length > estimateCount.max
          ? `Use between ${estimateCount.min} and ${estimateCount.max} estimate values.`
          : undefined,
    },
    onSubmit: async ({ value }) => {
      const result = await mutation.mutateAsync({
        type: "create",
        data: {
          estimate: { name: ESTIMATE_SYSTEMS[value.system].name, type: value.system, last_used: true },
          estimate_points: value.points.map((point, index) => ({
            key: index + 1,
            value: point.value.trim(),
            id: undefined,
          })),
        },
      });
      if (!mounted.current || !result.captured.isCurrent()) return;
      setToast({
        type: TOAST_TYPE.SUCCESS,
        title: t("project_settings.estimates.toasts.created.success.title"),
        message: t("project_settings.estimates.toasts.created.success.message"),
      });
      handleClose();
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const busy = useStore(form.store, (state) => state.isSubmitting);
  const validationError = useStore(form.store, (state) => state.errorMap.onSubmit);
  const close = () => {
    if (!busy) handleClose();
  };
  return (
    <ModalCore isOpen handleClose={close} position={EModalPosition.TOP} width={EModalWidth.XXL}>
      <form
        noValidate
        onSubmit={(event) => {
          event.preventDefault();
          if (step === 2) {
            form.setErrorMap({ onSubmit: undefined });
            void form.handleSubmit().catch(() => {
              if (!mounted.current) return;
              setToast({
                type: TOAST_TYPE.ERROR,
                title: t("project_settings.estimates.toasts.created.error.title"),
                message: t("project_settings.estimates.toasts.created.error.message"),
              });
              form.setErrorMap({
                onSubmit: { form: t("project_settings.estimates.toasts.created.error.message"), fields: {} },
              });
            });
          }
        }}
      >
        <fieldset disabled={busy} inert={busy} className="relative space-y-6 py-5">
          <div className="flex items-center justify-between gap-2 px-5">
            <div className="flex items-center gap-1">
              {step === 2 && (
                <button
                  type="button"
                  aria-label="Back to estimate systems"
                  onClick={() => setStep(1)}
                  className="grid size-5 place-items-center"
                >
                  <ChevronLeftIcon className="size-4" />
                </button>
              )}
              <h3 className="text-18 font-medium text-primary">{t("project_settings.estimates.new")}</h3>
            </div>
            <div className="text-11 text-secondary">
              {t("project_settings.estimates.create.step", { step, total: 2 })}
            </div>
          </div>
          <div className="px-5">
            {step === 1 ? (
              <EstimateCreateStageOne
                estimateSystem={values.system}
                handleEstimateSystem={(system) => form.setFieldValue("system", system)}
                handleEstimatePoints={(template) => {
                  form.setFieldValue(
                    "points",
                    ESTIMATE_SYSTEMS[values.system].templates[template].values.map((point) => ({
                      clientId: crypto.randomUUID(),
                      value: point.value,
                    }))
                  );
                  setStep(2);
                }}
              />
            ) : (
              <form.Field name="points" mode="array">
                {() => (
                  <div className="space-y-2">
                    <Sortable
                      data={values.points}
                      keyExtractor={(point) => point.clientId}
                      onChange={(points) => form.setFieldValue("points", points)}
                      render={(point, index) => (
                        <form.Field key={point.clientId} name={`points[${index}].value`}>
                          {(field) => (
                            <div className="my-1 space-y-1">
                              <div className="flex items-center gap-2 rounded-sm border border-subtle p-1">
                                <span className="px-1 text-11 text-secondary" aria-label={`Position ${index + 1}`}>
                                  {index + 1}
                                </span>
                                <Input
                                  name={field.name}
                                  aria-label={`Estimate value ${index + 1}`}
                                  value={field.state.value}
                                  onChange={(event) => field.handleChange(event.target.value)}
                                  onBlur={field.handleBlur}
                                  hasError={!field.state.meta.isValid}
                                  className="w-full"
                                />
                                <button
                                  type="button"
                                  aria-label={`Remove estimate ${index + 1}`}
                                  disabled={values.points.length <= estimateCount.min}
                                  onClick={() => void form.removeFieldValue("points", index)}
                                  className="grid size-7 shrink-0 place-items-center"
                                >
                                  <TrashIcon className="size-4" />
                                </button>
                              </div>
                              {field.state.meta.errors.map((message) => (
                                <p key={message} role="alert" className="text-11 text-danger-primary">
                                  {message ? t(message) : null}
                                </p>
                              ))}
                            </div>
                          )}
                        </form.Field>
                      )}
                    />
                    <Button
                      type="button"
                      variant="link"
                      disabled={values.points.length >= estimateCount.max}
                      onClick={() => form.pushFieldValue("points", { clientId: crypto.randomUUID(), value: "" })}
                    >
                      Add {values.system}
                    </Button>
                  </div>
                )}
              </form.Field>
            )}
            {validationError && (
              <p role="alert" className="mt-3 text-13 text-danger-primary">
                {t(validationError)}
              </p>
            )}
          </div>
          <div className="flex items-center justify-end gap-3 border-t border-subtle px-5 pt-5">
            <Button type="button" variant="secondary" size="lg" onClick={close} disabled={busy}>
              {t("common.cancel")}
            </Button>
            {step === 2 && (
              <Button type="submit" variant="primary" size="lg" loading={busy} disabled={busy}>
                {busy ? t("common.creating") : t("project_settings.estimates.create.label")}
              </Button>
            )}
          </div>
        </fieldset>
      </form>
    </ModalCore>
  );
}
