import { Suspense } from "react";
import InstanceDetailClient from "./client";

export default function Page() {
  return (
    <Suspense fallback={null}>
      <InstanceDetailClient />
    </Suspense>
  );
}

export const dynamicParams = false;

export async function generateStaticParams() {
  return [{ node_name: "placeholder-node", id: "placeholder-id" }];
}
