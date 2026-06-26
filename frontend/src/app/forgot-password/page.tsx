"use client";

import { useState } from "react";
import Link from "next/link";
import { api } from "@/lib/api";
import { AuthLinkAlert } from "@/components/auth-link-alert";
import { Alert, AuthShell, Button, FieldLabel, Input } from "@/components/ui";

export default function ForgotPasswordPage() {
  const [email, setEmail] = useState("");
  const [message, setMessage] = useState("");
  const [resetUrl, setResetUrl] = useState<string | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setMessage("");
    setResetUrl(null);
    setLoading(true);
    try {
      const data = await api<{ message: string; reset_url?: string }>(
        "/auth/forgot-password",
        {
          method: "POST",
          body: JSON.stringify({ email }),
        }
      );
      if (data.reset_url) {
        setMessage("Email is not configured. Use this link to reset your password:");
        setResetUrl(data.reset_url);
      } else {
        setMessage("If an account exists for that email, we sent a reset link.");
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Request failed");
    } finally {
      setLoading(false);
    }
  };

  return (
    <AuthShell
      title="Forgot password"
      subtitle="We'll email you a link to reset your password"
    >
      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <FieldLabel>Email</FieldLabel>
          <Input
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            required
          />
        </div>
        {message && <Alert variant="success">{message}</Alert>}
        {resetUrl && (
          <AuthLinkAlert
            url={resetUrl}
            description="Open this link to choose a new password (expires in 1 hour):"
          />
        )}
        {error && <Alert variant="error">{error}</Alert>}
        <Button type="submit" className="w-full" disabled={loading}>
          {loading ? "Sending..." : "Send reset link"}
        </Button>
      </form>
      <p className="mt-6 text-center text-sm text-muted">
        <Link href="/login" className="font-medium text-primary hover:underline">
          Back to sign in
        </Link>
      </p>
    </AuthShell>
  );
}
