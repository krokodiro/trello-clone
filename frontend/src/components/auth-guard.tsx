"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { PageLoader } from "./ui";

export function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !user) {
      router.replace("/login");
      return;
    }
    if (!loading && user && !user.email_verified_at) {
      router.replace("/verify-email");
    }
  }, [user, loading, router]);

  if (loading || !user || !user.email_verified_at) {
    return <PageLoader label="Loading..." />;
  }

  return <>{children}</>;
}
