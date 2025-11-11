import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://192.168.2.100:8080/api/:path*',
      },
    ];
  },
};

export default nextConfig;
