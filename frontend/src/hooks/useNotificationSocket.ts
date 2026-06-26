"use client";

import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { getAccessToken, getWsUrl } from "@/lib/api";
import type { Notification } from "@/lib/types";
import { useToast } from "@/providers/toast-provider";

export function useNotificationSocket(enabled: boolean) {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const toastRef = useRef(toast);
  toastRef.current = toast;

  useEffect(() => {
    if (!enabled) return;

    const token = getAccessToken();
    if (!token) return;

    let closed = false;
    let ws: WebSocket | null = null;
    let retry: ReturnType<typeof setTimeout> | null = null;

    const connect = () => {
      if (closed) return;
      ws = new WebSocket(
        `${getWsUrl()}/ws/notifications?token=${encodeURIComponent(token)}`
      );

      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.type === "notification.created") {
            queryClient.invalidateQueries({ queryKey: ["notifications"] });
            const n = data.payload as Notification | undefined;
            if (n) {
              toastRef.current({
                title: n.title,
                description: n.body,
                variant: "info",
              });
            }
          }
        } catch {
          // ignore malformed frames
        }
      };

      ws.onclose = () => {
        if (!closed) {
          retry = setTimeout(connect, 5000);
        }
      };
    };

    connect();

    return () => {
      closed = true;
      if (retry) clearTimeout(retry);
      ws?.close();
    };
  }, [enabled, queryClient]);
}
