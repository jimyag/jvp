import ConsolePage from "./client";

export const dynamicParams = false;

export default function Page() {
  return <ConsolePage />;
}

export async function generateStaticParams() {
  return [{ node_name: "placeholder-node", id: "placeholder-id" }];
}
