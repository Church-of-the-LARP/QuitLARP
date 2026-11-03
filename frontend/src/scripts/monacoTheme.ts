import type { Monaco } from "@monaco-editor/react";

export const APP_THEME = "quitlarp";

// monaco only accepts hex; this is mist-800 from index.css
const BACKGROUND = "#22292b";

export function defineAppTheme(monaco: Monaco) {
  monaco.editor.defineTheme(APP_THEME, {
    base: "vs-dark",
    inherit: true,
    rules: [],
    colors: {
      "editor.background": BACKGROUND,
      "editorGutter.background": BACKGROUND,
    },
  });
}
