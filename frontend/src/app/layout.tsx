import type { Metadata } from "next";
import { AppHeader } from "@/components/app-header";
import { AuthProvider } from "@/providers/auth-provider";
import { QueryProvider } from "@/providers/query-provider";
import { ToastProvider } from "@/providers/toast-provider";
import { getServerAppConfig } from "@/lib/runtime-config";
import "./globals.css";

export const metadata: Metadata = {
  title: "Trello Clone",
  description: "Workspace, board, and task management",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const config = getServerAppConfig();
  const configScript = `window.__APP_CONFIG__=${JSON.stringify(config)}`;

  return (
    <html lang="en" className="h-full">
      <head>
        <script dangerouslySetInnerHTML={{ __html: configScript }} />
      </head>
      <body className="flex min-h-full flex-col bg-background text-foreground">
        <QueryProvider>
          <ToastProvider>
            <AuthProvider>
              <AppHeader />
              <main className="flex-1">{children}</main>
            </AuthProvider>
          </ToastProvider>
        </QueryProvider>
      </body>
    </html>
  );
}
