import { createRootRoute, Outlet } from "@tanstack/react-router";
import { TanStackRouterDevtools } from "@tanstack/react-router-devtools";
import Header from "../components/Header";
import { EmailVerificationOnLoad } from "@/components/EmailVerificationOnLoad";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { AuthProvider } from "@/lib/auth";

export const Route = createRootRoute({
  component: RootComponent,
});

function RootComponent() {
  return (
    <AuthProvider>
      <TooltipProvider delayDuration={400}>
        <EmailVerificationOnLoad />
        <Header />
        <Outlet />
        <Toaster richColors position="top-center" />
        <TanStackRouterDevtools position="bottom-right" />
      </TooltipProvider>
    </AuthProvider>
  );
}
