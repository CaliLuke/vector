/**
 * Copyright (c) 2023-present Plane Software, Inc. and contributors
 * SPDX-License-Identifier: AGPL-3.0-only
 * See the LICENSE file for details.
 */

import { useMutation, useQueryClient } from "@tanstack/react-query";
import { updateConfigurationsOptions } from "@/queries/instance";
import { Lightbulb } from "lucide-react";
import type { IFormattedInstanceConfiguration, TInstanceAIConfigurationKeys } from "@plane/types";
// components
import type { SettingsInput } from "@/components/common/settings-form";
import { requiredField, SettingsForm } from "@/components/common/settings-form";
import { useConfigurationForm } from "@/components/common/use-configuration-form";
// hooks

type IInstanceAIForm = {
  config: Partial<IFormattedInstanceConfiguration>;
};

type AIFormValues = Record<TInstanceAIConfigurationKeys, string>;

export function InstanceAIForm(props: IInstanceAIForm) {
  const { config } = props;
  const queryClient = useQueryClient();
  const { mutateAsync: updateInstanceConfigurations } = useMutation(updateConfigurationsOptions(queryClient));
  const form = useConfigurationForm({
    defaultValues: { LLM_API_KEY: config.LLM_API_KEY ?? "", LLM_MODEL: config.LLM_MODEL ?? "" },
    save: updateInstanceConfigurations,
    successMessage: "Settings updated successfully.",
  });

  const aiFormFields: (SettingsInput & { key: keyof AIFormValues })[] = [
    {
      key: "LLM_MODEL",
      type: "text",
      label: "LLM Model",
      description: (
        <>
          Choose an OpenAI engine.{" "}
          <a
            href="https://platform.openai.com/docs/models/overview"
            target="_blank"
            className="text-accent-primary hover:underline"
            rel="noreferrer"
            aria-label="OpenAI models documentation"
          >
            Learn more
          </a>
        </>
      ),
      placeholder: "gpt-4o-mini",
      required: false,
    },
    {
      key: "LLM_API_KEY",
      type: "password",
      label: "API key",
      description: (
        <>
          You will find your API key{" "}
          <a
            href="https://platform.openai.com/api-keys"
            target="_blank"
            className="text-accent-primary hover:underline"
            rel="noreferrer"
            aria-label="OpenAI API keys page"
          >
            here.
          </a>
        </>
      ),
      placeholder: "sk-asddassdfasdefqsdfasd23das3dasdcasd",
      required: false,
    },
  ];

  return (
    <SettingsForm onSubmit={() => form.handleSubmit()} className="space-y-8">
      <div className="space-y-3">
        <div>
          <div className="pb-1 text-18 font-medium text-primary">OpenAI</div>
          <div className="text-13 font-regular text-tertiary">If you use ChatGPT, this is for you.</div>
        </div>
        <div className="grid-col grid w-full grid-cols-1 items-center justify-between gap-x-12 gap-y-8 lg:grid-cols-3">
          {aiFormFields.map(({ key, ...input }) => (
            <form.AppField
              key={key}
              name={key}
              validators={{ onChange: input.required ? requiredField(input.label) : undefined }}
            >
              {(field) => <field.TextField {...input} />}
            </form.AppField>
          ))}
        </div>
      </div>

      <div className="flex flex-col items-start gap-4">
        <form.AppForm>
          <form.SaveButton />
        </form.AppForm>

        <div className="relative inline-flex items-center gap-1.5 rounded-sm border border-accent-subtle bg-accent-subtle px-4 py-2 text-caption-sm-regular text-accent-secondary">
          <Lightbulb className="size-4" />
          <div>
            If you have a preferred AI models vendor, please get in{" "}
            <a className="font-medium underline" href="https://github.com/CaliLuke/vector">
              touch with us.
            </a>
          </div>
        </div>
      </div>
    </SettingsForm>
  );
}
