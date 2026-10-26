/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import isNil from "lodash-es/isNil.js";

import { Bell, BellOff } from "lucide-react";
// plane-i18n
import { EUserPermissions } from "@plane/constants";
import { useTranslation } from "@plane/i18n";
// UI
import { Button } from "@plane/propel/button";
import { TOAST_TYPE, setToast } from "@plane/propel/toast";
import { EIssueServiceType } from "@plane/types";
import { Loader } from "@plane/ui";
// hooks
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useCurrentUser } from "@/hooks/use-current-user";
import { issueSubscriptionOptions, updateIssueSubscriptionOptions } from "@/queries/issue-subscription";
import { useProjectPermissions } from "@/hooks/use-permissions";

export type TIssueSubscription = {
  workspaceSlug: string;
  projectId: string;
  issueId: string;
  serviceType?: EIssueServiceType;
};

export function IssueSubscription(props: TIssueSubscription) {
  return (
    <IssueSubscriptionContent
      key={`${props.workspaceSlug}/${props.projectId}/${props.serviceType}/${props.issueId}`}
      {...props}
    />
  );
}

const IssueSubscriptionContent = function IssueSubscriptionContent(props: TIssueSubscription) {
  const { workspaceSlug, projectId, issueId, serviceType = EIssueServiceType.ISSUES } = props;
  const { t } = useTranslation();
  const client = useQueryClient();
  const { data: user } = useCurrentUser();
  const scope = { userId: user?.id, workspaceSlug, projectId, issueId, serviceType };
  const subscription = useQuery(issueSubscriptionOptions(scope));
  const mutation = useMutation(updateIssueSubscriptionOptions(client, scope));
  const { can: canProject } = useProjectPermissions(workspaceSlug, projectId);
  const isSubscribed = subscription.data;
  const loading = mutation.isPending;
  const isEditable = canProject([EUserPermissions.ADMIN, EUserPermissions.MEMBER]);

  const handleSubscription = () => {
    mutation.mutate(!isSubscribed, {
      onSuccess: () =>
        setToast({
          type: TOAST_TYPE.SUCCESS,
          title: t("toast.success"),
          message: isSubscribed
            ? t("issue.subscription.actions.unsubscribed")
            : t("issue.subscription.actions.subscribed"),
        }),
      onError: () =>
        setToast({
          type: TOAST_TYPE.ERROR,
          title: t("toast.error"),
          message: t("common.error.message"),
        }),
    });
  };

  if (subscription.isError)
    return (
      <Button
        onClick={() => {
          void subscription.refetch();
        }}
      >
        Retry subscription
      </Button>
    );

  if (isNil(isSubscribed))
    return (
      <Loader>
        <Loader.Item width="106px" height="28px" />
      </Loader>
    );

  return (
    <div>
      <Button
        prependIcon={isSubscribed ? <BellOff /> : <Bell className="h-3 w-3" />}
        variant="secondary"
        className="hover:!bg-accent-primary/20"
        aria-label={isSubscribed ? t("common.actions.unsubscribe") : t("common.actions.subscribe")}
        onClick={handleSubscription}
        disabled={!isEditable || loading}
        size="lg"
      >
        {loading ? (
          <span>
            <span className="hidden sm:block">{t("common.loading")}</span>
          </span>
        ) : isSubscribed ? (
          <div className="hidden sm:block">{t("common.actions.unsubscribe")}</div>
        ) : (
          <div className="hidden sm:block">{t("common.actions.subscribe")}</div>
        )}
      </Button>
    </div>
  );
};
