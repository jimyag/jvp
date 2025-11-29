import { Suspense } from "react";
import NodeDetailPage from "./client";

export const dynamicParams = false;

export default function Page() {
  return (
    <Suspense fallback={null}>
      <NodeDetailPage />
    </Suspense>
  );
}

export async function generateStaticParams() {
  return [{ name: "placeholder-node" }];
}
