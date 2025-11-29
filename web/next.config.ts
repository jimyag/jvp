import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: 'export',
  images: {
    unoptimized: true,
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
