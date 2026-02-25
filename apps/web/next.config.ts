import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: process.env.TAURI_ENV ? "export" : undefined,
};

export default nextConfig;
