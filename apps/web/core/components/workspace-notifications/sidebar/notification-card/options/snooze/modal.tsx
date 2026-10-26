/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useState } from "react";
import { useForm, useStore } from "@tanstack/react-form";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { snoozeDate, type SnoozeTime } from "./time";
// plane imports
import { allTimeIn30MinutesInterval12HoursFormat } from "@plane/constants";
import { Button } from "@plane/propel/button";
import { CloseIcon } from "@plane/propel/icons";
import { CustomSelect, EModalPosition, EModalWidth, ModalCore } from "@plane/ui";
// components
import { getDate } from "@plane/utils/datetime";
import { cn } from "@plane/utils/common";
import { DateDropdown } from "@/components/dropdowns/date";

type TNotificationSnoozeModal = {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (dateTime?: Date) => Promise<void>;
};

const defaultValues: SnoozeTime = { date: undefined, time: undefined, period: "AM" };

const timeStamps = allTimeIn30MinutesInterval12HoursFormat;

export function NotificationSnoozeModal(props: TNotificationSnoozeModal) {
  if (!props.isOpen) return null;
  return <SnoozeForm {...props} />;
}

function SnoozeForm({ onClose, onSubmit }: TNotificationSnoozeModal) {
  const form = useForm({
    defaultValues,
    validators: {
      onSubmit: ({ value }) => {
        const date = snoozeDate(value);
        return date && date > new Date() ? undefined : "Choose a future date and time.";
      },
    },
    onSubmit: async ({ value }) => {
      try {
        await onSubmit(snoozeDate(value));
        onClose();
      } catch (cause) {
        setToast({ type: TOAST_TYPE.ERROR, title: "Could not snooze notification", message: "Please try again." });
        throw cause;
      }
    },
  });
  const values = useStore(form.store, (state) => state.values);
  const isSubmitting = useStore(form.store, (state) => state.isSubmitting);
  const errors = useStore(form.store, (state) => state.errors);
  const handleClose = () => {
    if (!isSubmitting) onClose();
  };
  const [openedAt] = useState(() => new Date());
  const availableTimes = timeStamps.filter(({ value }) => {
    if (!values.date) return true;
    const date = snoozeDate({ ...values, time: value });
    return date && date > openedAt;
  });

  return (
    <ModalCore isOpen handleClose={handleClose} position={EModalPosition.CENTER} width={EModalWidth.XXL}>
      <form
        onSubmit={(event) => {
          event.preventDefault();
          form.setErrorMap({ onSubmit: undefined });
          void form.handleSubmit().catch(() => {});
        }}
        className="p-5"
      >
        <div className="flex items-center justify-between">
          <h3 className="text-h5-medium leading-6 text-primary">Customize Snooze Time</h3>

          <div>
            <button type="button" onClick={handleClose}>
              <CloseIcon className="h-5 w-5 text-primary" />
            </button>
          </div>
        </div>

        <div className="mt-5 flex flex-col gap-3 md:!flex-row md:items-center">
          <div className="flex-1 pb-3 md:pb-0">
            <h6 className="mb-2 block text-body-xs-medium text-placeholder">Pick a date</h6>
            <form.Field name="date">
              {(field) => (
                <DateDropdown
                  value={field.state.value ?? null}
                  placeholder="Select date"
                  onChange={(value) => {
                    form.setFieldValue("time", undefined);
                    field.handleChange(value ? (getDate(value) ?? undefined) : undefined);
                  }}
                  minDate={openedAt}
                  disabled={isSubmitting}
                  buttonVariant="border-with-text"
                  buttonContainerClassName="w-full text-left"
                  buttonClassName="border-strong px-3 py-2.5"
                  hideIcon
                />
              )}
            </form.Field>
          </div>
          <div className="flex-1">
            <h6 className="mb-2 block text-body-xs-medium text-placeholder">Pick a time</h6>
            <form.Field name="time">
              {(field) => (
                <CustomSelect
                  value={field.state.value}
                  disabled={isSubmitting}
                  onChange={field.handleChange}
                  label={
                    <div className="truncate">
                      {field.state.value ? (
                        <span>
                          {field.state.value} {values.period.toLowerCase()}
                        </span>
                      ) : (
                        <span className="text-body-xs-medium text-placeholder">Select a time</span>
                      )}
                    </div>
                  }
                  input
                >
                  <div className="mb-2 flex h-9 w-full overflow-hidden rounded-xs">
                    <button
                      type="button"
                      onClick={() => {
                        form.setFieldValue("period", "AM");
                        form.setFieldValue("time", undefined);
                      }}
                      className={cn("flex h-full w-1/2 cursor-pointer items-center justify-center text-center", {
                        "bg-accent-primary/90 text-on-color": values.period === "AM",
                        "bg-layer-1": values.period !== "AM",
                      })}
                    >
                      AM
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        form.setFieldValue("period", "PM");
                        form.setFieldValue("time", undefined);
                      }}
                      className={cn("flex h-full w-1/2 cursor-pointer items-center justify-center text-center", {
                        "bg-accent-primary/90 text-on-color": values.period === "PM",
                        "bg-layer-1": values.period !== "PM",
                      })}
                    >
                      PM
                    </button>
                  </div>
                  {availableTimes.length > 0 ? (
                    availableTimes.map((time) => (
                      <CustomSelect.Option key={time.value} value={time.value}>
                        <div className="flex items-center">
                          <span className="ml-3 block truncate">{time.label}</span>
                        </div>
                      </CustomSelect.Option>
                    ))
                  ) : (
                    <p className="p-3 text-center text-secondary">No available time for this date.</p>
                  )}
                </CustomSelect>
              )}
            </form.Field>
          </div>
        </div>

        {errors.length > 0 && (
          <p role="alert" className="mt-3 text-13 text-danger-primary">
            {errors.join(" ")}
          </p>
        )}
        <div className="mt-5 flex items-center justify-between gap-2">
          <div className="flex w-full items-center justify-end gap-2">
            <Button variant="secondary" type="button" disabled={isSubmitting} size="lg" onClick={handleClose}>
              Cancel
            </Button>
            <Button variant="primary" size="lg" type="submit" loading={isSubmitting}>
              {isSubmitting ? "Submitting..." : "Submit"}
            </Button>
          </div>
        </div>
      </form>
    </ModalCore>
  );
}
