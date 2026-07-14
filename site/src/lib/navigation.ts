import { withBase } from "@/lib/paths";

export interface NavigationItem {
  href: string;
  label: string;
  description: string;
}

export const documentationNavigation: NavigationItem[] = [
  {
    href: withBase("docs/installation/"),
    label: "Installation",
    description: "Install a signed release and create the vault.",
  },
  {
    href: withBase("docs/deployment/"),
    label: "Deployment",
    description: "Run the published container safely on loopback.",
  },
  {
    href: withBase("docs/cli/"),
    label: "CLI",
    description: "Read credential metadata and codes from scripts.",
  },
  {
    href: withBase("docs/api/"),
    label: "API",
    description: "Use the versioned localhost HTTP interface.",
  },
];

export const githubURL = "https://github.com/JamieKennedy/local-totp";
export const releasesURL = `${githubURL}/releases`;
