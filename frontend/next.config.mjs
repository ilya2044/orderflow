/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",
  images: {
    remotePatterns: [
      { protocol: "http", hostname: "localhost" },
      { protocol: "http", hostname: "minio" },
    ],
  },
  async rewrites() {
    const apiGateway = process.env.API_GATEWAY_URL || "http://localhost:8080";
    // In Docker the internal MinIO host is "minio", locally it is "localhost"
    const minioInternal = process.env.MINIO_INTERNAL_URL || "http://localhost:9000";
    return [
      {
        source: "/api/:path*",
        destination: `${apiGateway}/api/:path*`,
      },
      {
        source: "/storage/:path*",
        destination: `${minioInternal}/:path*`,
      },
    ];
  },
};

export default nextConfig;
