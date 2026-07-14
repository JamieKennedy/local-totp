import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

const linux = `gh auth login
gh release download "v1.0.0" \\
  --repo JamieKennedy/local-totp \\
  --pattern "local-totp_1.0.0_linux_<ARCH>.tar.gz" \\
  --pattern "SHA256SUMS"
grep "local-totp_1.0.0_linux_<ARCH>.tar.gz" SHA256SUMS | sha256sum --check
tar -xzf "local-totp_1.0.0_linux_<ARCH>.tar.gz"
install "local-totp_1.0.0_linux_<ARCH>/local-totp" "<INSTALL_DIRECTORY>/local-totp"`;

const macOS = `gh auth login
gh release download "v1.0.0" \\
  --repo JamieKennedy/local-totp \\
  --pattern "local-totp_1.0.0_darwin_<ARCH>.tar.gz" \\
  --pattern "SHA256SUMS"
grep "local-totp_1.0.0_darwin_<ARCH>.tar.gz" SHA256SUMS | shasum -a 256 --check
tar -xzf "local-totp_1.0.0_darwin_<ARCH>.tar.gz"
install "local-totp_1.0.0_darwin_<ARCH>/local-totp" "<INSTALL_DIRECTORY>/local-totp"`;

const windows = `gh auth login
gh release download "v1.0.0" --repo JamieKennedy/local-totp --pattern "local-totp_1.0.0_windows_<ARCH>.zip" --pattern "SHA256SUMS"
$expected = (Select-String "local-totp_1.0.0_windows_<ARCH>.zip" SHA256SUMS).Line.Split()[0]
$actual = (Get-FileHash "local-totp_1.0.0_windows_<ARCH>.zip" -Algorithm SHA256).Hash.ToLower()
if ($actual -ne $expected) { throw "Checksum mismatch" }
Expand-Archive "local-totp_1.0.0_windows_<ARCH>.zip" -DestinationPath .
Copy-Item "local-totp_1.0.0_windows_<ARCH>/local-totp.exe" "<INSTALL_DIRECTORY>/local-totp.exe"`;

function CodePanel({ children }: { children: string }) {
  return (
    <pre className="not-prose bg-code text-code-foreground relative overflow-x-auto rounded-xl border p-5 text-sm leading-6">
      <code>{children}</code>
    </pre>
  );
}

export function InstallTabs() {
  return (
    <Tabs defaultValue="linux" className="not-prose my-6">
      <TabsList aria-label="Installation platform">
        <TabsTrigger value="linux">Linux</TabsTrigger>
        <TabsTrigger value="macos">macOS</TabsTrigger>
        <TabsTrigger value="windows">Windows</TabsTrigger>
      </TabsList>
      <TabsContent value="linux">
        <CodePanel>{linux}</CodePanel>
      </TabsContent>
      <TabsContent value="macos">
        <CodePanel>{macOS}</CodePanel>
      </TabsContent>
      <TabsContent value="windows">
        <CodePanel>{windows}</CodePanel>
      </TabsContent>
    </Tabs>
  );
}
