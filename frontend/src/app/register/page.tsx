"use client";

import { Suspense, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useAuth } from "@/providers/auth-provider";
import { getApiUrl } from "@/lib/api";
import { getRedirectPath, withRedirect } from "@/lib/redirect";
import { Alert, AuthShell, Button, FieldLabel, Input } from "@/components/ui";

export default function RegisterPage() {
  return (
    <Suspense
      fallback={
        <AuthShell title="Create account" subtitle="Get started in seconds">
          <p className="text-center text-sm text-muted">Loading...</p>
        </AuthShell>
      }
    >
      <RegisterForm />
    </Suspense>
  );
}

function RegisterForm() {
  const { register } = useAuth();
  const router = useRouter();
  const searchParams = useSearchParams();
  const redirect = getRedirectPath(searchParams.toString());
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await register(email, password, name);
      router.push(redirect ?? "/verify-email");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Registration failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell title="Create account" subtitle="Get started in seconds">
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <FieldLabel>Name</FieldLabel>
          <Input value={name} onChange={(e) => setName(e.target.value)} required />
        </div>
        <div>
          <FieldLabel>Email</FieldLabel>
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        <div>
          <FieldLabel>Password</FieldLabel>
          <Input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            minLength={8}
            required
          />
          <p className="mt-1 text-xs text-muted">At least 8 characters</p>
        </div>
        {error && <Alert variant="error">{error}</Alert>}
        <Button type="submit" className="w-full" disabled={loading}>
          {loading ? "Creating..." : "Create account"}
        </Button>
      </form>
      <div className="my-5 flex items-center gap-3">
        <div className="h-px flex-1 bg-border" />
        <span className="text-xs text-muted">or continue with</span>
        <div className="h-px flex-1 bg-border" />
      </div>
      <div className="flex flex-col gap-2">
        <a href={`${getApiUrl()}/api/v1/auth/google`}>
          <Button variant="secondary" className="w-full" type="button">
            Google
          </Button>
        </a>
        <a href={`${getApiUrl()}/api/v1/auth/github`}>
          <Button variant="secondary" className="w-full" type="button">
            GitHub
          </Button>
        </a>
      </div>
      <p className="mt-6 text-center text-sm text-muted">
        Already have an account?{" "}
        <Link href={withRedirect("/login", redirect)} className="font-medium text-primary hover:underline">
          Sign in
        </Link>
      </p>
    </AuthShell>
  );
}
