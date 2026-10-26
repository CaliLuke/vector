/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { Dialog } from "@headlessui/react";
import { useMutation } from "@tanstack/react-query";
import { Button } from "@plane/propel/button";
import { checkEmailValidity } from "@plane/utils/string";
import { useSettingsForm } from "@/components/common/settings-form";
import { sendTestEmailOptions } from "@/queries/instance";

export function SendTestEmailModal({ handleClose }: { handleClose: () => void }) {
  const sendEmail = useMutation(sendTestEmailOptions);
  const form = useSettingsForm({
    defaultValues: { receiverEmail: "" },
    onSubmit: async ({ value }) => {
      await sendEmail.mutateAsync(value.receiverEmail);
    },
  });
  return (
    <Dialog
      open
      onClose={() => {
        if (!sendEmail.isPending) handleClose();
      }}
      className="relative z-20"
    >
      <div className="fixed inset-0 bg-backdrop" />
      <div className="fixed inset-0 z-20 overflow-y-auto">
        <div className="my-10 flex justify-center p-4 md:my-20">
          <Dialog.Panel className="relative w-full rounded-lg bg-surface-1 p-5 text-left shadow-raised-200 sm:max-w-xl">
            <Dialog.Title className="text-16 font-medium text-primary">
              {sendEmail.isSuccess ? "Email sent" : "Send test email"}
            </Dialog.Title>
            {sendEmail.isSuccess ? (
              <>
                <p className="py-6 text-13">
                  The test email was sent to {sendEmail.variables}. Check your spam folder if you cannot find it.
                </p>
                <Button type="button" variant="secondary" size="lg" onClick={handleClose}>
                  Close
                </Button>
              </>
            ) : (
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  void form.handleSubmit().catch(() => {});
                }}
                className="space-y-5 pt-6"
              >
                <fieldset disabled={sendEmail.isPending}>
                  <form.AppField
                    name="receiverEmail"
                    validators={{
                      onChange: ({ value }) => (checkEmailValidity(value) ? undefined : "Enter a valid email address."),
                    }}
                  >
                    {(field) => (
                      <field.TextField type="email" label="Receiver email" placeholder="you@example.com" required />
                    )}
                  </form.AppField>
                </fieldset>
                {sendEmail.isError && (
                  <p role="alert" className="text-13 text-danger-primary">
                    Failed to send the test email. Check your SMTP settings and try again.
                  </p>
                )}
                <div className="flex justify-end gap-2">
                  <Button
                    type="button"
                    variant="secondary"
                    size="lg"
                    onClick={handleClose}
                    disabled={sendEmail.isPending}
                  >
                    Cancel
                  </Button>
                  <form.Subscribe selector={(state) => [state.canSubmit, state.isSubmitting]}>
                    {([canSubmit, isSubmitting]) => (
                      <Button type="submit" variant="primary" size="lg" disabled={!canSubmit} loading={isSubmitting}>
                        {isSubmitting ? "Sending email" : "Send email"}
                      </Button>
                    )}
                  </form.Subscribe>
                </div>
              </form>
            )}
          </Dialog.Panel>
        </div>
      </div>
    </Dialog>
  );
}
