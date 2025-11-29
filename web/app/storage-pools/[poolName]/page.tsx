import { Suspense } from "react";
import StoragePoolDetailPage from "./client";

export const dynamicParams = false;

export default function Page() {
  return (
    <Suspense fallback={null}>
      <StoragePoolDetailPage />
    </Suspense>
  );
}

export async function generateStaticParams() {
  return [{ poolName: "placeholder" }];
}
