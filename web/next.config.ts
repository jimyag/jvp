import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  async rewrites() {
    // 支持通过环境变量配置后端 API 地址
    // 默认使用 localhost:8080
    const apiUrl = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

    return [
      {
        source: '/api/:path*',
        destination: `${apiUrl}/api/:path*`,
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
