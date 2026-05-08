import { readFile, rm, stat, writeFile } from "node:fs/promises";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

import { resolveConfig, type Plugin, type ResolvedConfig } from "vite";

const consoleRoot = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const componentsDtsPath = resolve(consoleRoot, "components.d.ts");
const viteConfigPath = resolve(consoleRoot, "vite.config.ts");

const previousComponentsDts = await readFile(componentsDtsPath, "utf8").catch(() => undefined);

function getConfigResolvedHook(plugin: Plugin) {
  if (!plugin.configResolved) {
    return undefined;
  }

  return typeof plugin.configResolved === "function"
    ? plugin.configResolved
    : plugin.configResolved.handler;
}

function findComponentsPlugin(plugins: readonly Plugin[]) {
  const componentsPlugin = plugins.find((plugin) => plugin.name === "unplugin-vue-components");

  if (!componentsPlugin) {
    throw new Error("Could not find unplugin-vue-components in the resolved Vite plugins.");
  }

  return componentsPlugin;
}

async function waitForFile(path: string) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    const fileStat = await stat(path).catch(() => undefined);

    if (fileStat) {
      return;
    }

    await new Promise((resolveTimeout) => {
      setTimeout(resolveTimeout, 100);
    });
  }

  await stat(path);
}

try {
  await rm(componentsDtsPath, { force: true });
  const config = await resolveConfig({ configFile: viteConfigPath, root: consoleRoot }, "serve", "development");
  const componentsPlugin = findComponentsPlugin(config.plugins);
  const configResolved = getConfigResolvedHook(componentsPlugin);

  await configResolved?.call({} as ThisParameterType<typeof configResolved>, config as ResolvedConfig);

  await waitForFile(componentsDtsPath);
} catch (error) {
  if (previousComponentsDts !== undefined) {
    await writeFile(componentsDtsPath, previousComponentsDts);
  }

  throw error;
}
