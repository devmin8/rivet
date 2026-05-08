import { globalIgnores } from "eslint/config";
import { defineConfigWithVueTs, vueTsConfigs } from "@vue/eslint-config-typescript";
import pluginVue from "eslint-plugin-vue";
import pluginOxlint from "eslint-plugin-oxlint";
import skipFormatting from "eslint-config-prettier/flat";

// To allow more languages other than `ts` in `.vue` files, uncomment the following lines:
// import { configureVueProject } from '@vue/eslint-config-typescript'
// configureVueProject({ scriptLangs: ['ts', 'tsx'] })
// More info at https://github.com/vuejs/eslint-config-typescript/#advanced-setup

export default defineConfigWithVueTs(
  {
    name: "app/files-to-lint",
    files: ["**/*.{vue,ts,mts,tsx}"],
  },

  globalIgnores(["**/dist/**", "**/dist-ssr/**", "**/coverage/**"]),

  ...pluginVue.configs["flat/essential"],
  vueTsConfigs.recommended,

  ...pluginOxlint.buildFromOxlintConfigFile(".oxlintrc.json"),

  {
    name: "app/vue-rules",
    rules: {
      "vue/multi-word-component-names": "off",
    },
  },

  {
    name: "app/no-global-component-imports",
    files: ["src/**/*.{vue,ts,mts,tsx}"],
    ignores: ["src/components/**/*"],
    rules: {
      "no-restricted-imports": [
        "error",
        {
          patterns: [
            {
              group: ["~/components", "~/components/**"],
              message:
                "Do not import from ~/components outside src/components; unplugin-vue-components auto-imports those components when you use their PascalCase tags.",
            },
          ],
        },
      ],
    },
  },

  skipFormatting,
);
