"use client";

import { AuthProvider } from "@/lib/auth";
import { Sidebar } from "@/components/sidebar";
import { ToastProvider } from "@/components/Toast";

export function ClientLayout({ children }: { children: React.ReactNode }) {
  return (
    <AuthProvider>
      <ToastProvider>
        <div className="flex h-screen">
          <Sidebar />
          <main className="flex-1 overflow-auto p-4 md:p-6">{children}</main>
        </div>
      </ToastProvider>
    </AuthProvider>
  );
}
