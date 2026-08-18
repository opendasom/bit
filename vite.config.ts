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
      },
    },
  },
});
