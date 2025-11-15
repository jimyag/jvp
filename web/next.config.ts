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
  webpack: (config) => {
    // 配置 noVNC 作为外部模块处理
    config.resolve.alias = {
      ...config.resolve.alias,
    };

    // 添加对 .js 文件的 ES Module 支持
    config.module.rules.push({
      test: /\.m?js$/,
      type: 'javascript/auto',
      resolve: {
        fullySpecified: false,
      },
    });

    return config;
  },
  transpilePackages: ['@novnc/novnc'],
};

export default nextConfig;
