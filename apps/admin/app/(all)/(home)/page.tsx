/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useQuery } from "@tanstack/react-query";
import { instanceInfoOptions } from "@/queries/instance";
// components
import { LogoSpinner } from "@/components/common/logo-spinner";
import { InstanceFailureView } from "@/components/instance/failure";
import { InstanceSetupForm } from "@/components/instance/setup-form";
// hooks
// components
import type { Route } from "./+types/page";
import { InstanceSignInForm } from "./sign-in-form";

function HomePage() {
  const { data: instanceInfo, error } = useQuery(instanceInfoOptions);
  const instance = instanceInfo?.instance;

  // if instance is not fetched, show loading
  if (!instance && !error) {
    return (
      <div className="flex h-screen w-full items-center justify-center">
        <LogoSpinner />
      </div>
    );
  }

  // if instance fetch fails, show failure view
  if (error) {
    return <InstanceFailureView />;
  }

  // if instance is fetched and setup is not done, show setup form
  if (instance && !instance?.is_setup_done) {
    return <InstanceSetupForm />;
  }

  // if instance is fetched and setup is done, show sign in form
  return <InstanceSignInForm />;
}

export default HomePage;

export const meta: Route.MetaFunction = () => [
  { title: "Admin – Instance Setup & Sign-In" },
  { name: "description", content: "Configure your Vector instance or sign in to the admin portal." },
];
