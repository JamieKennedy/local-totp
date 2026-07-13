import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import {
  Copy,
  Edit3,
  Eye,
  KeyRound,
  Lock,
  Plus,
  Search,
  Settings as SettingsIcon,
  Star,
  Trash2,
} from "lucide-react";
import QRCode from "qrcode";
import { useEffect, useMemo, useState } from "react";
import { api } from "@/api/client";
import type { Credential, CredentialInput, CurrentCode, Group, SecretView } from "@/api/types";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Settings } from "@/features/settings/Settings";
import { CredentialForm } from "./CredentialForm";

const noGroups: Group[] = [];

export function Dashboard({ version, onLock }: { version: string; onLock: () => Promise<void> }) {
  const queryClient = useQueryClient();
  const credentialsQuery = useQuery({ queryKey: ["credentials"], queryFn: api.credentials });
  const groupsQuery = useQuery({ queryKey: ["groups"], queryFn: api.groups });
  const codesQuery = useQuery({ queryKey: ["codes"], queryFn: api.codes });
  const [now, setNow] = useState(Date.now());
  const [search, setSearch] = useState("");
  const [groupFilter, setGroupFilter] = useState("all");
  const [editing, setEditing] = useState<Credential | "new">();
  const [revealed, setRevealed] = useState<{
    credential: Credential;
    secret: SecretView;
    qr: string;
  }>();
  const [sorting, setSorting] = useState<SortingState>([
    { id: "favorite", desc: true },
    { id: "name", desc: false },
  ]);
  const groups = groupsQuery.data ?? noGroups;

  useEffect(() => {
    const interval = window.setInterval(() => setNow(Date.now()), 250);
    return () => window.clearInterval(interval);
  }, []);
  useEffect(() => {
    const expiries = (codesQuery.data?.codes ?? []).map((code) => Date.parse(code.validUntil));
    if (expiries.length === 0) return;
    const delay = Math.max(100, Math.min(...expiries) - Date.now() + 100);
    const timeout = window.setTimeout(
      () => void queryClient.invalidateQueries({ queryKey: ["codes"] }),
      delay,
    );
    return () => window.clearTimeout(timeout);
  }, [codesQuery.data, queryClient]);

  const refreshCredentials = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["credentials"] }),
      queryClient.invalidateQueries({ queryKey: ["codes"] }),
    ]);
  };
  const saveMutation = useMutation({
    mutationFn: ({
      credential,
      input,
    }: {
      credential: Credential | undefined;
      input: CredentialInput;
    }) =>
      credential === undefined
        ? api.createCredential(input)
        : api.updateCredential(credential.id, input),
    onSuccess: async () => {
      setEditing(undefined);
      await refreshCredentials();
    },
  });
  const deleteMutation = useMutation({
    mutationFn: api.deleteCredential,
    onSuccess: refreshCredentials,
  });
  const favoriteMutation = useMutation({
    mutationFn: (credential: Credential) =>
      api.updateCredential(credential.id, {
        source: "manual",
        issuer: credential.issuer,
        account: credential.account,
        algorithm: credential.algorithm,
        digits: credential.digits,
        period: credential.period,
        favorite: !credential.favorite,
        ...(credential.groupId === undefined ? {} : { groupId: credential.groupId }),
        tags: credential.tags,
        notes: credential.notes,
      }),
    onSuccess: refreshCredentials,
  });
  const reveal = async (credential: Credential) => {
    const secret = await api.revealSecret(credential.id);
    setRevealed({
      credential,
      secret,
      qr: await QRCode.toDataURL(secret.uri, { width: 280, margin: 2 }),
    });
  };

  const codeMap = useMemo(
    () => new Map((codesQuery.data?.codes ?? []).map((code) => [code.credentialId, code])),
    [codesQuery.data],
  );
  const groupMap = useMemo(() => new Map(groups.map((group) => [group.id, group])), [groups]);
  const filtered = useMemo(
    () =>
      (credentialsQuery.data ?? []).filter((credential) => {
        const haystack =
          `${credential.issuer} ${credential.account} ${credential.tags.join(" ")} ${credential.notes}`.toLowerCase();
        return (
          haystack.includes(search.toLowerCase()) &&
          (groupFilter === "all" || credential.groupId === groupFilter)
        );
      }),
    [credentialsQuery.data, groupFilter, search],
  );

  const columns = useMemo<ColumnDef<Credential>[]>(
    () => [
      {
        id: "favorite",
        accessorFn: (row) => (row.favorite ? 1 : 0),
        header: "Pinned",
        cell: ({ row }) => (
          <Button
            size="icon"
            variant="ghost"
            aria-label={row.original.favorite ? "Unpin credential" : "Pin credential"}
            onClick={() => favoriteMutation.mutate(row.original)}
          >
            <Star
              className={
                row.original.favorite ? "fill-primary text-primary" : "text-muted-foreground"
              }
            />
          </Button>
        ),
      },
      {
        id: "name",
        accessorFn: (row) => `${row.issuer}:${row.account}`,
        header: "Credential",
        cell: ({ row }) => (
          <div>
            <p className="font-semibold">{row.original.issuer || "No issuer"}</p>
            <p className="text-muted-foreground text-xs">{row.original.account}</p>
          </div>
        ),
      },
      {
        id: "code",
        header: "Current code",
        cell: ({ row }) => <CodeCell code={codeMap.get(row.original.id)} now={now} />,
      },
      {
        id: "group",
        accessorFn: (row) =>
          row.groupId === undefined ? "" : (groupMap.get(row.groupId)?.name ?? ""),
        header: "Group",
        cell: ({ row }) =>
          row.original.groupId === undefined ? (
            <span className="text-muted-foreground">—</span>
          ) : (
            <Badge variant="outline">{groupMap.get(row.original.groupId)?.name ?? "Unknown"}</Badge>
          ),
      },
      {
        id: "tags",
        header: "Tags",
        cell: ({ row }) => (
          <div className="flex max-w-56 flex-wrap gap-1">
            {row.original.tags.length === 0 ? (
              <span className="text-muted-foreground">—</span>
            ) : (
              row.original.tags.map((tag) => (
                <Badge key={tag} variant="secondary">
                  {tag}
                </Badge>
              ))
            )}
          </div>
        ),
      },
      {
        id: "actions",
        header: "",
        cell: ({ row }) => (
          <div className="flex justify-end gap-1">
            <Button
              size="icon"
              variant="ghost"
              aria-label="Reveal seed"
              onClick={() => void reveal(row.original)}
            >
              <Eye />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              aria-label="Edit credential"
              onClick={() => setEditing(row.original)}
            >
              <Edit3 />
            </Button>
            <Button
              size="icon"
              variant="ghost"
              aria-label="Delete credential"
              className="text-destructive"
              onClick={() => {
                if (
                  window.confirm(
                    `Permanently delete ${row.original.issuer}: ${row.original.account}?`,
                  )
                )
                  deleteMutation.mutate(row.original.id);
              }}
            >
              <Trash2 />
            </Button>
          </div>
        ),
      },
    ],
    [codeMap, deleteMutation, favoriteMutation, groupMap, now],
  );
  const table = useReactTable({
    data: filtered,
    columns,
    state: { sorting },
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
  });
  const drift =
    codesQuery.data === undefined
      ? 0
      : Math.abs(Date.parse(codesQuery.data.serverTime) - Date.now());

  return (
    <main className="mx-auto min-h-svh w-full max-w-[1500px] p-4 sm:p-6">
      <header className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-3">
          <div className="bg-primary text-primary-foreground grid size-10 place-items-center rounded-xl font-black tracking-tighter">
            LT
          </div>
          <div>
            <h1 className="text-xl font-bold tracking-tight">Local TOTP</h1>
            <p className="text-muted-foreground text-xs">Test credential workbench · {version}</p>
          </div>
        </div>
        <Button variant="outline" onClick={() => void api.lock().then(onLock)}>
          <Lock />
          Lock vault
        </Button>
      </header>
      {drift > 5_000 && (
        <div className="mb-4 rounded-lg border border-amber-500/40 bg-amber-500/10 p-3 text-sm text-amber-600">
          Browser and server clocks differ by more than five seconds. Codes follow server time.
        </div>
      )}
      <Tabs defaultValue="workbench">
        <TabsList>
          <TabsTrigger value="workbench">
            <KeyRound className="mr-2 size-4" />
            Workbench
          </TabsTrigger>
          <TabsTrigger value="settings">
            <SettingsIcon className="mr-2 size-4" />
            Settings
          </TabsTrigger>
        </TabsList>
        <TabsContent value="workbench">
          <Card>
            <CardContent className="p-0">
              <div className="flex flex-col gap-3 border-b p-4 md:flex-row md:items-center">
                <div className="relative flex-1">
                  <Search className="text-muted-foreground absolute top-1/2 left-3 size-4 -translate-y-1/2" />
                  <Input
                    className="pl-9"
                    value={search}
                    onChange={(event) => setSearch(event.target.value)}
                    placeholder="Search issuer, account, tags, or notes"
                  />
                </div>
                <Select value={groupFilter} onValueChange={setGroupFilter}>
                  <SelectTrigger className="w-full md:w-48">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">All groups</SelectItem>
                    {groups.map((group) => (
                      <SelectItem key={group.id} value={group.id}>
                        {group.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button onClick={() => setEditing("new")}>
                  <Plus />
                  Add credential
                </Button>
              </div>
              {credentialsQuery.isPending ? (
                <div className="text-muted-foreground p-12 text-center">Loading credentials…</div>
              ) : filtered.length === 0 ? (
                <div className="p-12 text-center">
                  <KeyRound className="text-muted-foreground mx-auto mb-3 size-8" />
                  <h2 className="font-semibold">No matching credentials</h2>
                  <p className="text-muted-foreground mt-1 text-sm">
                    Add a manual seed, URI, QR image, or generated test identity.
                  </p>
                </div>
              ) : (
                <>
                  <div className="hidden md:block">
                    <Table>
                      <TableHeader>
                        {table.getHeaderGroups().map((headerGroup) => (
                          <TableRow key={headerGroup.id}>
                            {headerGroup.headers.map((header) => (
                              <TableHead key={header.id}>
                                {header.isPlaceholder ? null : (
                                  <button
                                    className="text-left"
                                    onClick={header.column.getToggleSortingHandler()}
                                  >
                                    {flexRender(
                                      header.column.columnDef.header,
                                      header.getContext(),
                                    )}
                                  </button>
                                )}
                              </TableHead>
                            ))}
                          </TableRow>
                        ))}
                      </TableHeader>
                      <TableBody>
                        {table.getRowModel().rows.map((row) => (
                          <TableRow key={row.id}>
                            {row.getVisibleCells().map((cell) => (
                              <TableCell key={cell.id}>
                                {flexRender(cell.column.columnDef.cell, cell.getContext())}
                              </TableCell>
                            ))}
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </div>
                  <div className="grid gap-3 p-3 md:hidden">
                    {table.getRowModel().rows.map((row) => (
                      <MobileCredential
                        key={row.id}
                        credential={row.original}
                        code={codeMap.get(row.original.id)}
                        now={now}
                        group={
                          row.original.groupId === undefined
                            ? undefined
                            : groupMap.get(row.original.groupId)
                        }
                        onEdit={() => setEditing(row.original)}
                        onReveal={() => void reveal(row.original)}
                        onDelete={() => {
                          if (
                            window.confirm(
                              `Permanently delete ${row.original.issuer}: ${row.original.account}?`,
                            )
                          )
                            deleteMutation.mutate(row.original.id);
                        }}
                      />
                    ))}
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>
        <TabsContent value="settings">
          <Settings onDataChange={refreshCredentials} />
        </TabsContent>
      </Tabs>
      {editing !== undefined && (
        <CredentialForm
          credential={editing === "new" ? undefined : editing}
          groups={groups}
          onCancel={() => setEditing(undefined)}
          onSubmit={async (input) => {
            await saveMutation.mutateAsync({
              credential: editing === "new" ? undefined : editing,
              input,
            });
          }}
        />
      )}
      {revealed !== undefined && (
        <SecretDialog value={revealed} onClose={() => setRevealed(undefined)} />
      )}
    </main>
  );
}

function CodeCell({ code, now }: { code: CurrentCode | undefined; now: number }) {
  if (code === undefined) return <span className="text-muted-foreground">Unavailable</span>;
  const remaining = Math.max(0, Math.ceil((Date.parse(code.validUntil) - now) / 1000));
  return (
    <div className="flex items-center gap-2">
      <button
        className="hover:text-primary font-mono text-xl font-bold tracking-[0.16em]"
        onClick={() => void navigator.clipboard.writeText(code.code)}
        title="Copy code"
      >
        {code.code}
      </button>
      <span className="text-muted-foreground min-w-7 text-xs tabular-nums">{remaining}s</span>
    </div>
  );
}

function MobileCredential({
  credential,
  code,
  now,
  group,
  onEdit,
  onReveal,
  onDelete,
}: {
  credential: Credential;
  code: CurrentCode | undefined;
  now: number;
  group: Group | undefined;
  onEdit: () => void;
  onReveal: () => void;
  onDelete: () => void;
}) {
  return (
    <div className="rounded-lg border p-4">
      <div className="flex justify-between gap-3">
        <div>
          <p className="font-semibold">{credential.issuer || "No issuer"}</p>
          <p className="text-muted-foreground text-xs">{credential.account}</p>
        </div>
        {group !== undefined && <Badge variant="outline">{group.name}</Badge>}
      </div>
      <div className="my-4">
        <CodeCell code={code} now={now} />
      </div>
      <div className="flex items-center justify-between">
        <div className="flex gap-1">
          {credential.tags.slice(0, 2).map((tag) => (
            <Badge key={tag} variant="secondary">
              {tag}
            </Badge>
          ))}
        </div>
        <div className="flex gap-1">
          <Button size="icon" variant="ghost" onClick={onReveal}>
            <Eye />
          </Button>
          <Button size="icon" variant="ghost" onClick={onEdit}>
            <Edit3 />
          </Button>
          <Button size="icon" variant="ghost" className="text-destructive" onClick={onDelete}>
            <Trash2 />
          </Button>
        </div>
      </div>
    </div>
  );
}

function SecretDialog({
  value,
  onClose,
}: {
  value: { credential: Credential; secret: SecretView; qr: string };
  onClose: () => void;
}) {
  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open) onClose();
      }}
    >
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>
            Seed for {value.credential.issuer}: {value.credential.account}
          </DialogTitle>
          <DialogDescription>
            Anyone with this seed can generate future codes. Use it only for test setup.
          </DialogDescription>
        </DialogHeader>
        <div className="grid justify-items-center gap-4 py-4">
          <img
            className="rounded-xl bg-white p-3"
            src={value.qr}
            alt="TOTP enrollment QR code"
            width={240}
            height={240}
          />
          <code className="bg-muted w-full rounded-lg border p-3 text-sm break-all">
            {value.secret.secret}
          </code>
          <Button
            variant="outline"
            className="w-full"
            onClick={() => void navigator.clipboard.writeText(value.secret.secret)}
          >
            <Copy />
            Copy seed
          </Button>
          <Button
            variant="outline"
            className="w-full"
            onClick={() => void navigator.clipboard.writeText(value.secret.uri)}
          >
            <Copy />
            Copy otpauth URI
          </Button>
        </div>
        <DialogFooter>
          <Button onClick={onClose}>Done</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
