/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useIsMutating, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { disableEmailOptions, instanceConfigurationsOptions, instanceKeys } from "@/queries/instance";
import { useState } from "react";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { Button } from "@plane/propel/button";
import { Loader, ToggleSwitch } from "@plane/ui";
// components
import { PageWrapper } from "@/components/common/page-wrapper";
// hooks
// types
import type { Route } from "./+types/page";
// local
import { InstanceEmailForm } from "./email-config-form";

const InstanceEmailPage = function InstanceEmailPage(_props: Route.ComponentProps) {
  const { data: formattedConfig, isLoading, isError, refetch } = useQuery(instanceConfigurationsOptions);
  const queryClient = useQueryClient();
  const { mutateAsync: disableEmail } = useMutation(disableEmailOptions(queryClient));

  const isSubmitting = useIsMutating({ mutationKey: instanceKeys.configurations }) > 0;

  const [isConfiguringSMTP, setIsConfiguringSMTP] = useState(false);
  const isSMTPEnabled = isConfiguringSMTP || formattedConfig?.ENABLE_SMTP === "1";

  const handleToggle = async () => {
    if (isSMTPEnabled) {
      if (formattedConfig?.ENABLE_SMTP !== "1") {
        setIsConfiguringSMTP(false);
        return;
      }
      try {
        await disableEmail();
        setIsConfiguringSMTP(false);
        setToast({
          title: "Email feature disabled",
          message: "Email feature has been disabled",
          type: TOAST_TYPE.SUCCESS,
        });
      } catch (_error) {
        setToast({
          title: "Error disabling email",
          message: "Failed to disable email feature. Please try again.",
          type: TOAST_TYPE.ERROR,
        });
      }
      return;
    }
    setIsConfiguringSMTP(true);
  };

  return (
    <PageWrapper
      header={{
        title: "Secure emails from your own instance",
        description: (
          <>
            Vector can send useful emails to you and your users from your own instance without talking to the Internet.
            <div className="text-13 font-regular text-tertiary">
              Save your settings, then send a test email to verify them.&nbsp;
              <span className="text-danger-primary">Misconfigs can lead to email bounces and errors.</span>
            </div>
          </>
        ),
        actions: isLoading ? (
          <Loader>
            <Loader.Item width="24px" height="16px" className="rounded-full" />
          </Loader>
        ) : (
          <ToggleSwitch
            value={isSMTPEnabled}
            onChange={handleToggle}
            size="sm"
            disabled={isSubmitting || isError}
            label="Enable SMTP"
          />
        ),
      }}
    >
      {isError && (
        <div role="alert" className="space-y-3">
          <p>Unable to load email settings.</p>
          <Button variant="secondary" onClick={() => void refetch()}>
            Retry
          </Button>
        </div>
      )}
      {isSMTPEnabled && formattedConfig && !isLoading && <InstanceEmailForm config={formattedConfig} />}
    </PageWrapper>
  );
};

export const meta: Route.MetaFunction = () => [{ title: "Email Settings - God Mode" }];

export default InstanceEmailPage;
