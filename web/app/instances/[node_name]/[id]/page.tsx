import InstanceDetailPage from "./client";

export const dynamicParams = false;

export default function Page() {
  return <InstanceDetailPage />;
}

export async function generateStaticParams() {
  return [{ node_name: "placeholder-node", id: "placeholder-id" }];
}
