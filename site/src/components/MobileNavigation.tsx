import { Menu } from "lucide-react";
import { GitHubLogo } from "@/components/GitHubLogo";
import { Button } from "@/components/ui/button";
import {
  Sheet,
  SheetClose,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from "@/components/ui/sheet";
import { documentationNavigation, githubURL } from "@/lib/navigation";
import { withBase } from "@/lib/paths";

interface MobileNavigationProps {
  currentPath: string;
}

export function MobileNavigation({ currentPath }: MobileNavigationProps) {
  return (
    <Sheet>
      <SheetTrigger asChild>
        <Button variant="ghost" size="icon" aria-label="Open navigation">
          <Menu aria-hidden="true" />
        </Button>
      </SheetTrigger>
      <SheetContent>
        <SheetHeader className="pr-10">
          <SheetTitle>Local TOTP</SheetTitle>
          <SheetDescription>
            Documentation for the localhost test-credential workbench.
          </SheetDescription>
        </SheetHeader>
        <nav
          className="mt-8 flex flex-1 flex-col gap-2"
          aria-label="Mobile navigation"
        >
          <SheetClose asChild>
            <a
              href={withBase()}
              className="hover:bg-accent focus-visible:ring-ring rounded-lg px-3 py-3 font-medium focus-visible:ring-2 focus-visible:outline-none"
              aria-current={currentPath === withBase() ? "page" : undefined}
            >
              Home
            </a>
          </SheetClose>
          {documentationNavigation.map((item) => (
            <SheetClose asChild key={item.href}>
              <a
                href={item.href}
                className="hover:bg-accent focus-visible:ring-ring rounded-lg px-3 py-3 focus-visible:ring-2 focus-visible:outline-none"
                aria-current={currentPath === item.href ? "page" : undefined}
              >
                <span className="block font-medium">{item.label}</span>
                <span className="text-muted-foreground mt-0.5 block text-xs leading-5">
                  {item.description}
                </span>
              </a>
            </SheetClose>
          ))}
        </nav>
        <a
          href={githubURL}
          className="hover:bg-accent focus-visible:ring-ring mt-6 flex items-center gap-2 rounded-lg border px-3 py-3 text-sm font-medium focus-visible:ring-2 focus-visible:outline-none"
        >
          <GitHubLogo className="size-4 shrink-0" />
          View on GitHub
        </a>
      </SheetContent>
    </Sheet>
  );
}
