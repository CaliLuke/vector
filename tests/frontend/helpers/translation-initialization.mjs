import { createRequire } from "node:module";

const requireFromI18n = createRequire(new URL("../../../packages/i18n/package.json", import.meta.url));
const i18next = requireFromI18n("i18next");

export let i18nInstance;
export let initPromise;
let finishLoading;

export function startTranslations() {
  i18nInstance = i18next.createInstance();
  i18nInstance.use({
    type: "backend",
    init() {},
    read(_language, _namespace, callback) {
      finishLoading = () => callback(null, { greeting: "Translations ready" });
    },
  });
  initPromise = i18nInstance.init({
    lng: "en",
    fallbackLng: false,
    ns: ["translation"],
    initImmediate: false,
    react: { useSuspense: false },
  });
}

export async function finishTranslations() {
  finishLoading();
  await initPromise;
}
