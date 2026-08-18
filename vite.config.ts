import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          react: ["react", "react-dom"],
          web3: ["viem"],
        },
      },
    },
  },
  server: {
    proxy: {
      "/ipfs-api": {
        target: "http://127.0.0.1:5001",
        changeOrigin: true,
        rewrite: (path) => path.replace(/^\/ipfs-api/, ""),
        configure: (proxy) => {
          proxy.on("proxyReq", (proxyRequest) => {
            // Kubo rejects browser requests unless users modify their
            // node-wide CORS config. The Vite proxy is same-origin and
            // local-only, so do not forward browser-identifying headers.
            proxyRequest.removeHeader("origin");
            proxyRequest.removeHeader("referer");
            proxyRequest.removeHeader("user-agent");
          });
        },
      },
    },
  },
});
