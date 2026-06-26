"use client";

import Link from "next/link";
import { useAuth } from "@/providers/auth-provider";
import { Avatar, Badge, Button } from "./ui";
import { NotificationBell } from "./notification-bell";

export function AppHeader() {
  const { user, logout } = useAuth();

  return (
    <header className="sticky top-0 z-40 border-b border-border bg-surface/95 px-4 py-2.5 shadow-sm backdrop-blur-sm sm:px-6">
      <div className="mx-auto flex max-w-7xl items-center justify-between">
        <Link href="/" className="flex items-center gap-2.5">
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-primary text-sm font-bold text-white">
            T
          </span>
          <span className="text-base font-semibold text-foreground">Trello Clone</span>
        </Link>
        {user ? (
          <div className="flex items-center gap-2 sm:gap-3">
            <NotificationBell />
            <div className="hidden items-center gap-2 sm:flex">
              <Avatar name={user.name} size="sm" />
              <span className="text-sm text-muted">{user.name}</span>
              {user.is_admin && <Badge>Admin</Badge>}
            </div>
            <Button variant="ghost" size="sm" onClick={logout}>
              Log out
            </Button>
          </div>
        ) : (
          <div className="flex items-center gap-2">
            <Link href="/login">
              <Button variant="ghost" size="sm">
                Sign in
              </Button>
            </Link>
            <Link href="/register">
              <Button size="sm">Register</Button>
            </Link>
          </div>
        )}
      </div>
    </header>
  );
}
