import path from "node:path";
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: process.env.TAURI_ENV ? "export" : undefined,
  transpilePackages: ["@lifebase/features-todo", "@lifebase/design-tokens"],
  turbopack: {
    root: path.join(__dirname, "../.."),
  },
  images: {
    remotePatterns: [
      {
        protocol: "http",
        hostname: "localhost",
        port: "38117",
        pathname: "/api/v1/gallery/thumbnails/**",
      },
      {
        protocol: "https",
        hostname: "api.lifebase.cc",
        pathname: "/api/v1/gallery/thumbnails/**",
      },
    ],
  },
};

export default nextConfig;
