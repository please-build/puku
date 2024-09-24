import React from "react";

import { bar } from "./bar";

// Test an async import
export const routes = {
  "/some/path": {
    loader() {
      const { lazyLoadedFn } = await import("./lazy_loaded");
      // dynamic import paths should not work
      await import(`./lazy_${1 + 1}_loaded`);
    },
  },
};
