import adapter from "@sveltejs/adapter-static";
import { vitePreprocess } from "@sveltejs/vite-plugin-svelte";

/** @type {import('@sveltejs/kit').Config} */
const config = {
  // Consult https://github.com/sveltejs/svelte-preprocess
  // for more information about preprocessors
  preprocess: vitePreprocess(),

  kit: {
    adapter: adapter({
      fallback: "index.html",
    }),
    // TODO: enable CSP after addressing error in Pylon and Codemirror.
    // When it is enabled, `frame-src` must allow the help-center video host
    // (walkthroughs are embedded from YouTube's privacy-enhanced domain).
    // The production nginx CSP needs the same `frame-src` entry — that is where
    // CSP is actually enforced today; this block is inert until uncommented.
    // csp: {
    //   directives: {
    //     "script-src": ["self"],
    //     "frame-src": ["https://www.youtube-nocookie.com"],
    //   },
    // },
    files: {
      assets: "../web-common/static",
    },
  },
};

export default config;
