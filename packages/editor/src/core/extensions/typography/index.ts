import type { InputRule } from "@tiptap/core";
import { Extension } from "@tiptap/core";
// constants
import { CORE_EXTENSIONS } from "@/constants/extension";
import type { TypographyOptions } from "./inputRules";
import {
  emDash,
  ellipsis,
  leftArrow,
  rightArrow,
  copyright,
  trademark,
  servicemark,
  registeredTrademark,
  oneHalf,
  plusMinus,
  notEqual,
  laquo,
  raquo,
  multiplication,
  superscriptTwo,
  superscriptThree,
  oneQuarter,
  threeQuarters,
  impliesArrowRight,
} from "./inputRules";

const inputRules: [keyof TypographyOptions, (override?: string) => InputRule][] = [
  ["emDash", emDash],
  ["impliesArrowRight", impliesArrowRight],
  ["ellipsis", ellipsis],
  ["leftArrow", leftArrow],
  ["rightArrow", rightArrow],
  ["copyright", copyright],
  ["trademark", trademark],
  ["servicemark", servicemark],
  ["registeredTrademark", registeredTrademark],
  ["oneHalf", oneHalf],
  ["plusMinus", plusMinus],
  ["notEqual", notEqual],
  ["laquo", laquo],
  ["raquo", raquo],
  ["multiplication", multiplication],
  ["superscriptTwo", superscriptTwo],
  ["superscriptThree", superscriptThree],
  ["oneQuarter", oneQuarter],
  ["threeQuarters", threeQuarters],
];

export const CustomTypographyExtension = Extension.create<TypographyOptions>({
  name: CORE_EXTENSIONS.TYPOGRAPHY,
  addInputRules() {
    return inputRules.flatMap(([name, createRule]) => {
      const replacement = this.options[name];
      return replacement === false ? [] : [createRule(replacement)];
    });
  },
});
