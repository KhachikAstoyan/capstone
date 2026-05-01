import { useEffect, useState } from "react";
import { Link } from "@tanstack/react-router";
import { LogOut, Shield, User as UserIcon, BarChart3 } from "lucide-react";
import { useAuth } from "@/lib/auth";
import { canManageProblems, getMyPermissions } from "@/lib/permissions";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarImage, AvatarFallback } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import LoginModal from "./LoginModal";
import SignUpModal from "./SignUpModal";
import ThemeToggle from "./ThemeToggle";

function getInitials(displayName: string): string {
  return displayName
    .split(" ")
    .map((w) => w[0])
    .slice(0, 2)
    .join("")
    .toUpperCase();
}

export default function Header() {
  const { user, loading, logout } = useAuth();
  const [loginOpen, setLoginOpen] = useState(false);
  const [signUpOpen, setSignUpOpen] = useState(false);
  const [showAdminNav, setShowAdminNav] = useState(false);

  useEffect(() => {
    if (!user) {
      setShowAdminNav(false);
      return;
    }
    let cancelled = false;
    getMyPermissions()
      .then((perms) => {
        if (!cancelled) setShowAdminNav(canManageProblems(perms));
      })
      .catch(() => {
        if (!cancelled) setShowAdminNav(false);
      });
    return () => {
      cancelled = true;
    };
  }, [user]);

  return (
    <>
      <header className="sticky top-0 z-50 border-b bg-background/80 backdrop-blur-lg">
        <div className="flex h-14 items-center gap-4 px-4 sm:px-6">
          {/* Logo */}
          <Link
            search={{
              q: undefined,
              difficulty: undefined,
              tags: undefined,
              page: 1,
            }}
            to="/"
            className="flex shrink-0 items-center gap-2 no-underline"
          >
            <span className="rounded bg-primary px-1.5 py-0.5 text-xs font-bold tracking-widest text-primary-foreground">
              CP
            </span>
            <span className="font-bold text-foreground">Capstone</span>
          </Link>

          {/* Nav */}
          <nav className="hidden items-center gap-1 sm:flex">
            <Button variant="ghost" size="sm" asChild>
              <Link
                to="/"
                search={{ q: undefined, difficulty: undefined, tags: undefined, page: 1 }}
                className="nav-link text-sm no-underline"
              >
                Problems
              </Link>
            </Button>
          </nav>

          {/* User area */}
          <div className="ml-auto flex items-center gap-1">
            {showAdminNav && (
              <Button variant="ghost" size="sm" asChild>
                <Link
                  to="/admin/problems"
                  search={{ page: 1, q: undefined, visibility: undefined }}
                  className="flex items-center gap-1.5 no-underline"
                >
                  <Shield className="h-4 w-4" />
                  Admin
                </Link>
              </Button>
            )}
            <ThemeToggle />
            {loading ? (
              <div className="h-8 w-8 animate-pulse rounded-full bg-muted" />
            ) : user ? (
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <button className="rounded-full outline-none ring-offset-background focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2">
                    <Avatar className="h-8 w-8 cursor-pointer">
                      <AvatarImage
                        src={user.avatar_url}
                        alt={user.display_name}
                      />
                      <AvatarFallback className="text-xs">
                        {getInitials(user.display_name || user.handle)}
                      </AvatarFallback>
                    </Avatar>
                  </button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-48">
                  <DropdownMenuLabel className="font-normal text-muted-foreground">
                    @{user.handle}
                  </DropdownMenuLabel>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem asChild>
                    <Link
                      to="/dashboard"
                      className="flex cursor-pointer items-center gap-2"
                    >
                      <BarChart3 className="h-4 w-4" />
                      Dashboard
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuItem asChild>
                    <Link
                      to="/users/$userRef"
                      params={{ userRef: user.handle }}
                      className="flex cursor-pointer items-center gap-2"
                    >
                      <UserIcon className="h-4 w-4" />
                      Profile
                    </Link>
                  </DropdownMenuItem>
                  <DropdownMenuSeparator />
                  <DropdownMenuItem
                    className="flex cursor-pointer items-center gap-2 text-destructive focus:text-destructive"
                    onSelect={() => logout()}
                  >
                    <LogOut className="h-4 w-4" />
                    Log out
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            ) : (
              <>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setLoginOpen(true)}
                >
                  Log in
                </Button>
                <Button size="sm" onClick={() => setSignUpOpen(true)}>
                  Sign up
                </Button>
              </>
            )}
          </div>
        </div>
      </header>

      <LoginModal open={loginOpen} onOpenChange={setLoginOpen} />
      <SignUpModal open={signUpOpen} onOpenChange={setSignUpOpen} />
    </>
  );
}
