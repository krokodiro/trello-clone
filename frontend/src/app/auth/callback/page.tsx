"use client";

import { Suspense, useEffect } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";

function CallbackHandler() {
  const searchParams = useSearchParams();
  const { setAuthFromTokens } = useAuth();
  const router = useRouter();

  useEffect(() => {
    const access = searchParams.get("access_token");
    const refresh = searchParams.get("refresh_token");
    if (access && refresh) {
      setAuthFromTokens({ access_token: access, refresh_token: refresh }).then(
        () => router.replace("/")
      );
    } else {
      router.replace("/login?error=oauth");
    }
  }, [searchParams, setAuthFromTokens, router]);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <p className="text-zinc-500">Completing sign in...</p>
    </div>
  );
}

export default function AuthCallbackPage() {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <p className="text-zinc-500">Loading...</p>
        </div>
      }
    >
      <CallbackHandler />
    </Suspense>
  );
}
