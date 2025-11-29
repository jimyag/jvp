"use client";

import { useRouter, usePathname } from "next/navigation";
import { useEffect } from "react";

export default function Home() {
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    // If already on a route (e.g., deep link), let the client router handle it.
    if (pathname && pathname !== "/") {
      router.replace(pathname);
      return;
    }
    router.replace("/instances");
  }, [pathname, router]);

  return null;
}
