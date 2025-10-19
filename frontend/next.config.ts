import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
  eslint: {
    ignoreDuringBuilds: true,
  },
  output: "standalone",

  images: {
    unoptimized: true,
  },
};

export default nextConfig;
